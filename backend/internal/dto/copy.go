package dto

import "flagforge/internal/apperror"

type CopyFlag struct {
	TargetEnvironment string `json:"target_environment"`
}

func (r *CopyFlag) Validate() error {
	if err := ValidateEnvironment(r.TargetEnvironment); err != nil {
		return &apperror.Error{Code: apperror.InvalidEnvironment, Message: "target_environment must be development, staging or production", Details: []apperror.FieldError{{Field: "target_environment", Reason: "invalid environment"}}}
	}
	return nil
}
func DecodeCopyFlag(body []byte) (CopyFlag, error) {
	var r CopyFlag
	fields, err := DecodeObject(body)
	if err != nil {
		return r, err
	}
	if err = DecodeFields(body, fields, &r, "target_environment"); err != nil {
		return r, err
	}
	return r, r.Validate()
}
