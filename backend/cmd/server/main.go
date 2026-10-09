package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable"
	}
	// The data/API tasks will use this configuration to initialize persistence.
	_ = databaseURL

	router := gin.Default()
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	if err := router.Run(":8787"); err != nil {
		log.Fatal(err)
	}
}
