package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"flagforge/internal/apperror"
	"flagforge/internal/dto"
	"flagforge/internal/model"
	"flagforge/internal/repo"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RuleService struct{ db *gorm.DB }

func NewRuleService(db *gorm.DB) *RuleService { return &RuleService{db: db} }
func ruleUTC(r *model.Rule) *model.Rule {
	r.CreatedAt = r.CreatedAt.UTC()
	r.UpdatedAt = r.UpdatedAt.UTC()
	return r
}
func ruleError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &apperror.Error{Code: apperror.RuleNotFound, Message: "rule not found for this flag"}
	}
	var mapped *repo.Error
	if errors.As(err, &mapped) && mapped.Code == repo.DuplicatePriority {
		return &apperror.Error{Code: apperror.DuplicatePriority, Message: "priority already exists for this flag", Cause: err}
	}
	return err
}
func ruleSummary(r *model.Rule) map[string]any {
	return map[string]any{"priority": r.Priority, "attribute": r.Attribute, "operator": r.Operator, "expected_value": r.ExpectedValue, "roll_value": r.RollValue}
}
func (s *RuleService) Create(ctx context.Context, flagID string, r dto.CreateRule) (*model.Rule, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	rule := &model.Rule{FlagID: flagID, Priority: *r.Priority, Attribute: *r.Attribute, Operator: *r.Operator, ExpectedValue: datatypes.JSON(r.ExpectedValue), RollValue: *r.RollValue}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := repo.NewFlagRepo(tx).GetByID(ctx, flagID); err != nil {
			return flagError(err, "", "")
		}
		if err := repo.NewRuleRepo(tx).Create(ctx, rule); err != nil {
			return err
		}
		return WriteHistory(ctx, tx, flagID, "rule_created", map[string]any{"rule": ruleSummary(rule)})
	})
	if err != nil {
		return nil, ruleError(err)
	}
	return ruleUTC(rule), nil
}
func (s *RuleService) Update(ctx context.Context, flagID, id string, r dto.UpdateRule) (*model.Rule, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	var rule *model.Rule
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rules := repo.NewRuleRepo(tx)
		var err error
		rule, err = rules.GetByID(ctx, flagID, id)
		if err != nil {
			return err
		}
		next := *rule
		if r.Priority != nil {
			next.Priority = *r.Priority
		}
		if r.Attribute != nil {
			next.Attribute = *r.Attribute
		}
		if r.Operator != nil {
			next.Operator = *r.Operator
		}
		if r.RollValue != nil {
			next.RollValue = *r.RollValue
		}
		if len(r.ExpectedValue) > 0 {
			next.ExpectedValue = datatypes.JSON(r.ExpectedValue)
		}
		full := dto.CreateRule{Priority: &next.Priority, Attribute: &next.Attribute, Operator: &next.Operator, ExpectedValue: json.RawMessage(next.ExpectedValue), RollValue: &next.RollValue}
		if err = full.Validate(); err != nil {
			return err
		}
		fields := map[string]any{}
		changes := map[string]Change{}
		before, after := ruleSummary(rule), ruleSummary(&next)
		for _, field := range []string{"priority", "attribute", "operator", "roll_value"} {
			if !reflect.DeepEqual(before[field], after[field]) {
				fields[field] = after[field]
				changes[field] = Change{Old: before[field], New: after[field]}
			}
		}
		var oldExpected, newExpected any
		if err = json.Unmarshal(rule.ExpectedValue, &oldExpected); err != nil {
			return fmt.Errorf("decode stored expected value: %w", err)
		}
		if err = json.Unmarshal(next.ExpectedValue, &newExpected); err != nil {
			return err
		}
		if !reflect.DeepEqual(oldExpected, newExpected) {
			fields["expected_value"] = next.ExpectedValue
			changes["expected_value"] = Change{Old: oldExpected, New: newExpected}
		}
		if len(fields) == 0 {
			return nil
		}
		if err = rules.Update(ctx, flagID, id, fields); err != nil {
			return err
		}
		if err = WriteHistory(ctx, tx, flagID, "rule_updated", map[string]any{"rule_id": id, "priority": next.Priority, "changes": changes}); err != nil {
			return err
		}
		rule, err = rules.GetByID(ctx, flagID, id)
		return err
	})
	if err != nil {
		return nil, ruleError(err)
	}
	return ruleUTC(rule), nil
}
func (s *RuleService) Delete(ctx context.Context, flagID, id string) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rules := repo.NewRuleRepo(tx)
		rule, err := rules.GetByID(ctx, flagID, id)
		if err != nil {
			return err
		}
		if err = rules.Delete(ctx, flagID, id); err != nil {
			return err
		}
		return WriteHistory(ctx, tx, flagID, "rule_deleted", map[string]any{"rule": ruleSummary(rule)})
	})
	return ruleError(err)
}
