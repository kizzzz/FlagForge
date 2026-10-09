//go:build integration

package repo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"flagforge/internal/model"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL required for integration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

func TestFlagCRUDPreservesZeroValuesAndFiltersList(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	flags := NewFlagRepo(db)
	rules := NewRuleRepo(db)
	flag := model.Flag{Name: "T1 Unique Search Name", Key: "t1-repo-search", Environment: "development", Enabled: false, DefaultValue: true, Description: "before"}
	if err := flags.Create(ctx, &flag); err != nil {
		t.Fatal(err)
	}
	if flag.ID == "" || flag.CreatedAt.IsZero() || flag.UpdatedAt.IsZero() {
		t.Fatal("missing DB ID or automatic times")
	}
	other := model.Flag{Name: "T1 Unique Search Name", Key: flag.Key, Environment: "production", Enabled: true}
	if err := flags.Create(ctx, &other); err != nil {
		t.Fatal(err)
	}
	for _, priority := range []int{3, 1, 2} {
		rule := model.Rule{FlagID: flag.ID, Attribute: "plan", Operator: "equals", ExpectedValue: datatypes.JSON(`"pro"`), RollValue: true, Priority: priority}
		if err := rules.Create(ctx, &rule); err != nil {
			t.Fatal(err)
		}
	}
	disabled := "false"
	items, total, err := flags.List(ctx, "development", &disabled, "UNIQUE SEARCH", 1, 20)
	if err != nil || total != 1 || len(items) != 1 || items[0].RuleCount != 3 || items[0].Enabled {
		t.Fatalf("bad filtered list: %v %d %v", items, total, err)
	}
	items, total, err = flags.List(ctx, "development", &disabled, "T1-REPO-SEARCH", 2, 20)
	if err != nil || total != 1 || len(items) != 0 || items == nil {
		t.Fatalf("bad out-of-range page: %v %d %v", items, total, err)
	}
	before := flag.UpdatedAt
	if err := flags.Update(ctx, flag.ID, map[string]any{"enabled": true}); err != nil {
		t.Fatal(err)
	}
	intermediate, err := flags.GetByID(ctx, flag.ID)
	if err != nil || !intermediate.Enabled {
		t.Fatalf("enable update failed: %+v %v", intermediate, err)
	}
	if err := flags.Update(ctx, flag.ID, map[string]any{"enabled": false, "default_value": false, "description": ""}); err != nil {
		t.Fatal(err)
	}
	got, err := flags.GetByKeyEnv(ctx, flag.Key, flag.Environment)
	if err != nil || got.Enabled || got.DefaultValue || got.Description != "" || !got.UpdatedAt.After(before) {
		t.Fatalf("zero-value update failed: %+v %v", got, err)
	}
	got, err = flags.GetByID(ctx, flag.ID)
	if err != nil || got.ID != flag.ID {
		t.Fatalf("GetByID: %v", err)
	}
	if err := flags.Delete(ctx, flag.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := flags.GetByID(ctx, flag.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted flag found: %v", err)
	}
	if err := flags.Update(ctx, flag.ID, map[string]any{"enabled": true}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing update: %v", err)
	}
	if err := flags.Delete(ctx, flag.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing delete: %v", err)
	}
}

func TestRuleCRUDOrdersPriorityAndScopesToFlag(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	flags, rules := NewFlagRepo(db), NewRuleRepo(db)
	flag := model.Flag{Name: "Rule test", Key: "t1-rules", Environment: "development"}
	if err := flags.Create(ctx, &flag); err != nil {
		t.Fatal(err)
	}
	for _, priority := range []int{3, 1, 2} {
		rule := model.Rule{FlagID: flag.ID, Attribute: "country", Operator: "in", ExpectedValue: datatypes.JSON(`["US","CA"]`), Priority: priority, RollValue: true}
		if err := rules.Create(ctx, &rule); err != nil {
			t.Fatal(err)
		}
	}
	items, err := rules.ListByFlagOrdered(ctx, flag.ID)
	if err != nil || len(items) != 3 {
		t.Fatalf("list rules: %v", err)
	}
	for i, rule := range items {
		if rule.Priority != i+1 {
			t.Fatal("priority not ascending")
		}
	}
	rule := items[0]
	if err := rules.Update(ctx, flag.ID, rule.ID, map[string]any{"roll_value": false, "expected_value": datatypes.JSON(`true`)}); err != nil {
		t.Fatal(err)
	}
	got, err := rules.GetByID(ctx, flag.ID, rule.ID)
	if err != nil || got.RollValue || string(got.ExpectedValue) != "true" || !got.UpdatedAt.After(rule.UpdatedAt) {
		t.Fatalf("rule update: %+v %v", got, err)
	}
	wrongID := "00000000-0000-0000-0000-000000000000"
	if _, err := rules.GetByID(ctx, wrongID, rule.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign rule visible: %v", err)
	}
	if err := rules.Update(ctx, wrongID, rule.ID, map[string]any{"roll_value": true}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign update: %v", err)
	}
	if err := rules.Delete(ctx, wrongID, rule.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign delete: %v", err)
	}
	if err := rules.Delete(ctx, flag.ID, rule.ID); err != nil {
		t.Fatal(err)
	}
	if err := rules.Delete(ctx, flag.ID, rule.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing delete: %v", err)
	}
}

func TestHistoryListReturnsLatestFirstAndCorrectTotal(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	flag := model.Flag{Name: "History test", Key: "t1-history", Environment: "development"}
	if err := NewFlagRepo(db).Create(ctx, &flag); err != nil {
		t.Fatal(err)
	}
	histories := NewHistoryRepo(db)
	for i := 0; i < 3; i++ {
		h := model.History{FlagID: flag.ID, Action: "flag_created", Operator: "admin", Summary: datatypes.JSON(`{"flag":{}}`), CreatedAt: time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC)}
		if err := histories.Create(ctx, &h); err != nil {
			t.Fatal(err)
		}
		if h.ID == 0 {
			t.Fatal("missing history ID")
		}
	}
	items, total, err := histories.ListByFlag(ctx, flag.ID, 1, 2)
	if err != nil || total != 3 || len(items) != 2 || items[0].CreatedAt.Day() != 3 || items[1].CreatedAt.Day() != 2 {
		t.Fatalf("history page: %v %d %v", items, total, err)
	}
	items, total, err = histories.ListByFlag(ctx, flag.ID, 3, 2)
	if err != nil || total != 3 || len(items) != 0 || items == nil {
		t.Fatalf("empty history page: %v %d %v", items, total, err)
	}
}

func TestRepoMapsRealPostgresUniqueViolations(t *testing.T) {
	ctx := context.Background()
	t.Run("flag create", func(t *testing.T) {
		db := testDB(t)
		flags := NewFlagRepo(db)
		flag := model.Flag{Name: "Duplicate test", Key: "t1-duplicate", Environment: "development"}
		if err := flags.Create(ctx, &flag); err != nil {
			t.Fatal(err)
		}
		duplicate := model.Flag{Name: flag.Name, Key: flag.Key, Environment: flag.Environment}
		var mapped *Error
		if err := flags.Create(ctx, &duplicate); !errors.As(err, &mapped) || mapped.Code != DuplicateFlagKey {
			t.Fatalf("flag mapping: %v", err)
		}
	})
	t.Run("rule update", func(t *testing.T) {
		db := testDB(t)
		flag := model.Flag{Name: "Duplicate priority test", Key: "t1-priority-duplicate", Environment: "development"}
		if err := NewFlagRepo(db).Create(ctx, &flag); err != nil {
			t.Fatal(err)
		}
		rules := NewRuleRepo(db)
		var last model.Rule
		for _, priority := range []int{1, 2} {
			last = model.Rule{FlagID: flag.ID, Attribute: "plan", Operator: "equals", ExpectedValue: datatypes.JSON(`"pro"`), Priority: priority}
			if err := rules.Create(ctx, &last); err != nil {
				t.Fatal(err)
			}
		}
		var mapped *Error
		if err := rules.Update(ctx, flag.ID, last.ID, map[string]any{"priority": 1}); !errors.As(err, &mapped) || mapped.Code != DuplicatePriority {
			t.Fatalf("rule mapping: %v", err)
		}
	})
}
