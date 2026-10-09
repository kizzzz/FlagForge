package repo

import (
	"context"

	"flagforge/internal/model"
	"gorm.io/gorm"
)

type FlagRepo struct{ db *gorm.DB }

func NewFlagRepo(db *gorm.DB) *FlagRepo { return &FlagRepo{db: db} }

type FlagListItem struct {
	model.Flag `gorm:"embedded"`
	RuleCount  int64 `gorm:"column:rule_count" json:"rule_count"`
}

func (r *FlagRepo) Create(ctx context.Context, flag *model.Flag) error {
	return MapError(r.db.WithContext(ctx).Create(flag).Error)
}

func (r *FlagRepo) GetByID(ctx context.Context, id string) (*model.Flag, error) {
	var flag model.Flag
	if err := r.db.WithContext(ctx).First(&flag, "id = ?", id).Error; err != nil {
		return nil, MapError(err)
	}
	return &flag, nil
}

func (r *FlagRepo) GetByKeyEnv(ctx context.Context, key, environment string) (*model.Flag, error) {
	var flag model.Flag
	if err := r.db.WithContext(ctx).Where("key = ? AND environment = ?", key, environment).First(&flag).Error; err != nil {
		return nil, MapError(err)
	}
	return &flag, nil
}

// Update takes a map so legitimate false and empty-string values are persisted.
// The service chooses allowed fields and owns the transaction and history.
func (r *FlagRepo) Update(ctx context.Context, id string, fields map[string]any) error {
	return writeResult(r.db.WithContext(ctx).Model(&model.Flag{}).Where("id = ?", id).Updates(fields))
}

func (r *FlagRepo) Delete(ctx context.Context, id string) error {
	return writeResult(r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Flag{}))
}

func (r *FlagRepo) List(ctx context.Context, environment string, enabled *string, q string, page, pageSize int) ([]FlagListItem, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Flag{})
	if environment != "" {
		query = query.Where("flags.environment = ?", environment)
	}
	if enabled != nil {
		query = query.Where("flags.enabled = ?", *enabled)
	}
	if q != "" {
		query = query.Where("(flags.name ILIKE ? OR flags.key ILIKE ?)", "%"+q+"%", "%"+q+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, MapError(err)
	}
	items := make([]FlagListItem, 0)
	err := query.Select("flags.*, (SELECT COUNT(*) FROM rules WHERE rules.flag_id = flags.id) AS rule_count").
		Order("flags.created_at DESC, flags.id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&items).Error
	return items, total, MapError(err)
}
