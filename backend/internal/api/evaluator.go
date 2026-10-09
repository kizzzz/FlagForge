package api

import (
	"flagforge/internal/dto"
	"flagforge/internal/service"
	"github.com/gin-gonic/gin"
)

type EvaluatorHandler struct{ service *service.EvaluatorService }

func NewEvaluatorHandler(s *service.EvaluatorService) *EvaluatorHandler {
	return &EvaluatorHandler{service: s}
}
func (h *EvaluatorHandler) Evaluate(c *gin.Context) {
	body, err := ReadBody(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	r, err := dto.DecodeEvaluate(body)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Evaluate(c.Request.Context(), r)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, 200, out)
}
func (h *EvaluatorHandler) Batch(c *gin.Context) {
	body, err := ReadBody(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	r, err := dto.DecodeBatch(body)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Batch(c.Request.Context(), r)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, 200, out)
}
