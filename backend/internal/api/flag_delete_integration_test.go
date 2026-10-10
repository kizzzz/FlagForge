//go:build integration

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"flagforge/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func removeFlag(t *testing.T, h http.Handler, id string) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/flags/"+id, nil))
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	require.Empty(t, w.Body.String())
}

func TestDeleteFlagCascadeAndEnvironmentIsolation(t *testing.T) {
	db, h := setupAPI(t)
	id := create(t, h, "delete-shared-key", "production")
	sibling := create(t, h, "delete-shared-key", "staging")
	addRule(t, h, id, 1, "plan", "equals", `"pro"`, true)
	require.Len(t, history(t, h, id), 2)
	removeFlag(t, h, id)
	for _, table := range []string{"rules", "histories"} {
		var count int64
		require.NoError(t, db.Table(table).Where("flag_id = ?", id).Count(&count).Error)
		require.Zero(t, count, table)
	}
	for _, path := range []string{"/api/v1/flags/" + id, "/api/v1/flags/" + id + "/history"} {
		expectCode(t, request(t, h, "GET", path, "", 404), "FLAG_NOT_FOUND")
	}
	expectCode(t, request(t, h, "POST", "/api/v1/evaluate", evalRequest("delete-shared-key", `{}`), 404), "FLAG_NOT_FOUND")
	expectCode(t, request(t, h, "POST", "/api/v1/evaluate/batch", batchRequest("delete-shared-key", `[{}]`), 404), "FLAG_NOT_FOUND")
	request(t, h, "GET", "/api/v1/flags/"+sibling, "", 200)
	require.Len(t, history(t, h, sibling), 1)
	replacement := create(t, h, "delete-shared-key", "production")
	require.NotEqual(t, id, replacement)
	require.Len(t, history(t, h, replacement), 1)
}

func TestDeleteFlagValidationAndRepeatedDeletion(t *testing.T) {
	_, h := setupAPI(t)
	expectCode(t, request(t, h, "DELETE", "/api/v1/flags/not-a-uuid", "", 400), "VALIDATION_ERROR")
	expectCode(t, request(t, h, "DELETE", "/api/v1/flags/"+uuid.NewString(), "", 404), "FLAG_NOT_FOUND")
	id := create(t, h, "delete-twice", "development")
	removeFlag(t, h, id)
	expectCode(t, request(t, h, "DELETE", "/api/v1/flags/"+id, "", 404), "FLAG_NOT_FOUND")
}

func TestDeleteFlagFailureRollsBackCascade(t *testing.T) {
	db, h := setupAPI(t)
	id := create(t, h, "delete-rollback", "production")
	rule := addRule(t, h, id, 1, "plan", "equals", `"pro"`, true)
	before := history(t, h, id)
	require.NoError(t, db.Exec(`CREATE FUNCTION reject_history_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected delete failure'; END $$;
 CREATE TRIGGER reject_history_delete BEFORE DELETE ON histories FOR EACH ROW EXECUTE FUNCTION reject_history_delete();`).Error)
	out := request(t, h, "DELETE", "/api/v1/flags/"+id, "", 500)
	expectCode(t, out, "INTERNAL_ERROR")
	require.NotContains(t, out["error"].(map[string]any)["message"], "injected")
	detail := request(t, h, "GET", "/api/v1/flags/"+id, "", 200)
	require.Equal(t, id, detail["id"])
	require.Equal(t, rule["id"], detail["rules"].([]any)[0].(map[string]any)["id"])
	require.Equal(t, before, history(t, h, id))
	var count int64
	require.NoError(t, db.Model(&model.Flag{}).Where("id = ?", id).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, db.Exec("DROP TRIGGER reject_history_delete ON histories").Error)
	removeFlag(t, h, id)
}
