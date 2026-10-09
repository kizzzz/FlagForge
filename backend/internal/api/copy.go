package api

import (
	"flagforge/internal/dto"
	"github.com/gin-gonic/gin"
)

func (h *FlagHandler) Copy(c *gin.Context) {
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
	r, err := dto.DecodeCopyFlag(body)
	if err != nil {
		HandleError(c, err)
		return
	}
	out, err := h.service.Copy(c.Request.Context(), id, r)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, 201, out)
}
