package api

import (
	"log"

	"flagforge/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Printf("request panic: %v", recovered)
		respondError(c, 500, InternalError, "internal server error", nil)
	}))
	r.HandleMethodNotAllowed = true
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.NoRoute(func(c *gin.Context) { respondError(c, 404, ValidationError, "route not found", nil) })
	r.NoMethod(func(c *gin.Context) { respondError(c, 405, ValidationError, "method not allowed", nil) })
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	RegisterRoutes(r, db)
	return r
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	v1 := r.Group("/api/v1")
	h := NewFlagHandler(service.NewFlagService(db))
	v1.POST("/flags", h.Create)
	v1.GET("/flags", h.List)
	v1.GET("/flags/:id", h.Detail)
	v1.PATCH("/flags/:id", h.Update)
	v1.PATCH("/flags/:id/status", h.Status)
	v1.GET("/flags/:id/history", h.History)
	v1.POST("/flags/:id/copy", h.Copy)
	rules := NewRuleHandler(service.NewRuleService(db))
	v1.POST("/flags/:id/rules", rules.Create)
	v1.PATCH("/flags/:id/rules/:rid", rules.Update)
	v1.DELETE("/flags/:id/rules/:rid", rules.Delete)
	evaluator := NewEvaluatorHandler(service.NewEvaluatorService(db))
	v1.POST("/evaluate", evaluator.Evaluate)
	v1.POST("/evaluate/batch", evaluator.Batch)
}
