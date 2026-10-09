package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"flagforge/internal/apperror"
	"flagforge/internal/model"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,63}$`)

type CreateFlag struct {
	Name         string `json:"name"`
	Key          string `json:"key"`
	Environment  string `json:"environment"`
	DefaultValue *bool  `json:"default_value"`
	Description  string `json:"description"`
	Enabled      *bool  `json:"enabled"`
}
type UpdateFlag struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	DefaultValue *bool   `json:"default_value"`
}
type FlagStatus struct {
	Enabled *bool `json:"enabled"`
}
type PageQuery struct {
	Page     int
	PageSize int
}
type ListFlags struct {
	PageQuery
	Environment string
	Enabled     *string
	Q           string
}
type FlagDetail struct {
	model.Flag
	Rules []model.Rule `json:"rules"`
}
type HistoryItem struct {
	ID        int64          `json:"id"`
	Action    string         `json:"action"`
	Operator  string         `json:"operator"`
	Summary   datatypes.JSON `json:"summary"`
	CreatedAt time.Time      `json:"created_at"`
}

func ValidateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return &apperror.Error{Code: apperror.InvalidKey, Message: "key must match ^[a-z][a-z0-9._-]{1,63}$", Details: []apperror.FieldError{{Field: "key", Reason: "invalid format"}}}
	}
	return nil
}
func ValidateEnvironment(environment string) error {
	switch environment {
	case "development", "staging", "production":
		return nil
	}
	return &apperror.Error{Code: apperror.InvalidEnvironment, Message: "environment must be development, staging or production", Details: []apperror.FieldError{{Field: "environment", Reason: "invalid environment"}}}
}
func ValidateID(id string) error {
	// Require the database-compatible canonical UUID representation. uuid.Parse
	// also accepts URNs, which PostgreSQL cannot use directly as a uuid value.
	if parsed, err := uuid.Parse(id); err != nil || !strings.EqualFold(id, parsed.String()) {
		return apperror.Validation("id", "must be a UUID in xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx format")
	}
	return nil
}
func validateName(name *string) error {
	*name = strings.TrimSpace(*name)
	if n := utf8.RuneCountInString(*name); n < 1 || n > 100 {
		return apperror.Validation("name", "must contain 1 to 100 characters after trimming")
	}
	return nil
}
func (r *CreateFlag) Validate() error {
	if err := ValidateKey(r.Key); err != nil {
		return err
	}
	if err := ValidateEnvironment(r.Environment); err != nil {
		return err
	}
	if err := validateName(&r.Name); err != nil {
		return err
	}
	if r.DefaultValue == nil {
		return apperror.Validation("default_value", "required boolean")
	}
	return nil
}
func (r *UpdateFlag) Validate() error {
	if r.Name != nil {
		return validateName(r.Name)
	}
	return nil
}
func (r *FlagStatus) Validate() error {
	if r.Enabled == nil {
		return apperror.Validation("enabled", "required boolean")
	}
	return nil
}

// DecodeObject preserves field presence independently of null and zero values.
// T3b can reuse this when validating raw JSON fields.
func DecodeObject(body []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if !json.Valid(body) {
		var value any
		err := json.Unmarshal(body, &value)
		var syntax *json.SyntaxError
		offset := int64(len(body) + 1)
		if errors.As(err, &syntax) {
			offset = syntax.Offset
		}
		return nil, &apperror.Error{Code: apperror.InvalidJSON, Message: fmt.Sprintf("invalid JSON at byte %d", offset)}
	}
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return nil, apperror.Validation("body", "must be a JSON object")
	}
	return fields, nil
}
func DecodeFields(body []byte, fields map[string]json.RawMessage, target any, allowed ...string) error {
	allow := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		allow[key] = true
	}
	// Stable error order keeps responses reproducible when multiple fields are wrong.
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !allow[key] {
			return apperror.Validation(key, "unknown field")
		}
		if string(fields[key]) == "null" {
			return apperror.Validation(key, "must not be null")
		}
	}
	if err := json.Unmarshal(body, target); err != nil {
		var mismatch *json.UnmarshalTypeError
		if errors.As(err, &mismatch) {
			return apperror.Validation(mismatch.Field, "invalid field type")
		}
		return apperror.Validation("body", "invalid field type")
	}
	return nil
}
func DecodeCreateFlag(body []byte) (CreateFlag, error) {
	var r CreateFlag
	fields, err := DecodeObject(body)
	if err != nil {
		return r, err
	}
	if err = DecodeFields(body, fields, &r, "name", "key", "environment", "default_value", "description", "enabled"); err != nil {
		return r, err
	}
	return r, r.Validate()
}
func DecodeUpdateFlag(body []byte) (UpdateFlag, error) {
	var r UpdateFlag
	fields, err := DecodeObject(body)
	if err != nil {
		return r, err
	}
	for _, key := range []string{"key", "environment"} {
		if _, exists := fields[key]; exists {
			return r, &apperror.Error{Code: apperror.ImmutableField, Message: key + " cannot be changed after creation", Details: []apperror.FieldError{{Field: key, Reason: "immutable field"}}}
		}
	}
	if err = DecodeFields(body, fields, &r, "name", "description", "default_value"); err != nil {
		return r, err
	}
	return r, r.Validate()
}
func DecodeFlagStatus(body []byte) (FlagStatus, error) {
	var r FlagStatus
	fields, err := DecodeObject(body)
	if err != nil {
		return r, err
	}
	if err = DecodeFields(body, fields, &r, "enabled"); err != nil {
		return r, err
	}
	return r, r.Validate()
}
func ParsePageQuery(values url.Values) (PageQuery, error) {
	r := PageQuery{Page: 1, PageSize: 20}
	for _, field := range []string{"page", "page_size"} {
		if len(values[field]) > 1 {
			return r, apperror.Validation(field, "must occur only once")
		}
		value := values.Get(field)
		if value == "" {
			continue
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return r, apperror.Validation(field, "must be a non-negative integer")
		}
		if n == 0 {
			continue
		}
		if field == "page" {
			r.Page = n
		} else {
			r.PageSize = n
		}
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	if r.Page-1 > int(^uint(0)>>1)/r.PageSize {
		return r, apperror.Validation("page", "page offset is too large")
	}
	return r, nil
}
func ParseListFlags(values url.Values) (ListFlags, error) {
	page, err := ParsePageQuery(values)
	r := ListFlags{PageQuery: page, Environment: values.Get("environment"), Q: values.Get("q")}
	if err != nil {
		return r, err
	}
	if r.Environment != "" {
		if err = ValidateEnvironment(r.Environment); err != nil {
			return r, err
		}
	}
	if enabled := values.Get("enabled"); enabled != "" {
		if enabled != "true" && enabled != "false" {
			return r, apperror.Validation("enabled", "must be true or false")
		}
		r.Enabled = &enabled
	}
	return r, nil
}
