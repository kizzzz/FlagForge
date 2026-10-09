package main

import (
	"log"
	"os"
	"time"

	"flagforge/internal/api"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{NowFunc: func() time.Time { return time.Now().UTC() }})
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err = sqlDB.Ping(); err != nil {
		log.Fatal(err)
	}
	router := api.NewRouter(db)
	address := os.Getenv("SERVER_ADDR")
	if address == "" {
		address = ":8787"
	}
	if err := router.Run(address); err != nil {
		log.Fatal(err)
	}
}
