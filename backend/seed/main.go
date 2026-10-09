package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"flagforge/internal/model"
	"flagforge/internal/repo"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{NowFunc: func() time.Time { return time.Now().UTC() }})
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err := runSeed(db); err != nil {
		log.Fatal(err)
	}
	log.Println("seed complete")
}

type demoFlag struct {
	flag  model.Flag
	rules []model.Rule
}

// runSeed is an atomic initialization operation, separate from business services.
// Existing flags are deliberately untouched, even if their rules are missing.
func runSeed(db *gorm.DB) error {
	demos := []demoFlag{
		{
			flag: model.Flag{Name: "Checkout redesign 2026", Key: "checkout-redesign-2026", Environment: "production", Enabled: true, DefaultValue: false, Description: "Checkout rollout for supported countries, pro plans and internal users."},
			rules: []model.Rule{
				{Attribute: "country", Operator: "in", ExpectedValue: datatypes.JSON(`["US","CA","GB"]`), RollValue: true, Priority: 1},
				{Attribute: "plan", Operator: "equals", ExpectedValue: datatypes.JSON(`"pro"`), RollValue: true, Priority: 2},
				{Attribute: "user_id", Operator: "in", ExpectedValue: datatypes.JSON(`[9001,9002,9003]`), RollValue: true, Priority: 3},
			},
		},
		{
			flag: model.Flag{Name: "Checkout redesign 2026", Key: "checkout-redesign-2026", Environment: "development", Enabled: true, DefaultValue: true, Description: "Development defaults to the redesign, with the old checkout for CN."},
			rules: []model.Rule{
				{Attribute: "country", Operator: "equals", ExpectedValue: datatypes.JSON(`"CN"`), RollValue: false, Priority: 1},
			},
		},
		{
			flag: model.Flag{Name: "New search ranking", Key: "new-search-rank", Environment: "staging", Enabled: true, DefaultValue: false, Description: "Staging rollout for pro and team plans or beta testers."},
			rules: []model.Rule{
				{Attribute: "plan", Operator: "in", ExpectedValue: datatypes.JSON(`["pro","team"]`), RollValue: true, Priority: 1},
				{Attribute: "beta_tester", Operator: "equals", ExpectedValue: datatypes.JSON(`true`), RollValue: true, Priority: 2},
			},
		},
		{
			flag: model.Flag{Name: "Billing portal", Key: "billing-portal", Environment: "development", Enabled: false, DefaultValue: true, Description: "Disabled flag demonstrates that even a matching rule cannot enable it."},
			rules: []model.Rule{
				{Attribute: "plan", Operator: "equals", ExpectedValue: datatypes.JSON(`"pro"`), RollValue: true, Priority: 1},
			},
		},
	}
	ctx := context.Background()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, demo := range demos {
			created := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "environment"}, {Name: "key"}},
				DoNothing: true,
			}).Create(&demo.flag)
			if created.Error != nil {
				return repo.MapError(created.Error)
			}
			if created.RowsAffected == 0 {
				continue
			}
			rules := repo.NewRuleRepo(tx)
			for _, rule := range demo.rules {
				rule.FlagID = demo.flag.ID
				if err := rules.Create(ctx, &rule); err != nil {
					return err
				}
			}
			summary, err := json.Marshal(map[string]any{"flag": map[string]any{
				"name": demo.flag.Name, "key": demo.flag.Key, "environment": demo.flag.Environment,
				"default_value": demo.flag.DefaultValue, "enabled": demo.flag.Enabled,
			}})
			if err != nil {
				return fmt.Errorf("marshal seed history: %w", err)
			}
			history := model.History{FlagID: demo.flag.ID, Action: "flag_created", Operator: "admin", Summary: datatypes.JSON(summary)}
			if err := repo.NewHistoryRepo(tx).Create(ctx, &history); err != nil {
				return err
			}
		}
		return nil
	})
}
