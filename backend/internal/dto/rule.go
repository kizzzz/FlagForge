package dto

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"flagforge/internal/apperror"
)

type CreateRule struct {
	Priority      *int            `json:"priority"`
	Attribute     *string         `json:"attribute"`
	Operator      *string         `json:"operator"`
	ExpectedValue json.RawMessage `json:"expected_value"`
	RollValue     *bool           `json:"roll_value"`
}
type UpdateRule CreateRule

func ValidateExpected(operator string, raw json.RawMessage) error {
	var value any
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return apperror.Validation("expected_value", "required valid JSON scalar or non-empty scalar array")
	}
	scalar := func(v any) bool {
		switch v.(type) {
		case string, float64, bool:
			return true
		}
		return false
	}
	if operator == "equals" {
		if scalar(value) {
			return nil
		}
		return apperror.Validation("expected_value", "equals requires a string, number or boolean")
	}
	if operator == "" && scalar(value) {
		return nil
	}
	list, ok := value.([]any)
	if !ok || len(list) == 0 {
		return apperror.Validation("expected_value", "in requires a non-empty array of scalars")
	}
	for _, v := range list {
		if !scalar(v) {
			return apperror.Validation("expected_value", "array elements must be strings, numbers or booleans")
		}
	}
	return nil
}
func validateRuleFields(r *CreateRule, required bool) error {
	if r.Priority == nil {
		if required {
			return apperror.Validation("priority", "required integer")
		}
	} else if *r.Priority < 1 || *r.Priority > 2147483647 {
		return apperror.Validation("priority", "must be an integer between 1 and 2147483647")
	}
	if r.Attribute == nil {
		if required {
			return apperror.Validation("attribute", "required string")
		}
	} else {
		*r.Attribute = strings.TrimSpace(*r.Attribute)
		if n := utf8.RuneCountInString(*r.Attribute); n < 1 || n > 64 {
			return apperror.Validation("attribute", "must contain 1 to 64 characters after trimming")
		}
	}
	operator := ""
	if r.Operator == nil {
		if required {
			return apperror.Validation("operator", "required equals or in")
		}
	} else {
		operator = *r.Operator
		if operator != "equals" && operator != "in" {
			return apperror.Validation("operator", "must be equals or in")
		}
	}
	if required || len(r.ExpectedValue) > 0 {
		if err := ValidateExpected(operator, r.ExpectedValue); err != nil {
			return err
		}
	}
	if required && r.RollValue == nil {
		return apperror.Validation("roll_value", "required boolean")
	}
	return nil
}
func (r *CreateRule) Validate() error { return validateRuleFields(r, true) }
func (r *UpdateRule) Validate() error { return validateRuleFields((*CreateRule)(r), false) }
func DecodeCreateRule(body []byte) (CreateRule, error) {
	var r CreateRule
	fields, err := DecodeObject(body)
	if err != nil {
		return r, err
	}
	if err = DecodeFields(body, fields, &r, "priority", "attribute", "operator", "expected_value", "roll_value"); err != nil {
		return r, err
	}
	return r, r.Validate()
}
func DecodeUpdateRule(body []byte) (UpdateRule, error) {
	var r UpdateRule
	fields, err := DecodeObject(body)
	if err != nil {
		return r, err
	}
	if err = DecodeFields(body, fields, &r, "priority", "attribute", "operator", "expected_value", "roll_value"); err != nil {
		return r, err
	}
	return r, r.Validate()
}
