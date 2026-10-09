package api

import (
	"errors"
	"log"

	"flagforge/internal/apperror"
	"github.com/gin-gonic/gin"
)

type Error = apperror.Error
type FieldError = apperror.FieldError

const (
	ValidationError       = apperror.ValidationError
	InvalidKey            = apperror.InvalidKey
	InvalidEnvironment    = apperror.InvalidEnvironment
	ImmutableField        = apperror.ImmutableField
	InvalidJSON           = apperror.InvalidJSON
	InvalidUserAttributes = apperror.InvalidUserAttributes
	BatchLimitExceeded    = apperror.BatchLimitExceeded
	CopyTargetConflict    = apperror.CopyTargetConflict
	FlagNotFound          = apperror.FlagNotFound
	RuleNotFound          = apperror.RuleNotFound
	DuplicateFlagKey      = apperror.DuplicateFlagKey
	DuplicatePriority     = apperror.DuplicatePriority
	InternalError         = apperror.InternalError
)

type ErrorEnvelope struct {
	Error *Error `json:"error"`
}

func HandleError(c *gin.Context, err error) {
	var business *apperror.Error
	if errors.As(err, &business) && apperror.HTTPStatus(business.Code) != 500 {
		respondError(c, apperror.HTTPStatus(business.Code), business.Code, business.Message, business.Details)
		return
	}
	log.Printf("request %s %s failed: %v", c.Request.Method, c.Request.URL.Path, err)
	respondError(c, 500, InternalError, "internal server error", nil)
}
