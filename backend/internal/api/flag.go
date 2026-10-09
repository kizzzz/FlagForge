package api

import (
	"io"

	"flagforge/internal/apperror"
	"flagforge/internal/dto"
	"flagforge/internal/service"
	"github.com/gin-gonic/gin"
)

type FlagHandler struct{ service *service.FlagService }

func NewFlagHandler(s *service.FlagService) *FlagHandler { return &FlagHandler{service: s} }
func ReadBody(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, apperror.Validation("body", "unable to read request body")
	}
	return body, nil
}
func flagID(c *gin.Context) (string, error) { id := c.Param("id"); return id, dto.ValidateID(id) }
func (h *FlagHandler) Create(c *gin.Context) {
	body, err := ReadBody(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	r, err := dto.DecodeCreateFlag(body)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Create(c.Request.Context(), r)
	if err != nil {
		HandleError(c, err)
		return
	}
	respondOK(c, 201, out)
}
func (h *FlagHandler) List(c *gin.Context) {
	r, err := dto.ParseListFlags(c.Request.URL.Query())
	if err != nil {
		HandleError(c, err)
		return
	}
	items, total, err := h.service.List(c.Request.Context(), r)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPage(c, items, r.Page, r.PageSize, total)
}
func (h *FlagHandler) Detail(c *gin.Context) {
	id, err := flagID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Detail(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}
	respondOK(c, 200, out)
}
func (h *FlagHandler) Update(c *gin.Context) {
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
	r, err := dto.DecodeUpdateFlag(body)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Update(c.Request.Context(), id, r)
	if err != nil {
		HandleError(c, err)
		return
	}
	respondOK(c, 200, out)
}
func (h *FlagHandler) Status(c *gin.Context) {
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
	r, err := dto.DecodeFlagStatus(body)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Status(c.Request.Context(), id, r)
	if err != nil {
		HandleError(c, err)
		return
	}
	respondOK(c, 200, out)
}
func (h *FlagHandler) History(c *gin.Context) {
	id, err := flagID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	r, err := dto.ParsePageQuery(c.Request.URL.Query())
	if err != nil {
		HandleError(c, err)
		return
	}
	items, total, err := h.service.History(c.Request.Context(), id, r)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPage(c, items, r.Page, r.PageSize, total)
}
