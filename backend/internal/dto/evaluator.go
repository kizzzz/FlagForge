package dto

import (
	"encoding/json"
	"time"

	"flagforge/internal/apperror"
)

type EvaluateRequest struct {
	FlagKey        string         `json:"flag_key"`
	Environment    string         `json:"environment"`
	UserAttributes map[string]any `json:"user_attributes"`
}
type BatchRequest struct {
	FlagKey            string           `json:"flag_key"`
	Environment        string           `json:"environment"`
	UserAttributesList []map[string]any `json:"user_attributes_list"`
}
type Reason struct {
	Type     string `json:"type"`
	RuleID   string `json:"rule_id,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Summary  string `json:"summary,omitempty"`
}
type TraceStep struct {
	RuleID    string `json:"rule_id,omitempty"`
	Priority  int    `json:"priority,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	Operator  string `json:"operator,omitempty"`
	Matched   bool   `json:"matched"`
	Cause     string `json:"cause"`
}
type Evaluation struct {
	Value  bool        `json:"value"`
	Reason Reason      `json:"reason"`
	Trace  []TraceStep `json:"trace"`
}
type EvaluateResponse struct {
	FlagKey     string `json:"flag_key"`
	Environment string `json:"environment"`
	Enabled     bool   `json:"enabled"`
	Evaluation
	EvaluatedAt time.Time `json:"evaluated_at"`
}
type BatchResult struct {
	Index int `json:"index"`
	Evaluation
}
type HitRuleCount struct {
	RuleID    string `json:"rule_id"`
	Priority  int    `json:"priority"`
	Attribute string `json:"attribute"`
	Count     int    `json:"count"`
}
type BatchSummary struct {
	Total        int            `json:"total"`
	TrueCount    int            `json:"true_count"`
	FalseCount   int            `json:"false_count"`
	DefaultCount int            `json:"default_count"`
	HitRules     []HitRuleCount `json:"hit_rules"`
}
type BatchResponse struct {
	Results []BatchResult `json:"results"`
	Summary BatchSummary  `json:"summary"`
}

func decodeAttributes(raw json.RawMessage, index *int) (map[string]any, error) {
	var attrs map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &attrs) != nil || attrs == nil {
		details := map[string]any{"field": "user_attributes", "reason": "must be a JSON object"}
		if index != nil {
			details["field"] = "user_attributes_list"
			details["index"] = *index
		}
		return nil, &apperror.Error{Code: apperror.InvalidUserAttributes, Message: "user_attributes must be a JSON object", Details: details}
	}
	return attrs, nil
}
func validateEvaluationKey(key string) error {
	if ValidateKey(key) != nil {
		return apperror.Validation("flag_key", "must match ^[a-z][a-z0-9._-]{1,63}$")
	}
	return nil
}
func (r *EvaluateRequest) Validate() error {
	if r.UserAttributes == nil {
		_, err := decodeAttributes(nil, nil)
		return err
	}
	if err := validateEvaluationKey(r.FlagKey); err != nil {
		return err
	}
	return ValidateEnvironment(r.Environment)
}
func (r *BatchRequest) Validate() error {
	if len(r.UserAttributesList) == 0 {
		return apperror.Validation("user_attributes_list", "must contain 1 to 100 objects")
	}
	if len(r.UserAttributesList) > 100 {
		return &apperror.Error{Code: apperror.BatchLimitExceeded, Message: "batch must not contain more than 100 objects"}
	}
	for i, attrs := range r.UserAttributesList {
		if attrs == nil {
			_, err := decodeAttributes(nil, &i)
			return err
		}
	}
	if err := validateEvaluationKey(r.FlagKey); err != nil {
		return err
	}
	return ValidateEnvironment(r.Environment)
}
func DecodeEvaluate(body []byte) (EvaluateRequest, error) {
	var r EvaluateRequest
	fields, err := DecodeObject(body)
	if err != nil {
		return r, err
	}
	attrs, err := decodeAttributes(fields["user_attributes"], nil)
	if err != nil {
		return r, err
	}
	if err = DecodeFields(body, fields, &r, "flag_key", "environment", "user_attributes"); err != nil {
		return r, err
	}
	r.UserAttributes = attrs
	return r, r.Validate()
}
func DecodeBatch(body []byte) (BatchRequest, error) {
	var r BatchRequest
	fields, err := DecodeObject(body)
	if err != nil {
		return r, err
	}
	var rawList []json.RawMessage
	raw, exists := fields["user_attributes_list"]
	if !exists || json.Unmarshal(raw, &rawList) != nil || len(rawList) == 0 {
		return r, apperror.Validation("user_attributes_list", "must contain 1 to 100 objects")
	}
	if len(rawList) > 100 {
		return r, &apperror.Error{Code: apperror.BatchLimitExceeded, Message: "batch must not contain more than 100 objects"}
	}
	for i, raw := range rawList {
		if _, err = decodeAttributes(raw, &i); err != nil {
			return r, err
		}
	}
	if err = DecodeFields(body, fields, &r, "flag_key", "environment", "user_attributes_list"); err != nil {
		return r, err
	}
	return r, r.Validate()
}
