//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"flagforge/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAPI(t *testing.T) (*gorm.DB, http.Handler) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, dsn, "DATABASE_URL required: integration tests must use real PostgreSQL")
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), NowFunc: func() time.Time { return time.Now().UTC() }}
	admin, err := gorm.Open(postgres.Open(dsn), cfg)
	require.NoError(t, err)
	conn, err := admin.DB()
	require.NoError(t, err)
	schema := "t3a_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schema).Error)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), cfg)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		require.NoError(t, admin.Exec("DROP SCHEMA "+schema+" CASCADE").Error)
		_ = conn.Close()
	})
	ddl, err := os.ReadFile("../../migrations/0001_init.up.sql")
	require.NoError(t, err)
	// The shared extension already exists; only create tables in this test's schema.
	lines := strings.Split(string(ddl), "\n")
	var isolated []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "CREATE EXTENSION") {
			isolated = append(isolated, line)
		}
	}
	require.NoError(t, db.Exec(strings.Join(isolated, "\n")).Error)
	return db, NewRouter(db)
}

func request(t *testing.T, h http.Handler, method, path, body string, want int) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	require.Equal(t, want, w.Code, w.Body.String())
	var result map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), w.Body.String())
	assertUTCTimes(t, result)
	return result
}

func assertUTCTimes(t *testing.T, value any) {
	t.Helper()
	switch node := value.(type) {
	case map[string]any:
		for key, v := range node {
			if key == "created_at" || key == "updated_at" {
				stamp, ok := v.(string)
				require.True(t, ok)
				require.True(t, strings.HasSuffix(stamp, "Z"), "%s must use UTC Z: %s", key, stamp)
			}
			assertUTCTimes(t, v)
		}
	case []any:
		for _, v := range node {
			assertUTCTimes(t, v)
		}
	}
}
func create(t *testing.T, h http.Handler, key, env string) string {
	t.Helper()
	out := request(t, h, "POST", "/api/v1/flags", `{"name":"  中文开关  ","key":"`+key+`","environment":"`+env+`","default_value":false}`, 201)
	require.Equal(t, "中文开关", out["name"])
	require.Equal(t, true, out["enabled"])
	require.Equal(t, false, out["default_value"])
	require.NotEmpty(t, out["created_at"])
	return out["id"].(string)
}
func expectCode(t *testing.T, out map[string]any, code string) {
	t.Helper()
	e, ok := out["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, code, e["code"])
	require.NotEmpty(t, e["message"])
	require.Contains(t, e, "details")
}
func history(t *testing.T, h http.Handler, id string) []any {
	t.Helper()
	out := request(t, h, "GET", "/api/v1/flags/"+id+"/history", "", 200)
	return out["items"].([]any)
}

func TestCreateAndUniqueEnvironment(t *testing.T) {
	_, h := setupAPI(t)
	id := create(t, h, "a.b-c_d", "production")
	hs := history(t, h, id)
	require.Len(t, hs, 1)
	row := hs[0].(map[string]any)
	require.Equal(t, "flag_created", row["action"])
	require.Equal(t, "admin", row["operator"])
	summary := row["summary"].(map[string]any)["flag"].(map[string]any)
	require.Len(t, summary, 5)
	require.Equal(t, "a.b-c_d", summary["key"])
	out := request(t, h, "POST", "/api/v1/flags", `{"name":"Duplicate","key":"a.b-c_d","environment":"production","default_value":true}`, 409)
	expectCode(t, out, "DUPLICATE_FLAG_KEY")
	require.Contains(t, out["error"].(map[string]any)["message"], "production")
	create(t, h, "a.b-c_d", "staging")
}

func TestPatchPresenceAndExactDiff(t *testing.T) {
	_, h := setupAPI(t)
	id := create(t, h, "patch-flag", "development")
	path := "/api/v1/flags/" + id
	for _, body := range []string{`{"key":""}`, `{"key":null}`, `{"key":"patch-flag"}`, `{"environment":""}`, `{"environment":null}`, `{"environment":"development"}`} {
		expectCode(t, request(t, h, "PATCH", path, body, 400), "IMMUTABLE_FIELD")
	}
	out := request(t, h, "PATCH", path, `{"name":"  改名\u3000","default_value":false,"description":""}`, 200)
	require.Equal(t, "改名", out["name"])
	hs := history(t, h, id)
	require.Len(t, hs, 2)
	changes := hs[0].(map[string]any)["summary"].(map[string]any)["changes"].(map[string]any)
	require.Len(t, changes, 1)
	require.Equal(t, map[string]any{"old": "中文开关", "new": "改名"}, changes["name"])
	request(t, h, "PATCH", path, `{"name":"改名","default_value":false,"description":""}`, 200)
	request(t, h, "PATCH", path, `{}`, 200)
	require.Len(t, history(t, h, id), 2)
	request(t, h, "PATCH", path, `{"default_value":true,"description":"x"}`, 200)
	hs = history(t, h, id)
	require.Len(t, hs, 3)
	changes = hs[0].(map[string]any)["summary"].(map[string]any)["changes"].(map[string]any)
	require.Len(t, changes, 2)
}

func TestStatusIsIdempotent(t *testing.T) {
	_, h := setupAPI(t)
	id := create(t, h, "status-flag", "staging")
	path := "/api/v1/flags/" + id + "/status"
	out := request(t, h, "PATCH", path, `{"enabled":false}`, 200)
	require.Equal(t, false, out["enabled"])
	hs := history(t, h, id)
	require.Len(t, hs, 2)
	require.Equal(t, "flag_disabled", hs[0].(map[string]any)["action"])
	request(t, h, "PATCH", path, `{"enabled":false}`, 200)
	require.Len(t, history(t, h, id), 2)
	request(t, h, "PATCH", path, `{"enabled":true}`, 200)
	hs = history(t, h, id)
	require.Len(t, hs, 3)
	require.Equal(t, map[string]any{"old": false, "new": true}, hs[0].(map[string]any)["summary"].(map[string]any)["changes"].(map[string]any)["enabled"])
	out = request(t, h, "GET", "/api/v1/flags/"+id+"/history?page=2&page_size=1", "", 200)
	require.Equal(t, float64(3), out["total"])
	require.Equal(t, "flag_disabled", out["items"].([]any)[0].(map[string]any)["action"])
}

func TestListFiltersPaginationAndOrderedDetails(t *testing.T) {
	db, h := setupAPI(t)
	id := create(t, h, "needle-key", "production")
	create(t, h, "needle-key", "development")
	other := create(t, h, "needle-other", "production")
	request(t, h, "PATCH", "/api/v1/flags/"+other+"/status", `{"enabled":false}`, 200)
	for _, p := range []int{3, 1, 2} {
		require.NoError(t, db.Create(&model.Rule{FlagID: id, Priority: p, Attribute: "plan", Operator: "equals", ExpectedValue: datatypes.JSON(`"pro"`), RollValue: true}).Error)
	}
	out := request(t, h, "GET", "/api/v1/flags?environment=production&enabled=true&q=NEEDLE&page_size=1", "", 200)
	require.Equal(t, float64(1), out["total"])
	require.Equal(t, float64(3), out["items"].([]any)[0].(map[string]any)["rule_count"])
	out = request(t, h, "GET", "/api/v1/flags?q=中文", "", 200)
	require.Equal(t, float64(3), out["total"])
	out = request(t, h, "GET", "/api/v1/flags?page=99", "", 200)
	require.Empty(t, out["items"])
	require.Equal(t, float64(3), out["total"])
	for _, tc := range []struct {
		query string
		size  float64
	}{{"0", 20}, {"101", 100}} {
		out = request(t, h, "GET", "/api/v1/flags?page_size="+tc.query, "", 200)
		require.Equal(t, tc.size, out["page_size"])
	}
	expectCode(t, request(t, h, "GET", "/api/v1/flags?page_size=nope", "", 400), "VALIDATION_ERROR")
	for _, query := range []string{"page=1&page=2", "page_size=10&page_size=20", "page=9223372036854775808", "page=9223372036854775807&page_size=100", "page=-1", "page=1.1", "enabled=yes", "environment=prod"} {
		out = request(t, h, "GET", "/api/v1/flags?"+query, "", 400)
		if query == "environment=prod" {
			expectCode(t, out, "INVALID_ENVIRONMENT")
		} else {
			expectCode(t, out, "VALIDATION_ERROR")
		}
	}
	out = request(t, h, "GET", "/api/v1/flags/"+id, "", 200)
	rules := out["rules"].([]any)
	require.Len(t, rules, 3)
	for i, r := range rules {
		require.Equal(t, float64(i+1), r.(map[string]any)["priority"])
	}
}

func TestValidationAndErrorEnvelope(t *testing.T) {
	_, h := setupAPI(t)
	cases := []struct{ body, code string }{
		{`{"name":"ok","key":"Bad","environment":"production","default_value":false}`, "INVALID_KEY"},
		{`{"name":"ok","key":"a1","environment":"prod","default_value":false}`, "INVALID_ENVIRONMENT"},
		{`{"name":"\u3000  ","key":"a1","environment":"production","default_value":false}`, "VALIDATION_ERROR"},
		{`{"name":"ok","key":"a1","environment":"production"}`, "VALIDATION_ERROR"},
		{`{"name":"ok","key":"a1","environment":"production","default_value":null}`, "VALIDATION_ERROR"},
		{`{"name":`, "INVALID_JSON"}, {`[]`, "VALIDATION_ERROR"}, {`null`, "VALIDATION_ERROR"},
	}
	for _, tc := range cases {
		out := request(t, h, "POST", "/api/v1/flags", tc.body, 400)
		expectCode(t, out, tc.code)
	}
	id := create(t, h, "valid-flag", "production")
	for _, body := range []string{`{"name":null}`, `{"description":null}`, `{"default_value":null}`, `{"default_value":"false"}`, `{"name":"` + strings.Repeat("中", 101) + `"}`} {
		expectCode(t, request(t, h, "PATCH", "/api/v1/flags/"+id, body, 400), "VALIDATION_ERROR")
	}
	for _, body := range []string{`{}`, `{"enabled":null}`, `{"enabled":"false"}`} {
		expectCode(t, request(t, h, "PATCH", "/api/v1/flags/"+id+"/status", body, 400), "VALIDATION_ERROR")
	}
	for _, path := range []string{"/api/v1/flags/" + uuid.NewString(), "/api/v1/flags/" + uuid.NewString() + "/history"} {
		expectCode(t, request(t, h, "GET", path, "", 404), "FLAG_NOT_FOUND")
	}
	expectCode(t, request(t, h, "GET", "/api/v1/flags/not-uuid", "", 400), "VALIDATION_ERROR")
	expectCode(t, request(t, h, "GET", "/api/v1/flags/urn:uuid:"+id, "", 400), "VALIDATION_ERROR")
	expectCode(t, request(t, h, "GET", "/missing", "", 404), "VALIDATION_ERROR")
	expectCode(t, request(t, h, "PUT", "/api/v1/flags/"+id, "", 405), "VALIDATION_ERROR")
	require.Equal(t, "ok", request(t, h, "GET", "/healthz", "", 200)["status"])
}

func TestHistoryFailureRollsBackBusinessWrites(t *testing.T) {
	db, h := setupAPI(t)
	id := create(t, h, "rollback-flag", "production")
	// This CHECK is only in the random test schema; every real history INSERT fails.
	require.NoError(t, db.Exec("ALTER TABLE histories ADD CONSTRAINT fail_new_history CHECK (action = 'flag_created')").Error)
	expectCode(t, request(t, h, "PATCH", "/api/v1/flags/"+id, `{"name":"Must roll back"}`, 500), "INTERNAL_ERROR")
	expectCode(t, request(t, h, "PATCH", "/api/v1/flags/"+id+"/status", `{"enabled":false}`, 500), "INTERNAL_ERROR")
	detail := request(t, h, "GET", "/api/v1/flags/"+id, "", 200)
	require.Equal(t, "中文开关", detail["name"])
	require.Equal(t, true, detail["enabled"])
	require.Len(t, history(t, h, id), 1)
	require.NoError(t, db.Exec("ALTER TABLE histories DROP CONSTRAINT fail_new_history").Error)
	require.NoError(t, db.Exec("ALTER TABLE histories ADD CONSTRAINT fail_new_history CHECK (false) NOT VALID").Error)
	out := request(t, h, "POST", "/api/v1/flags", `{"name":"Atomic create","key":"rollback-create","environment":"production","default_value":false}`, 500)
	expectCode(t, out, "INTERNAL_ERROR")
	require.NotContains(t, out["error"].(map[string]any)["message"], "constraint")
	var count int64
	require.NoError(t, db.Model(&model.Flag{}).Where("key = ?", "rollback-create").Count(&count).Error)
	require.Zero(t, count)
}
