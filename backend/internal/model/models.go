package model

import (
	"time"

	"gorm.io/datatypes"
)

type Flag struct {
	ID           string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name         string    `gorm:"column:name" json:"name"`
	Key          string    `gorm:"column:key" json:"key"`
	Environment  string    `gorm:"column:environment" json:"environment"`
	Enabled      bool      `gorm:"column:enabled" json:"enabled"`
	DefaultValue bool      `gorm:"column:default_value" json:"default_value"`
	Description  string    `gorm:"column:description" json:"description"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Flag) TableName() string { return "flags" }

type Rule struct {
	ID            string         `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	FlagID        string         `gorm:"column:flag_id;type:uuid" json:"flag_id"`
	Attribute     string         `gorm:"column:attribute" json:"attribute"`
	Operator      string         `gorm:"column:operator" json:"operator"`
	ExpectedValue datatypes.JSON `gorm:"column:expected_value;type:jsonb" json:"expected_value"`
	RollValue     bool           `gorm:"column:roll_value" json:"roll_value"`
	Priority      int            `gorm:"column:priority" json:"priority"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Rule) TableName() string { return "rules" }

type History struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FlagID    string         `gorm:"column:flag_id;type:uuid" json:"flag_id"`
	Action    string         `gorm:"column:action" json:"action"`
	Operator  string         `gorm:"column:operator" json:"operator"`
	Summary   datatypes.JSON `gorm:"column:summary;type:jsonb" json:"summary"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (History) TableName() string { return "histories" }
