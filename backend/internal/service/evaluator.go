package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"flagforge/internal/dto"
	"flagforge/internal/eval"
	"flagforge/internal/model"
	"flagforge/internal/repo"
	"gorm.io/gorm"
)

type EvaluatorService struct{ db *gorm.DB }

func NewEvaluatorService(db *gorm.DB) *EvaluatorService { return &EvaluatorService{db: db} }

// Normalize only JSON scalars and each in-list element. Nested attributes remain
// intact: the engine compares top-level values and enforces strict JSON types.
func normalizedAttributes(attrs map[string]any) map[string]any {
	out := make(map[string]any, len(attrs))
	for key, value := range attrs {
		out[key] = eval.Normalize(value)
	}
	return out
}
func normalizeExpected(value any) any {
	if list, ok := value.([]any); ok {
		out := make([]any, len(list))
		for i, v := range list {
			out[i] = eval.Normalize(v)
		}
		return out
	}
	return eval.Normalize(value)
}
func (s *EvaluatorService) load(ctx context.Context, key, environment string) (*model.Flag, []eval.Rule, error) {
	flag, err := repo.NewFlagRepo(s.db).GetByKeyEnv(ctx, key, environment)
	if err != nil {
		return nil, nil, flagError(err, key, environment)
	}
	rules := make([]eval.Rule, 0)
	if !flag.Enabled {
		return flag, rules, nil
	}
	stored, err := repo.NewRuleRepo(s.db).ListByFlagOrdered(ctx, flag.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range stored {
		var expected any
		if err = json.Unmarshal(r.ExpectedValue, &expected); err != nil {
			return nil, nil, fmt.Errorf("decode rule %s expected value: %w", r.ID, err)
		}
		rules = append(rules, eval.Rule{ID: r.ID, Priority: r.Priority, Attribute: r.Attribute, Operator: r.Operator, Expected: normalizeExpected(expected), RollValue: r.RollValue})
	}
	return flag, rules, nil
}
func evaluationResponse(result eval.Result) dto.Evaluation {
	out := dto.Evaluation{Value: result.Value, Reason: dto.Reason{Type: result.ReasonType}, Trace: make([]dto.TraceStep, 0, len(result.Trace))}
	if rule := result.HitRule; rule != nil {
		expected, _ := json.Marshal(rule.Expected)
		out.Reason = dto.Reason{Type: result.ReasonType, RuleID: rule.ID, Priority: rule.Priority, Summary: fmt.Sprintf("%s %s %s → %t", rule.Attribute, rule.Operator, expected, rule.RollValue)}
	}
	for _, step := range result.Trace {
		out.Trace = append(out.Trace, dto.TraceStep{RuleID: step.RuleID, Priority: step.Priority, Attribute: step.Attribute, Operator: step.Operator, Matched: step.Matched, Cause: step.Cause})
	}
	return out
}
func runEvaluation(flag *model.Flag, rules []eval.Rule, attrs map[string]any) eval.Result {
	return eval.Evaluate(eval.Input{Enabled: flag.Enabled, DefaultValue: flag.DefaultValue, Rules: rules, UserAttrs: normalizedAttributes(attrs)})
}
func (s *EvaluatorService) Evaluate(ctx context.Context, r dto.EvaluateRequest) (*dto.EvaluateResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	flag, rules, err := s.load(ctx, r.FlagKey, r.Environment)
	if err != nil {
		return nil, err
	}
	return &dto.EvaluateResponse{FlagKey: flag.Key, Environment: flag.Environment, Enabled: flag.Enabled, Evaluation: evaluationResponse(runEvaluation(flag, rules, r.UserAttributes)), EvaluatedAt: time.Now().UTC()}, nil
}
func (s *EvaluatorService) Batch(ctx context.Context, r dto.BatchRequest) (*dto.BatchResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	flag, rules, err := s.load(ctx, r.FlagKey, r.Environment)
	if err != nil {
		return nil, err
	}
	out := &dto.BatchResponse{Results: make([]dto.BatchResult, 0, len(r.UserAttributesList)), Summary: dto.BatchSummary{Total: len(r.UserAttributesList), HitRules: make([]dto.HitRuleCount, 0)}}
	hits := map[string]dto.HitRuleCount{}
	for i, attrs := range r.UserAttributesList {
		result := runEvaluation(flag, rules, attrs)
		out.Results = append(out.Results, dto.BatchResult{Index: i, Evaluation: evaluationResponse(result)})
		if result.Value {
			out.Summary.TrueCount++
		} else {
			out.Summary.FalseCount++
		}
		switch result.ReasonType {
		case "default":
			out.Summary.DefaultCount++
		case "rule_hit":
			rule := result.HitRule
			hit := hits[rule.ID]
			hit.RuleID = rule.ID
			hit.Priority = rule.Priority
			hit.Attribute = rule.Attribute
			hit.Count++
			hits[rule.ID] = hit
		}
	}
	for _, hit := range hits {
		out.Summary.HitRules = append(out.Summary.HitRules, hit)
	}
	sort.Slice(out.Summary.HitRules, func(i, j int) bool {
		a, b := out.Summary.HitRules[i], out.Summary.HitRules[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.RuleID < b.RuleID
	})
	return out, nil
}
