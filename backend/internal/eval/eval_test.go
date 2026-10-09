package eval

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluate(t *testing.T) {
	rule := func(id string, priority int, attribute, operator string, expected any, roll bool) Rule {
		return Rule{ID: id, Priority: priority, Attribute: attribute, Operator: operator, Expected: expected, RollValue: roll}
	}
	step := func(r Rule, matched bool, cause string) TraceStep {
		return TraceStep{RuleID: r.ID, Priority: r.Priority, Attribute: r.Attribute, Operator: r.Operator, Matched: matched, Cause: cause}
	}
	miss := func(value bool, steps ...TraceStep) Result {
		if steps == nil {
			steps = []TraceStep{}
		}
		return Result{Value: value, ReasonType: "default", Trace: steps}
	}
	hit := func(r Rule, steps ...TraceStep) Result {
		return Result{Value: r.RollValue, ReasonType: "rule_hit", HitRule: &r, Trace: steps}
	}
	plan := rule("plan", 1, "plan", "equals", "pro", true)
	level := rule("level", 2, "level", "equals", 42, true)
	vip := rule("vip", 3, "vip", "equals", true, true)
	country := rule("country", 1, "country", "in", []any{"CN", "US"}, false)
	mixed := rule("mixed", 1, "level", "in", []any{"pro", int64(42)}, true)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(`{"level":42}`), &decoded))

	tests := []struct {
		name string
		in   Input
		want Result
	}{
		{"4.1_disabled_short_circuit", Input{DefaultValue: true, Rules: []Rule{plan}, UserAttrs: map[string]any{"plan": "pro"}}, Result{ReasonType: "disabled", Trace: []TraceStep{{Cause: "short_circuited_by_disabled_flag"}}}},
		{"4.2_empty_rules_default_false", Input{Enabled: true, Rules: []Rule{}}, miss(false)},
		{"4.2_nil_rules_default_true", Input{Enabled: true, DefaultValue: true}, miss(true)},
		{"4.3_all_attributes_missing", Input{Enabled: true, Rules: []Rule{plan, level}, UserAttrs: map[string]any{}}, miss(false, step(plan, false, "attribute_missing"), step(level, false, "attribute_missing"))},
		{"4.4_second_hit_stops_before_three_remaining_rules", Input{Enabled: true, Rules: []Rule{plan, level, vip, rule("four", 4, "four", "equals", true, true), rule("five", 5, "five", "equals", true, true)}, UserAttrs: map[string]any{"plan": "team", "level": float64(42), "vip": true}}, hit(level, step(plan, false, "value_not_equal"), step(level, true, "matched"))},
		{"4.5_priority_3_1_2_sorted_without_mutation", Input{Enabled: true, Rules: []Rule{vip, plan, level}, UserAttrs: map[string]any{}}, miss(false, step(plan, false, "attribute_missing"), step(level, false, "attribute_missing"), step(vip, false, "attribute_missing"))},
		{"4.6_equals_string_hit", Input{Enabled: true, Rules: []Rule{plan}, UserAttrs: map[string]any{"plan": "pro"}}, hit(plan, step(plan, true, "matched"))},
		{"4.7_equals_json_float64_vs_int_42", Input{Enabled: true, Rules: []Rule{level}, UserAttrs: decoded}, hit(level, step(level, true, "matched"))},
		{"4.7_equals_int_vs_float64_42", Input{Enabled: true, Rules: []Rule{rule("number", 1, "level", "equals", float64(42), false)}, UserAttrs: map[string]any{"level": 42}}, hit(rule("number", 1, "level", "equals", float64(42), false), step(rule("number", 1, "level", "equals", float64(42), false), true, "matched"))},
		{"4.8_equals_bool_hit", Input{Enabled: true, Rules: []Rule{vip}, UserAttrs: map[string]any{"vip": true}}, hit(vip, step(vip, true, "matched"))},
		{"4.9_equals_string_1_vs_number_1", Input{Enabled: true, Rules: []Rule{rule("strict", 1, "value", "equals", 1, true)}, UserAttrs: map[string]any{"value": "1"}}, miss(false, step(rule("strict", 1, "value", "equals", 1, true), false, "type_mismatch"))},
		{"4.9_equals_bool_true_vs_string_true", Input{Enabled: true, Rules: []Rule{rule("strict", 1, "value", "equals", "true", true)}, UserAttrs: map[string]any{"value": true}}, miss(false, step(rule("strict", 1, "value", "equals", "true", true), false, "type_mismatch"))},
		{"4.10_equals_same_type_different_value", Input{Enabled: true, Rules: []Rule{plan}, UserAttrs: map[string]any{"plan": "team"}}, miss(false, step(plan, false, "value_not_equal"))},
		{"4.11_in_hit_returns_false_roll_value", Input{Enabled: true, DefaultValue: true, Rules: []Rule{country}, UserAttrs: map[string]any{"country": "US"}}, hit(country, step(country, true, "matched"))},
		{"4.11_in_miss", Input{Enabled: true, Rules: []Rule{country}, UserAttrs: map[string]any{"country": "DE"}}, miss(false, step(country, false, "value_not_in_list"))},
		{"4.11_in_single_element_hit", Input{Enabled: true, Rules: []Rule{rule("single", 1, "plan", "in", []any{"pro"}, true)}, UserAttrs: map[string]any{"plan": "pro"}}, hit(rule("single", 1, "plan", "in", []any{"pro"}, true), step(rule("single", 1, "plan", "in", []any{"pro"}, true), true, "matched"))},
		{"4.12_in_mixed_types_numeric_hit", Input{Enabled: true, Rules: []Rule{mixed}, UserAttrs: map[string]any{"level": float32(42)}}, hit(mixed, step(mixed, true, "matched"))},
		{"4.12_in_mixed_types_strict_string_42_miss", Input{Enabled: true, Rules: []Rule{mixed}, UserAttrs: map[string]any{"level": "42"}}, miss(false, step(mixed, false, "value_not_in_list"))},
		{"4.13_missing_attribute_continues_to_hit", Input{Enabled: true, Rules: []Rule{plan, level}, UserAttrs: map[string]any{"level": uint(42)}}, hit(level, step(plan, false, "attribute_missing"), step(level, true, "matched"))},
		{"4.13_nil_user_attrs", Input{Enabled: true, Rules: []Rule{plan}}, miss(false, step(plan, false, "attribute_missing"))},
		{"4.14_nested_object_equals_scalar", Input{Enabled: true, Rules: []Rule{plan}, UserAttrs: map[string]any{"plan": map[string]any{"tier": "pro"}}}, miss(false, step(plan, false, "type_mismatch"))},
		{"4.14_nested_object_in_scalar_list", Input{Enabled: true, Rules: []Rule{country}, UserAttrs: map[string]any{"country": map[string]any{"code": "CN"}}}, miss(false, step(country, false, "value_not_in_list"))},
		{"4.14_nested_array_in_scalar_list", Input{Enabled: true, Rules: []Rule{country}, UserAttrs: map[string]any{"country": []any{"CN"}}}, miss(false, step(country, false, "value_not_in_list"))},
		{"4.15_default_true_all_miss", Input{Enabled: true, DefaultValue: true, Rules: []Rule{plan}, UserAttrs: map[string]any{"plan": "team"}}, miss(true, step(plan, false, "value_not_equal"))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before, err := json.Marshal(tc.in)
			require.NoError(t, err)
			var got Result
			require.NotPanics(t, func() { got = Evaluate(tc.in) })
			assert.Equal(t, tc.want, got)
			assert.NotNil(t, got.Trace, "empty trace must encode as an array")
			traceJSON, err := json.Marshal(got.Trace)
			require.NoError(t, err)
			if len(tc.want.Trace) == 0 {
				assert.JSONEq(t, `[]`, string(traceJSON))
			}
			after, err := json.Marshal(tc.in)
			require.NoError(t, err)
			assert.Equal(t, before, after, "evaluation must preserve caller input")
		})
	}
}

func TestEvaluateDefensiveInputs(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		expected any
		actual   any
		cause    string
	}{
		{"4.11_in_scalar_expected", "in", "pro", "pro", "value_not_in_list"},
		{"4.11_in_typed_slice_expected", "in", []string{"pro"}, "pro", "value_not_in_list"},
		{"4.11_in_nil_expected", "in", nil, nil, "value_not_in_list"},
		{"4.11_in_empty_expected", "in", []any{}, "pro", "value_not_in_list"},
		{"4.14_in_nested_object_element", "in", []any{map[string]any{"tier": "pro"}}, map[string]any{"tier": "pro"}, "value_not_in_list"},
		{"4.14_in_nested_array_element", "in", []any{[]any{"pro"}}, []any{"pro"}, "value_not_in_list"},
		{"4.14_in_nil_element", "in", []any{nil}, nil, "value_not_in_list"},
		{"4.14_in_unsupported_element", "in", []any{struct{ Value string }{"pro"}}, struct{ Value string }{"pro"}, "value_not_in_list"},
		{"4.10_unknown_operator", "unknown", "pro", "pro", "value_not_equal"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Rule{ID: "defensive", Priority: 1, Attribute: "value", Operator: tc.operator, Expected: tc.expected, RollValue: true}
			var got Result
			require.NotPanics(t, func() {
				got = Evaluate(Input{Enabled: true, Rules: []Rule{r}, UserAttrs: map[string]any{"value": tc.actual}})
			})
			assert.Equal(t, Result{ReasonType: "default", Trace: []TraceStep{{RuleID: r.ID, Priority: r.Priority, Attribute: r.Attribute, Operator: r.Operator, Cause: tc.cause}}}, got)
		})
	}
}

func TestEvaluateStablePriority(t *testing.T) {
	rules := []Rule{
		{ID: "first", Priority: 1, Attribute: "plan", Operator: "equals", Expected: "pro", RollValue: false},
		{ID: "second", Priority: 1, Attribute: "plan", Operator: "equals", Expected: "pro", RollValue: true},
	}
	got := Evaluate(Input{Enabled: true, Rules: rules, UserAttrs: map[string]any{"plan": "pro"}})
	require.NotNil(t, got.HitRule)
	assert.Equal(t, "first", got.HitRule.ID)
	assert.False(t, got.Value)
	require.Len(t, got.Trace, 1)
	got.HitRule.ID = "changed"
	assert.Equal(t, "first", rules[0].ID, "returned rule must not alias the input rule struct")
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{"int", int(42), float64(42)}, {"int8", int8(42), float64(42)},
		{"int16", int16(42), float64(42)}, {"int32", int32(42), float64(42)},
		{"int64", int64(42), float64(42)}, {"uint", uint(42), float64(42)},
		{"uint8", uint8(42), float64(42)}, {"uint16", uint16(42), float64(42)},
		{"uint32", uint32(42), float64(42)}, {"uint64", uint64(42), float64(42)},
		{"uintptr", uintptr(42), float64(42)}, {"float32", float32(1.5), float64(1.5)},
		{"float64", float64(42), float64(42)}, {"negative", int64(-42), float64(-42)},
		{"string", "42", "42"}, {"bool", true, true}, {"nil", nil, nil},
		{"object_unchanged", map[string]any{"number": 42}, map[string]any{"number": 42}},
		{"array_unchanged", []any{42}, []any{42}},
		{"json_number_unchanged", json.Number("42"), json.Number("42")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { assert.Equal(t, tc.want, Normalize(tc.in)) })
	}
}

func TestDeepEqualNormalized(t *testing.T) {
	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"4.7_int_vs_float64", 42, float64(42), true},
		{"4.7_float64_vs_uint", float64(42), uint(42), true},
		{"4.7_int64_vs_float32", int64(42), float32(42), true},
		{"4.9_string_vs_number", "1", 1, false},
		{"4.9_bool_vs_string", true, "true", false},
		{"4.10_numbers_differ", 42, float64(43), false},
		{"4.14_object_safely_compared", map[string]any{"value": "pro"}, map[string]any{"value": "pro"}, true},
		{"4.14_nested_numbers_unchanged", []any{42}, []any{float64(42)}, false},
		{"nil", nil, nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { assert.Equal(t, tc.want, DeepEqualNormalized(tc.a, tc.b)) })
	}
}
