package repo

import (
	"context"

	"flagforge/internal/model"
	"gorm.io/gorm"
)

type HistoryRepo struct{ db *gorm.DB }

func NewHistoryRepo(db *gorm.DB) *HistoryRepo { return &HistoryRepo{db: db} }

func (r *HistoryRepo) Create(ctx context.Context, history *model.History) error {
	return MapError(r.db.WithContext(ctx).Create(history).Error)
}

func (r *HistoryRepo) ListByFlag(ctx context.Context, flagID string, page, pageSize int) ([]model.History, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.History{}).Where("flag_id = ?", flagID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, MapError(err)
	}
	items := make([]model.History, 0)
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, MapError(err)
}
