package api

import "github.com/gin-gonic/gin"

func respondError(c *gin.Context, status int, code, message string, details any) {
	if details == nil {
		details = gin.H{}
	}
	c.AbortWithStatusJSON(status, ErrorEnvelope{Error: &Error{Code: code, Message: message, Details: details}})
}
func respondOK(c *gin.Context, status int, value any) { c.JSON(status, value) }
func RespondError(c *gin.Context, status int, code, message string, details any) {
	respondError(c, status, code, message, details)
}
func RespondOK(c *gin.Context, status int, value any) { respondOK(c, status, value) }

type PageResponse struct {
	Items    any   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func RespondPage(c *gin.Context, items any, page, pageSize int, total int64) {
	respondOK(c, 200, PageResponse{Items: items, Page: page, PageSize: pageSize, Total: total})
}
