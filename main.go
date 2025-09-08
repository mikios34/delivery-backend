package main

import (
	"github.com/gin-gonic/gin"

	courierrepo "github.com/mikios34/delivery-backend/courier/repository"
	couriersvc "github.com/mikios34/delivery-backend/courier/service"
	api "github.com/mikios34/delivery-backend/handler"
)

func main() {

	db := setupDatabase()

	r := gin.Default()
	// setup driver repository + service (impl-style constructors)

	// setup courier repository + service
	courierRepo := courierrepo.NewGormCourierRepo(db)
	courierService := couriersvc.NewCourierService(courierRepo)
	courierHandler := api.NewCourierHandler(courierService)

	r.Use(gin.Recovery(), gin.Logger())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		v1.GET("/guaranty-options", courierHandler.ListGuarantyOptions())
		v1.POST("/couriers/register", courierHandler.RegisterCourier())
	}

	// r.POST("/users", func(c *gin.Context) {
	// 	var user models.User
	// 	if err := c.ShouldBindJSON(&user); err != nil {
	// 		c.JSON(400, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	if err := db.Create(&user).Error; err != nil {
	// 		c.JSON(500, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	c.JSON(201, user)
	// })

	r.Run() // listen and serve on 0.0.0.0:8080
}
