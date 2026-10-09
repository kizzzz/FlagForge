package repo

import (
	"context"

	"flagforge/internal/model"
	"gorm.io/gorm"
)

type RuleRepo struct{ db *gorm.DB }

func NewRuleRepo(db *gorm.DB) *RuleRepo { return &RuleRepo{db: db} }

func (r *RuleRepo) Create(ctx context.Context, rule *model.Rule) error {
	return MapError(r.db.WithContext(ctx).Create(rule).Error)
}

// GetByID scopes the rule to its parent flag to avoid exposing another flag's rule.
func (r *RuleRepo) GetByID(ctx context.Context, flagID, id string) (*model.Rule, error) {
	var rule model.Rule
	if err := r.db.WithContext(ctx).Where("flag_id = ? AND id = ?", flagID, id).First(&rule).Error; err != nil {
		return nil, MapError(err)
	}
	return &rule, nil
}

func (r *RuleRepo) ListByFlagOrdered(ctx context.Context, flagID string) ([]model.Rule, error) {
	items := make([]model.Rule, 0)
	err := r.db.WithContext(ctx).Where("flag_id = ?", flagID).Order("priority ASC").Find(&items).Error
	return items, MapError(err)
}

func (r *RuleRepo) Update(ctx context.Context, flagID, id string, fields map[string]any) error {
	return writeResult(r.db.WithContext(ctx).Model(&model.Rule{}).Where("flag_id = ? AND id = ?", flagID, id).Updates(fields))
}

func (r *RuleRepo) Delete(ctx context.Context, flagID, id string) error {
	return writeResult(r.db.WithContext(ctx).Where("flag_id = ? AND id = ?", flagID, id).Delete(&model.Rule{}))
}
