//go:build integration

package main

import (
	"os"
	"testing"

	"flagforge/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func seedTestDB(t *testing.T) *gorm.DB {
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
	if err := tx.Exec("DELETE FROM flags").Error; err != nil {
		t.Fatal(err)
	}
	return tx
}

func assertCounts(t *testing.T, db *gorm.DB, flags, rules, histories int64) {
	t.Helper()
	for _, tc := range []struct {
		table string
		want  int64
	}{{"flags", flags}, {"rules", rules}, {"histories", histories}} {
		var count int64
		if err := db.Table(tc.table).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != tc.want {
			t.Fatalf("%s count: got %d want %d", tc.table, count, tc.want)
		}
	}
}

func TestSeedIdempotentPreservesExistingFlagsAndMissingRules(t *testing.T) {
	db := seedTestDB(t)
	if err := runSeed(db); err != nil {
		t.Fatal(err)
	}
	assertCounts(t, db, 4, 7, 4)
	if err := runSeed(db); err != nil {
		t.Fatal(err)
	}
	assertCounts(t, db, 4, 7, 4)
	var billing model.Flag
	if err := db.Where("key = ? AND environment = ?", "billing-portal", "development").First(&billing).Error; err != nil {
		t.Fatal(err)
	}
	if billing.Enabled || !billing.DefaultValue {
		t.Fatalf("billing flags incorrect: %+v", billing)
	}
	var production model.Flag
	if err := db.Where("key = ? AND environment = ?", "checkout-redesign-2026", "production").First(&production).Error; err != nil {
		t.Fatal(err)
	}
	var rules []model.Rule
	if err := db.Where("flag_id = ?", production.ID).Order("priority ASC").Find(&rules).Error; err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 || string(rules[0].ExpectedValue) != `["US", "CA", "GB"]` || string(rules[1].ExpectedValue) != `"pro"` || string(rules[2].ExpectedValue) != `[9001, 9002, 9003]` {
		t.Fatalf("incorrect production JSON rules: %+v", rules)
	}
	if err := db.Where("flag_id = ? AND priority = 3", production.ID).Delete(&model.Rule{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Flag{}).Where("id = ?", production.ID).Update("name", "Changed by user").Error; err != nil {
		t.Fatal(err)
	}
	if err := runSeed(db); err != nil {
		t.Fatal(err)
	}
	assertCounts(t, db, 4, 6, 4)
	if err := db.First(&production, "id = ?", production.ID).Error; err != nil {
		t.Fatal(err)
	}
	if production.Name != "Changed by user" {
		t.Fatal("existing flag overwritten")
	}
	var history model.History
	if err := db.Where("flag_id = ?", billing.ID).First(&history).Error; err != nil {
		t.Fatal(err)
	}
	if history.Action != "flag_created" || history.Operator != "admin" {
		t.Fatalf("bad seed history: %+v", history)
	}
}

func TestSeedRollsBackFlagsAndRulesWhenHistoryFails(t *testing.T) {
	db := seedTestDB(t)
	if err := db.Exec("ALTER TABLE histories ADD CONSTRAINT t1_seed_fail CHECK (action <> 'flag_created')").Error; err != nil {
		t.Fatal(err)
	}
	if err := runSeed(db); err == nil {
		t.Fatal("expected history CHECK failure")
	}
	assertCounts(t, db, 0, 0, 0)
}
