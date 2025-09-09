package main

import (
	"github.com/gin-gonic/gin"

	adminrepo "github.com/mikios34/delivery-backend/admin/repository"
	adminsvc "github.com/mikios34/delivery-backend/admin/service"
	authrepo "github.com/mikios34/delivery-backend/auth/repository"
	authsvc "github.com/mikios34/delivery-backend/auth/service"
	courierrepo "github.com/mikios34/delivery-backend/courier/repository"
	couriersvc "github.com/mikios34/delivery-backend/courier/service"
	customerrepo "github.com/mikios34/delivery-backend/customer/repository"
	customersvc "github.com/mikios34/delivery-backend/customer/service"
	api "github.com/mikios34/delivery-backend/handler"
	mw "github.com/mikios34/delivery-backend/middleware"
)

func main() {

	db := setupDatabase()

	r := gin.Default()
	// setup driver repository + service (impl-style constructors)

	// setup courier repository + service
	courierRepo := courierrepo.NewGormCourierRepo(db)
	courierService := couriersvc.NewCourierService(courierRepo)
	courierHandler := api.NewCourierHandler(courierService)

	// setup customer repository + service
	customerRepo := customerrepo.NewGormCustomerRepo(db)
	customerService := customersvc.NewCustomerService(customerRepo)
	customerHandler := api.NewCustomerHandler(customerService)

	// setup admin repository + service
	adminRepo := adminrepo.NewGormAdminRepo(db)
	adminService := adminsvc.NewAdminService(adminRepo)
	adminHandler := api.NewAdminHandler(adminService)

	// setup auth repository + service
	authRepo := authrepo.NewGormAuthRepo(db)
	authService := authsvc.NewAuthService(authRepo)
	authHandler := api.NewAuthHandler(authService)

	r.Use(gin.Recovery(), gin.Logger())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		v1.GET("/guaranty-options", mw.RequireAuth(), courierHandler.ListGuarantyOptions())
		v1.POST("/couriers/register", courierHandler.RegisterCourier())
		v1.POST("/customers/register", customerHandler.RegisterCustomer())
		v1.POST("/admins/register", adminHandler.RegisterAdmin())
		v1.POST("/login", authHandler.Login())
	}

	// Example protected groups (not yet used by any specific endpoints):
	courierGroup := v1.Group("/courier")
	courierGroup.Use(mw.RequireAuth(), mw.RequireRoles("courier"))
	courierGroup.POST("/availability", courierHandler.SetAvailability())
	courierGroup.POST("/location", courierHandler.UpdateLocation())

	customerGroup := v1.Group("/customer")
	customerGroup.Use(mw.RequireAuth(), mw.RequireRoles("customer"))

	adminGroup := v1.Group("/admin")
	adminGroup.Use(mw.RequireAuth(), mw.RequireRoles("admin"))

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
