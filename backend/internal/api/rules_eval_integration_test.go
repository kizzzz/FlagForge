//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"flagforge/internal/eval"
	"flagforge/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func ruleBody(priority int, attribute, operator, expected string, roll bool) string {
	return fmt.Sprintf(`{"priority":%d,"attribute":%q,"operator":%q,"expected_value":%s,"roll_value":%t}`, priority, attribute, operator, expected, roll)
}
func addRule(t *testing.T, h http.Handler, flag string, priority int, attribute, operator, expected string, roll bool) map[string]any {
	t.Helper()
	return request(t, h, "POST", "/api/v1/flags/"+flag+"/rules", ruleBody(priority, attribute, operator, expected, roll), 201)
}
func deleteRule(t *testing.T, h http.Handler, path string) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("DELETE", path, nil))
	require.Equal(t, 204, w.Code, w.Body.String())
	require.Empty(t, w.Body.String())
}
func TestRuleLifecycleScopeAndExactHistory(t *testing.T) {
	_, h := setupAPI(t)
	id := create(t, h, "rule-flag", "production")
	path := "/api/v1/flags/" + id + "/rules"
	first := addRule(t, h, id, 2, "  plan\u3000", "equals", `"pro"`, true)
	require.Equal(t, "plan", first["attribute"])
	rid := first["id"].(string)
	hs := history(t, h, id)
	require.Len(t, hs, 2)
	summary := hs[0].(map[string]any)["summary"].(map[string]any)["rule"].(map[string]any)
	require.Equal(t, map[string]any{"priority": float64(2), "attribute": "plan", "operator": "equals", "expected_value": "pro", "roll_value": true}, summary)
	addRule(t, h, id, 1, "country", "in", `["US",42,true]`, false)
	expectCode(t, request(t, h, "POST", path, ruleBody(2, "other", "equals", `1`, false), 409), "DUPLICATE_PRIORITY")
	expectCode(t, request(t, h, "PATCH", path+"/"+rid, `{"priority":1}`, 409), "DUPLICATE_PRIORITY")
	require.Len(t, history(t, h, id), 3)
	request(t, h, "PATCH", path+"/"+rid, `{"roll_value":false}`, 200)
	hs = history(t, h, id)
	require.Len(t, hs, 4)
	summary = hs[0].(map[string]any)["summary"].(map[string]any)
	require.Equal(t, rid, summary["rule_id"])
	require.Equal(t, float64(2), summary["priority"])
	require.Equal(t, map[string]any{"roll_value": map[string]any{"old": true, "new": false}}, summary["changes"])
	request(t, h, "PATCH", path+"/"+rid, `{"priority":2,"attribute":"plan","operator":"equals","expected_value":"pro","roll_value":false}`, 200)
	request(t, h, "PATCH", path+"/"+rid, `{}`, 200)
	require.Len(t, history(t, h, id), 4)
	for _, body := range []string{`{"expected_value":null}`, `{"roll_value":null}`, `{"priority":null}`, `{"attribute":null}`, `{"operator":null}`, `{"priority":2147483648}`, `{"flag_id":"x"}`, `{"operator":"in"}`} {
		expectCode(t, request(t, h, "PATCH", path+"/"+rid, body, 400), "VALIDATION_ERROR")
	}
	require.Len(t, history(t, h, id), 4)
	request(t, h, "PATCH", path+"/"+rid, `{"operator":"in","expected_value":["pro"]}`, 200)
	expectCode(t, request(t, h, "PATCH", path+"/"+rid, `{"operator":"equals"}`, 400), "VALIDATION_ERROR")
	request(t, h, "PATCH", "/api/v1/flags/"+id+"/status", `{"enabled":false}`, 200)
	addRule(t, h, id, 3, "vip", "equals", `true`, false)
	request(t, h, "PATCH", path+"/"+rid, `{"priority":4}`, 200)
	other := create(t, h, "other-flag", "production")
	for _, method := range []string{"PATCH", "DELETE"} {
		expectCode(t, request(t, h, method, "/api/v1/flags/"+other+"/rules/"+rid, `{}`, 404), "RULE_NOT_FOUND")
		expectCode(t, request(t, h, method, path+"/"+uuid.NewString(), `{}`, 404), "RULE_NOT_FOUND")
	}
	deleteRule(t, h, path+"/"+rid)
	summary = history(t, h, id)[0].(map[string]any)["summary"].(map[string]any)["rule"].(map[string]any)
	require.Equal(t, map[string]any{"priority": float64(4), "attribute": "plan", "operator": "in", "expected_value": []any{"pro"}, "roll_value": false}, summary)
	expectCode(t, request(t, h, "POST", "/api/v1/flags/"+uuid.NewString()+"/rules", ruleBody(1, "a", "equals", `1`, false), 404), "FLAG_NOT_FOUND")
}
func TestRuleValidation(t *testing.T) {
	_, h := setupAPI(t)
	id := create(t, h, "validation-rule", "staging")
	path := "/api/v1/flags/" + id + "/rules"
	cases := []struct{ body, field string }{
		{`{"priority":0,"attribute":"x","operator":"equals","expected_value":1,"roll_value":true}`, "priority"},
		{`{"priority":-1,"attribute":"x","operator":"equals","expected_value":1,"roll_value":true}`, "priority"},
		{`{"priority":2.5,"attribute":"x","operator":"equals","expected_value":1,"roll_value":true}`, "priority"},
		{`{"priority":"1","attribute":"x","operator":"equals","expected_value":1,"roll_value":true}`, "priority"},
		{ruleBody(2147483648, "x", "equals", `1`, true), "priority"},
		{ruleBody(1, " ", "equals", `1`, true), "attribute"},
		{ruleBody(1, strings.Repeat("中", 65), "equals", `1`, true), "attribute"},
		{ruleBody(1, "x", "bad", `1`, true), "operator"},
		{`{"priority":1,"attribute":"x","operator":"equals","expected_value":1}`, "roll_value"},
		{`{"priority":1,"attribute":"x","operator":"equals","roll_value":true}`, "expected_value"},
	}
	for _, operator := range []string{"equals", "in"} {
		vals := []string{`null`, `{}`, `{"a":1}`, `[]`, `[[1]]`, `[null]`, `[{}]`}
		if operator == "equals" {
			vals = append(vals, `[1]`)
		} else {
			vals = append(vals, `1`, `true`, `"pro"`)
		}
		for _, v := range vals {
			cases = append(cases, struct{ body, field string }{ruleBody(1, "x", operator, v, true), "expected_value"})
		}
	}
	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			out := request(t, h, "POST", path, tc.body, 400)
			expectCode(t, out, "VALIDATION_ERROR")
			detail := out["error"].(map[string]any)["details"].([]any)[0].(map[string]any)
			require.Equal(t, tc.field, detail["field"])
			require.NotEmpty(t, detail["reason"])
		})
	}
	for i, v := range []string{`42`, `true`, `"pro"`} {
		addRule(t, h, id, i+1, "x", "equals", v, false)
	}
	detail := request(t, h, "GET", "/api/v1/flags/"+id, "", 200)
	rid := detail["rules"].([]any)[0].(map[string]any)["id"].(string)
	request(t, h, "PATCH", path+"/"+rid, `{"expected_value":42.0}`, 200)
	require.Len(t, history(t, h, id), 4)
}
func TestRuleHistoryFailuresRollbackCRUD(t *testing.T) {
	db, h := setupAPI(t)
	id := create(t, h, "rule-atomic", "staging")
	r := addRule(t, h, id, 1, "x", "equals", `1`, true)
	path := "/api/v1/flags/" + id + "/rules"
	rid := r["id"].(string)
	require.NoError(t, db.Exec("ALTER TABLE histories ADD CONSTRAINT fail_history CHECK (false) NOT VALID").Error)
	for _, tc := range []struct{ method, path, body string }{
		{"POST", path, ruleBody(2, "x", "equals", `2`, false)},
		{"PATCH", path + "/" + rid, `{"roll_value":false,"priority":3}`},
		{"DELETE", path + "/" + rid, ""},
	} {
		expectCode(t, request(t, h, tc.method, tc.path, tc.body, 500), "INTERNAL_ERROR")
	}
	var rules []model.Rule
	require.NoError(t, db.Find(&rules).Error)
	require.Len(t, rules, 1)
	require.Equal(t, rid, rules[0].ID)
	require.Equal(t, 1, rules[0].Priority)
	require.True(t, rules[0].RollValue)
	require.Len(t, history(t, h, id), 2)
}
func evalRequest(key, attrs string) string {
	return `{"flag_key":"` + key + `","environment":"production","user_attributes":` + attrs + `}`
}
func insertEvalRules(t *testing.T, db *gorm.DB, id string) []eval.Rule {
	t.Helper()
	specs := []struct {
		a, op, value string
		roll         bool
	}{{"country", "in", `["US","CA","GB"]`, true}, {"plan", "equals", `"pro"`, true}, {"level", "equals", `42`, false}, {"vip", "equals", `true`, true}}
	rules := []eval.Rule{}
	for i, s := range specs {
		rule := model.Rule{FlagID: id, Priority: i + 1, Attribute: s.a, Operator: s.op, ExpectedValue: datatypes.JSON(s.value), RollValue: s.roll}
		require.NoError(t, db.Create(&rule).Error)
		var value any
		require.NoError(t, json.Unmarshal([]byte(s.value), &value))
		rules = append(rules, eval.Rule{ID: rule.ID, Priority: rule.Priority, Attribute: s.a, Operator: s.op, Expected: value, RollValue: s.roll})
	}
	return rules
}
func TestEvaluateContractAndEngineAgreement(t *testing.T) {
	db, h := setupAPI(t)
	id := create(t, h, "evaluate-flag", "production")
	rules := insertEvalRules(t, db, id)
	attrsList := []string{`{"country":"CN","plan":"pro"}`, `{"country":"US","plan":"pro"}`, `{}`, `{"level":42.0}`, `{"level":"42"}`, `{"vip":true}`, `{"country":{"nested":1},"plan":["pro"]}`, `{"country":null}`, `{"plan":"team"}`}
	for _, attrs := range attrsList {
		t.Run(attrs, func(t *testing.T) {
			out := request(t, h, "POST", "/api/v1/evaluate", evalRequest("evaluate-flag", attrs), 200)
			var values map[string]any
			require.NoError(t, json.Unmarshal([]byte(attrs), &values))
			want := eval.Evaluate(eval.Input{Enabled: true, Rules: rules, UserAttrs: values})
			require.Equal(t, want.Value, out["value"])
			require.Equal(t, want.ReasonType, out["reason"].(map[string]any)["type"])
			require.Equal(t, "evaluate-flag", out["flag_key"])
			require.Equal(t, "production", out["environment"])
			require.Equal(t, true, out["enabled"])
			stamp := out["evaluated_at"].(string)
			_, err := time.Parse(time.RFC3339Nano, stamp)
			require.NoError(t, err)
			require.True(t, strings.HasSuffix(stamp, "Z"))
			trace := out["trace"].([]any)
			require.Len(t, trace, len(want.Trace))
			for i, step := range want.Trace {
				require.Equal(t, map[string]any{"rule_id": step.RuleID, "priority": float64(step.Priority), "attribute": step.Attribute, "operator": step.Operator, "matched": step.Matched, "cause": step.Cause}, trace[i])
			}
			if want.HitRule != nil {
				require.Equal(t, want.HitRule.ID, out["reason"].(map[string]any)["rule_id"])
				require.Equal(t, float64(want.HitRule.Priority), out["reason"].(map[string]any)["priority"])
			}
		})
	}
	out := request(t, h, "POST", "/api/v1/evaluate", evalRequest("evaluate-flag", attrsList[0]), 200)
	require.Equal(t, `plan equals "pro" → true`, out["reason"].(map[string]any)["summary"])
	request(t, h, "PATCH", "/api/v1/flags/"+id, `{"default_value":true}`, 200)
	out = request(t, h, "POST", "/api/v1/evaluate", evalRequest("evaluate-flag", `{}`), 200)
	require.Equal(t, true, out["value"])
	create(t, h, "no-rules", "production")
	out = request(t, h, "POST", "/api/v1/evaluate", evalRequest("no-rules", `{}`), 200)
	require.Equal(t, []any{}, out["trace"])
	request(t, h, "PATCH", "/api/v1/flags/"+id+"/status", `{"enabled":false}`, 200)
	reads := []string{}
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("count_disabled", func(tx *gorm.DB) { reads = append(reads, tx.Statement.Table) }))
	out = request(t, h, "POST", "/api/v1/evaluate", evalRequest("evaluate-flag", `{"plan":"pro"}`), 200)
	require.Equal(t, []string{"flags"}, reads)
	require.Equal(t, false, out["value"])
	require.Equal(t, "disabled", out["reason"].(map[string]any)["type"])
	require.Equal(t, []any{map[string]any{"matched": false, "cause": "short_circuited_by_disabled_flag"}}, out["trace"])
}
func TestEvaluateRequestValidationBeforeLookup(t *testing.T) {
	_, h := setupAPI(t)
	for _, v := range []string{`null`, `[]`, `"a"`, `1`, `true`} {
		expectCode(t, request(t, h, "POST", "/api/v1/evaluate", evalRequest("missing-flag", v), 400), "INVALID_USER_ATTRIBUTES")
	}
	expectCode(t, request(t, h, "POST", "/api/v1/evaluate", `{"flag_key":"missing-flag","environment":"production"}`, 400), "INVALID_USER_ATTRIBUTES")
	expectCode(t, request(t, h, "POST", "/api/v1/evaluate", evalRequest("missing-flag", `{}`), 404), "FLAG_NOT_FOUND")
	for _, body := range []string{`{"flag_key":`, evalRequest("missing-flag", `{"a":1,}`)} {
		out := request(t, h, "POST", "/api/v1/evaluate", body, 400)
		expectCode(t, out, "INVALID_JSON")
		require.Contains(t, out["error"].(map[string]any)["message"], "byte")
	}
	expectCode(t, request(t, h, "POST", "/api/v1/evaluate", `{"flag_key":"missing-flag","environment":"prod","user_attributes":{}}`, 400), "INVALID_ENVIRONMENT")
	out := request(t, h, "POST", "/api/v1/evaluate", `{"flag_key":"Bad","environment":"production","user_attributes":{}}`, 400)
	expectCode(t, out, "VALIDATION_ERROR")
	require.Equal(t, "flag_key", out["error"].(map[string]any)["details"].([]any)[0].(map[string]any)["field"])
}
func batchRequest(key, list string) string {
	return `{"flag_key":"` + key + `","environment":"production","user_attributes_list":` + list + `}`
}
func TestBatchStatisticsLimitsAndQueryCount(t *testing.T) {
	db, h := setupAPI(t)
	id := create(t, h, "batch-flag", "production")
	rules := insertEvalRules(t, db, id)
	reads := []string{}
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("count_batch", func(tx *gorm.DB) { reads = append(reads, tx.Statement.Table) }))
	list := `[{"plan":"pro"},{"country":"US"},{},{"plan":"pro"},{"level":42}]`
	out := request(t, h, "POST", "/api/v1/evaluate/batch", batchRequest("batch-flag", list), 200)
	require.Equal(t, []string{"flags", "rules"}, reads)
	results := out["results"].([]any)
	require.Len(t, results, 5)
	for i, r := range results {
		require.Equal(t, float64(i), r.(map[string]any)["index"])
	}
	summary := out["summary"].(map[string]any)
	require.Equal(t, float64(5), summary["total"])
	require.Equal(t, float64(3), summary["true_count"])
	require.Equal(t, float64(2), summary["false_count"])
	require.Equal(t, float64(1), summary["default_count"])
	require.Equal(t, []any{map[string]any{"rule_id": rules[1].ID, "priority": float64(2), "attribute": "plan", "count": float64(2)}, map[string]any{"rule_id": rules[0].ID, "priority": float64(1), "attribute": "country", "count": float64(1)}, map[string]any{"rule_id": rules[2].ID, "priority": float64(3), "attribute": "level", "count": float64(1)}}, summary["hit_rules"])
	expectCode(t, request(t, h, "POST", "/api/v1/evaluate/batch", batchRequest("batch-flag", `[]`), 400), "VALIDATION_ERROR")
	expectCode(t, request(t, h, "POST", "/api/v1/evaluate/batch", batchRequest("batch-flag", "["+strings.TrimSuffix(strings.Repeat("{},", 101), ",")+"]"), 400), "BATCH_LIMIT_EXCEEDED")
	request(t, h, "POST", "/api/v1/evaluate/batch", batchRequest("batch-flag", "["+strings.TrimSuffix(strings.Repeat("{},", 100), ",")+"]"), 200)
	for _, bad := range []string{`null`, `[]`, `"x"`, `1`, `true`} {
		reads = nil
		out = request(t, h, "POST", "/api/v1/evaluate/batch", batchRequest("missing-flag", `[{},{},`+bad+`]`), 400)
		expectCode(t, out, "INVALID_USER_ATTRIBUTES")
		require.Equal(t, float64(2), out["error"].(map[string]any)["details"].(map[string]any)["index"])
		require.Empty(t, reads)
	}
	for _, bad := range []string{`null`, `{}`, `"x"`} {
		expectCode(t, request(t, h, "POST", "/api/v1/evaluate/batch", batchRequest("batch-flag", bad), 400), "VALIDATION_ERROR")
	}
	request(t, h, "PATCH", "/api/v1/flags/"+id+"/status", `{"enabled":false}`, 200)
	reads = nil
	out = request(t, h, "POST", "/api/v1/evaluate/batch", batchRequest("batch-flag", list), 200)
	require.Equal(t, []string{"flags"}, reads)
	summary = out["summary"].(map[string]any)
	require.Equal(t, float64(0), summary["true_count"])
	require.Equal(t, float64(5), summary["false_count"])
	require.Equal(t, float64(0), summary["default_count"])
	require.Empty(t, summary["hit_rules"])
	for _, r := range out["results"].([]any) {
		require.Equal(t, "disabled", r.(map[string]any)["reason"].(map[string]any)["type"])
	}
}
