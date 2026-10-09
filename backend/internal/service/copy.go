package service

import (
	"context"

	"flagforge/internal/apperror"
	"flagforge/internal/dto"
	"flagforge/internal/model"
	"flagforge/internal/repo"
	"gorm.io/gorm"
)

func (s *FlagService) Copy(ctx context.Context, id string, r dto.CopyFlag) (*model.Flag, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	var copied *model.Flag
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		flags := repo.NewFlagRepo(tx)
		source, err := flags.GetByID(ctx, id)
		if err != nil {
			return flagError(err, "", "")
		}
		if source.Environment == r.TargetEnvironment {
			return &apperror.Error{Code: apperror.CopyTargetConflict, Message: "target environment must differ from source", Details: []apperror.FieldError{{Field: "target_environment", Reason: "same as source environment"}}}
		}
		copied = &model.Flag{Name: source.Name, Key: source.Key, Environment: r.TargetEnvironment, Enabled: false, DefaultValue: source.DefaultValue, Description: source.Description}
		if err = flags.Create(ctx, copied); err != nil {
			return flagError(err, source.Key, r.TargetEnvironment)
		}
		rules := repo.NewRuleRepo(tx)
		sourceRules, err := rules.ListByFlagOrdered(ctx, id)
		if err != nil {
			return err
		}
		for _, rule := range sourceRules {
			next := model.Rule{FlagID: copied.ID, Attribute: rule.Attribute, Operator: rule.Operator, ExpectedValue: rule.ExpectedValue, RollValue: rule.RollValue, Priority: rule.Priority}
			if err = rules.Create(ctx, &next); err != nil {
				return ruleError(err)
			}
		}
		return WriteHistory(ctx, tx, copied.ID, "flag_copied", map[string]any{
			"flag":   map[string]any{"name": copied.Name, "key": copied.Key, "environment": copied.Environment, "default_value": copied.DefaultValue, "enabled": copied.Enabled},
			"source": map[string]any{"flag_id": source.ID, "environment": source.Environment},
		})
	})
	if err != nil {
		return nil, err
	}
	return flagUTC(copied), nil
}
