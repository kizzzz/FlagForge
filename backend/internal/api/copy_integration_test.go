//go:build integration

package api

import (
	"testing"

	"flagforge/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCopyPreservesSourceAndPriorities(t *testing.T) {
	_, h := setupAPI(t)
	id := create(t, h, "copy-source", "staging")
	request(t, h, "PATCH", "/api/v1/flags/"+id, `{"description":"copy description","default_value":true}`, 200)
	addRule(t, h, id, 7, "level", "equals", `42`, false)
	addRule(t, h, id, 2, "plan", "in", `["pro",true,42]`, true)
	source := request(t, h, "GET", "/api/v1/flags/"+id, "", 200)
	hs := history(t, h, id)
	path := "/api/v1/flags/" + id + "/copy"
	expectCode(t, request(t, h, "POST", path, `{"target_environment":"staging"}`, 400), "COPY_TARGET_CONFLICT")
	for _, body := range []string{`{}`, `{"target_environment":null}`, `{"target_environment":"prod"}`, `{"target_environment":1}`, `{"target_environment":"production","extra":true}`} {
		request(t, h, "POST", path, body, 400)
	}
	copied := request(t, h, "POST", path, `{"target_environment":"production"}`, 201)
	require.Equal(t, false, copied["enabled"])
	require.Equal(t, true, copied["default_value"])
	require.Equal(t, "copy description", copied["description"])
	require.Equal(t, source["name"], copied["name"])
	require.Equal(t, source["key"], copied["key"])
	require.Equal(t, "production", copied["environment"])
	require.NotEqual(t, id, copied["id"])
	targetID := copied["id"].(string)
	detail := request(t, h, "GET", "/api/v1/flags/"+targetID, "", 200)
	oldRules := source["rules"].([]any)
	newRules := detail["rules"].([]any)
	require.Len(t, newRules, 2)
	for i, v := range newRules {
		rule := v.(map[string]any)
		old := oldRules[i].(map[string]any)
		require.NotEqual(t, old["id"], rule["id"])
		require.Equal(t, targetID, rule["flag_id"])
		for _, field := range []string{"priority", "attribute", "operator", "expected_value", "roll_value"} {
			require.Equal(t, old[field], rule[field], field)
		}
	}
	copiedHistory := history(t, h, targetID)
	require.Len(t, copiedHistory, 1)
	row := copiedHistory[0].(map[string]any)
	require.Equal(t, "flag_copied", row["action"])
	summary := row["summary"].(map[string]any)
	require.Equal(t, map[string]any{"flag_id": id, "environment": "staging"}, summary["source"])
	require.Equal(t, map[string]any{"name": source["name"], "key": "copy-source", "environment": "production", "default_value": true, "enabled": false}, summary["flag"])
	require.Equal(t, source, request(t, h, "GET", "/api/v1/flags/"+id, "", 200))
	require.Equal(t, hs, history(t, h, id))
	expectCode(t, request(t, h, "POST", path, `{"target_environment":"production"}`, 409), "DUPLICATE_FLAG_KEY")
	expectCode(t, request(t, h, "POST", "/api/v1/flags/"+uuid.NewString()+"/copy", `{"target_environment":"production"}`, 404), "FLAG_NOT_FOUND")
	empty := create(t, h, "copy-empty", "development")
	copyEmpty := request(t, h, "POST", "/api/v1/flags/"+empty+"/copy", `{"target_environment":"staging"}`, 201)
	require.Empty(t, request(t, h, "GET", "/api/v1/flags/"+copyEmpty["id"].(string), "", 200)["rules"])
}
func TestCopyFailureRollsBackEveryWrite(t *testing.T) {
	for _, failure := range []string{"rule", "history"} {
		t.Run(failure, func(t *testing.T) {
			db, h := setupAPI(t)
			id := create(t, h, "copy-atomic", "staging")
			addRule(t, h, id, 1, "good", "equals", `true`, true)
			addRule(t, h, id, 2, "blocked", "equals", `1`, false)
			source := request(t, h, "GET", "/api/v1/flags/"+id, "", 200)
			hs := history(t, h, id)
			if failure == "rule" {
				require.NoError(t, db.Exec("ALTER TABLE rules ADD CONSTRAINT fail_copy_rule CHECK (attribute <> 'blocked') NOT VALID").Error)
			} else {
				require.NoError(t, db.Exec("ALTER TABLE histories ADD CONSTRAINT fail_copy_history CHECK (false) NOT VALID").Error)
			}
			expectCode(t, request(t, h, "POST", "/api/v1/flags/"+id+"/copy", `{"target_environment":"production"}`, 500), "INTERNAL_ERROR")
			var count int64
			require.NoError(t, db.Model(&model.Flag{}).Count(&count).Error)
			require.Equal(t, int64(1), count)
			require.NoError(t, db.Model(&model.Rule{}).Count(&count).Error)
			require.Equal(t, int64(2), count)
			require.Equal(t, source, request(t, h, "GET", "/api/v1/flags/"+id, "", 200))
			require.Equal(t, hs, history(t, h, id))
		})
	}
}
