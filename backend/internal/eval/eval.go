package eval

import (
	"reflect"
	"sort"
)

type Rule struct {
	ID        string
	Priority  int
	Attribute string
	Operator  string
	Expected  any
	RollValue bool
}

type Input struct {
	Enabled      bool
	DefaultValue bool
	Rules        []Rule
	UserAttrs    map[string]any
}

type TraceStep struct {
	RuleID    string
	Priority  int
	Attribute string
	Operator  string
	Matched   bool
	Cause     string
}

type Result struct {
	Value      bool
	ReasonType string
	HitRule    *Rule
	Trace      []TraceStep
}

// Evaluate applies the first matching rule in ascending priority order.
// It only reads the supplied input and returns the trace of evaluated rules.
func Evaluate(in Input) Result {
	if !in.Enabled {
		return Result{
			ReasonType: "disabled",
			Trace:      []TraceStep{{Cause: "short_circuited_by_disabled_flag"}},
		}
	}

	result := Result{
		Value:      in.DefaultValue,
		ReasonType: "default",
		Trace:      make([]TraceStep, 0, len(in.Rules)),
	}
	rules := append([]Rule(nil), in.Rules...)
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})
	for _, rule := range rules {
		step := TraceStep{
			RuleID: rule.ID, Priority: rule.Priority,
			Attribute: rule.Attribute, Operator: rule.Operator,
		}
		actual, exists := in.UserAttrs[rule.Attribute]
		if !exists {
			step.Cause = "attribute_missing"
			result.Trace = append(result.Trace, step)
			continue
		}

		switch rule.Operator {
		case "equals":
			actual, expected := Normalize(actual), Normalize(rule.Expected)
			switch {
			case reflect.TypeOf(actual) != reflect.TypeOf(expected):
				step.Cause = "type_mismatch"
			case DeepEqualNormalized(actual, expected):
				step.Matched = true
			default:
				step.Cause = "value_not_equal"
			}
		case "in":
			step.Cause = "value_not_in_list"
			if expected, ok := rule.Expected.([]any); ok && isScalar(actual) {
				for _, candidate := range expected {
					// Invalid nested or null elements are skipped defensively.
					if isScalar(candidate) && DeepEqualNormalized(actual, candidate) {
						step.Matched = true
						break
					}
				}
			}
		default:
			// Validation rejects unknown operators; treat them as a miss here.
			step.Cause = "value_not_equal"
		}

		if step.Matched {
			step.Cause = "matched"
			result.Value = rule.RollValue
			result.ReasonType = "rule_hit"
			result.HitRule = &rule
			result.Trace = append(result.Trace, step)
			return result
		}
		result.Trace = append(result.Trace, step)
	}
	return result
}

func isScalar(v any) bool {
	switch Normalize(v).(type) {
	case string, bool, float64:
		return true
	default:
		return false
	}
}
