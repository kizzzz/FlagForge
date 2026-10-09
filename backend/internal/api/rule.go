package api

import (
	"flagforge/internal/dto"
	"flagforge/internal/service"
	"github.com/gin-gonic/gin"
)

type RuleHandler struct{ service *service.RuleService }

func NewRuleHandler(s *service.RuleService) *RuleHandler { return &RuleHandler{service: s} }
func ruleIDs(c *gin.Context) (string, string, error) {
	flag, err := flagID(c)
	if err != nil {
		return "", "", err
	}
	id := c.Param("rid")
	return flag, id, dto.ValidateID(id)
}
func (h *RuleHandler) Create(c *gin.Context) {
	id, err := flagID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	body, err := ReadBody(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	r, err := dto.DecodeCreateRule(body)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Create(c.Request.Context(), id, r)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, 201, out)
}
func (h *RuleHandler) Update(c *gin.Context) {
	flag, id, err := ruleIDs(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	body, err := ReadBody(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	r, err := dto.DecodeUpdateRule(body)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Update(c.Request.Context(), flag, id, r)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, 200, out)
}
func (h *RuleHandler) Delete(c *gin.Context) {
	flag, id, err := ruleIDs(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	if err = h.service.Delete(c.Request.Context(), flag, id); err != nil {
		HandleError(c, err)
		return
	}
	c.Status(204)
}
