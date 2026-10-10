package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"flagforge/internal/apperror"
	"flagforge/internal/dto"
	"flagforge/internal/model"
	"flagforge/internal/repo"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FlagService struct{ db *gorm.DB }

func NewFlagService(db *gorm.DB) *FlagService { return &FlagService{db: db} }

// Delete removes the Flag and its dependent rules/history in one transaction.
// PostgreSQL's cascading foreign keys ensure a failed dependent delete rolls back.
func (s *FlagService) Delete(ctx context.Context, id string) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return repo.NewFlagRepo(tx).Delete(ctx, id)
	})
	return flagError(err, "", "")
}

// pgx may scan timestamptz into the process's local time zone. Normalize at
// the service boundary so JSON dates consistently follow the UTC contract.
func flagUTC(flag *model.Flag) *model.Flag {
	flag.CreatedAt = flag.CreatedAt.UTC()
	flag.UpdatedAt = flag.UpdatedAt.UTC()
	return flag
}

// WriteHistory uses the caller's transaction. It never starts or commits one.
func WriteHistory(ctx context.Context, tx *gorm.DB, flagID, action string, summary any) error {
	data, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	return repo.NewHistoryRepo(tx).Create(ctx, &model.History{FlagID: flagID, Action: action, Operator: "admin", Summary: datatypes.JSON(data)})
}
func flagError(err error, key, environment string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &apperror.Error{Code: apperror.FlagNotFound, Message: "flag not found"}
	}
	var mapped *repo.Error
	if errors.As(err, &mapped) && mapped.Code == repo.DuplicateFlagKey {
		return &apperror.Error{Code: apperror.DuplicateFlagKey, Message: fmt.Sprintf("key '%s' already exists in %s", key, environment), Cause: err}
	}
	return err
}
func (s *FlagService) Create(ctx context.Context, r dto.CreateFlag) (*model.Flag, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	flag := &model.Flag{Name: r.Name, Key: r.Key, Environment: r.Environment, Description: r.Description, DefaultValue: *r.DefaultValue, Enabled: true}
	if r.Enabled != nil {
		flag.Enabled = *r.Enabled
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := repo.NewFlagRepo(tx).Create(ctx, flag); err != nil {
			return err
		}
		return WriteHistory(ctx, tx, flag.ID, "flag_created", map[string]any{"flag": map[string]any{"name": flag.Name, "key": flag.Key, "environment": flag.Environment, "default_value": flag.DefaultValue, "enabled": flag.Enabled}})
	})
	if err != nil {
		return nil, flagError(err, r.Key, r.Environment)
	}
	return flagUTC(flag), nil
}
func (s *FlagService) Detail(ctx context.Context, id string) (*dto.FlagDetail, error) {
	flag, err := repo.NewFlagRepo(s.db).GetByID(ctx, id)
	if err != nil {
		return nil, flagError(err, "", "")
	}
	rules, err := repo.NewRuleRepo(s.db).ListByFlagOrdered(ctx, id)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].CreatedAt = rules[i].CreatedAt.UTC()
		rules[i].UpdatedAt = rules[i].UpdatedAt.UTC()
	}
	return &dto.FlagDetail{Flag: *flagUTC(flag), Rules: rules}, nil
}
func (s *FlagService) List(ctx context.Context, r dto.ListFlags) ([]repo.FlagListItem, int64, error) {
	items, total, err := repo.NewFlagRepo(s.db).List(ctx, r.Environment, r.Enabled, r.Q, r.Page, r.PageSize)
	for i := range items {
		flagUTC(&items[i].Flag)
	}
	return items, total, err
}

type Change struct {
	Old any `json:"old"`
	New any `json:"new"`
}

func (s *FlagService) Update(ctx context.Context, id string, r dto.UpdateFlag) (*model.Flag, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	var flag *model.Flag
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		flags := repo.NewFlagRepo(tx)
		var err error
		flag, err = flags.GetByID(ctx, id)
		if err != nil {
			return err
		}
		fields := map[string]any{}
		changes := map[string]Change{}
		if r.Name != nil && *r.Name != flag.Name {
			fields["name"] = *r.Name
			changes["name"] = Change{Old: flag.Name, New: *r.Name}
		}
		if r.Description != nil && *r.Description != flag.Description {
			fields["description"] = *r.Description
			changes["description"] = Change{Old: flag.Description, New: *r.Description}
		}
		if r.DefaultValue != nil && *r.DefaultValue != flag.DefaultValue {
			fields["default_value"] = *r.DefaultValue
			changes["default_value"] = Change{Old: flag.DefaultValue, New: *r.DefaultValue}
		}
		if len(fields) == 0 {
			return nil
		}
		if err = flags.Update(ctx, id, fields); err != nil {
			return err
		}
		if err = WriteHistory(ctx, tx, id, "flag_updated", map[string]any{"changes": changes}); err != nil {
			return err
		}
		flag, err = flags.GetByID(ctx, id)
		return err
	})
	if err != nil {
		return nil, flagError(err, "", "")
	}
	return flagUTC(flag), nil
}
func (s *FlagService) Status(ctx context.Context, id string, r dto.FlagStatus) (*model.Flag, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	var flag *model.Flag
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		flags := repo.NewFlagRepo(tx)
		var err error
		flag, err = flags.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if flag.Enabled == *r.Enabled {
			return nil
		}
		old := flag.Enabled
		if err = flags.Update(ctx, id, map[string]any{"enabled": *r.Enabled}); err != nil {
			return err
		}
		action := "flag_disabled"
		if *r.Enabled {
			action = "flag_enabled"
		}
		if err = WriteHistory(ctx, tx, id, action, map[string]any{"changes": map[string]Change{"enabled": {Old: old, New: *r.Enabled}}}); err != nil {
			return err
		}
		flag, err = flags.GetByID(ctx, id)
		return err
	})
	if err != nil {
		return nil, flagError(err, "", "")
	}
	return flagUTC(flag), nil
}
func (s *FlagService) History(ctx context.Context, id string, r dto.PageQuery) ([]dto.HistoryItem, int64, error) {
	if _, err := repo.NewFlagRepo(s.db).GetByID(ctx, id); err != nil {
		return nil, 0, flagError(err, "", "")
	}
	histories, total, err := repo.NewHistoryRepo(s.db).ListByFlag(ctx, id, r.Page, r.PageSize)
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.HistoryItem, 0, len(histories))
	for _, h := range histories {
		items = append(items, dto.HistoryItem{ID: h.ID, Action: h.Action, Operator: h.Operator, Summary: h.Summary, CreatedAt: h.CreatedAt.UTC()})
	}
	return items, total, nil
}
