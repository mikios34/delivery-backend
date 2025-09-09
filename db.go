package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mikios34/delivery-backend/entity"
)

const (
	host     = "aws-1-eu-north-1.pooler.supabase.com"
	port     = 5432
	user     = "postgres.zxceeyortveyafherfwk"
	password = "mikios34@yahoo"
	dbname   = "postgres"
)

func setupDatabase() *gorm.DB {

	dsn := fmt.Sprintf(
		"host=%s user=%s password='%s' dbname=%s port=%d sslmode=require",
		host, user, password, dbname, port,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// Ensure required extensions for UUID are present (if using Postgres with uuid_generate_v4)
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Println("warning: failed to ensure uuid-ossp extension:", err)
	}

	// Auto-migrate courier-related tables
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.GuarantyOption{},
		&entity.Courier{},
		&entity.GuarantyPayment{},
		&entity.Customer{},
		&entity.Admin{},
		&entity.OrderType{},
		&entity.Order{},
	); err != nil {
		log.Fatal("failed to run migrations:", err)
	}
	return db
}
