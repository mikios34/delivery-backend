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

	// Auto-migrate tables
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.GuarantyOption{},
		&entity.Courier{},
		&entity.GuarantyPayment{},
		&entity.Customer{},
		&entity.Admin{},
		&entity.OrderType{},
		&entity.Order{},
		&entity.OrderAssignmentAttempt{},
		&entity.VehicleTypeConfig{}, // pricing table: vehicle_types
	); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

	// Optional: seed default vehicle types if none exist
	if err := seedVehicleTypes(db); err != nil {
		log.Println("warning: failed to seed vehicle types:", err)
	}
	return db
}

// seedVehicleTypes inserts a few default vehicle types with reasonable pricing if the table is empty.
func seedVehicleTypes(db *gorm.DB) error {
	var count int64
	if err := db.Model(&entity.VehicleTypeConfig{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	seed := []entity.VehicleTypeConfig{
		{Code: "bike", Name: "Bike", Active: true, BaseFare: 20, PerKm: 10, PerMinute: 0, AvgSpeedKmh: 16, MinimumFare: 35, BookingFee: 0},
		{Code: "motorbike", Name: "Motorbike", Active: true, BaseFare: 25, PerKm: 12, PerMinute: 1.5, AvgSpeedKmh: 25, MinimumFare: 40, BookingFee: 0},
		{Code: "car", Name: "Car", Active: true, BaseFare: 30, PerKm: 15, PerMinute: 2.0, AvgSpeedKmh: 25, MinimumFare: 50, BookingFee: 0},
		// Combined public transport/taxi option
		{Code: "transport", Name: "Transport (Taxi/Bus/Train)", Active: true, BaseFare: 35, PerKm: 18, PerMinute: 2.5, AvgSpeedKmh: 22, MinimumFare: 60, BookingFee: 0},
	}
	return db.Create(&seed).Error
}
