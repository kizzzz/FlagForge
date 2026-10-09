// Package apperror defines errors shared by DTOs, services and HTTP adapters.
package apperror

const (
	ValidationError       = "VALIDATION_ERROR"
	InvalidKey            = "INVALID_KEY"
	InvalidEnvironment    = "INVALID_ENVIRONMENT"
	ImmutableField        = "IMMUTABLE_FIELD"
	InvalidJSON           = "INVALID_JSON"
	InvalidUserAttributes = "INVALID_USER_ATTRIBUTES"
	BatchLimitExceeded    = "BATCH_LIMIT_EXCEEDED"
	CopyTargetConflict    = "COPY_TARGET_CONFLICT"
	FlagNotFound          = "FLAG_NOT_FOUND"
	RuleNotFound          = "RULE_NOT_FOUND"
	DuplicateFlagKey      = "DUPLICATE_FLAG_KEY"
	DuplicatePriority     = "DUPLICATE_PRIORITY"
	InternalError         = "INTERNAL_ERROR"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
	Cause   error  `json:"-"`
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }

type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func Validation(field, reason string) *Error {
	return &Error{Code: ValidationError, Message: "request validation failed", Details: []FieldError{{Field: field, Reason: reason}}}
}

func HTTPStatus(code string) int {
	switch code {
	case FlagNotFound, RuleNotFound:
		return 404
	case DuplicateFlagKey, DuplicatePriority:
		return 409
	case ValidationError, InvalidKey, InvalidEnvironment, ImmutableField, InvalidJSON, InvalidUserAttributes, BatchLimitExceeded, CopyTargetConflict:
		return 400
	default:
		return 500
	}
}
