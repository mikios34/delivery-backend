package main

import (
	"context"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	adminrepo "github.com/mikios34/delivery-backend/admin/repository"
	adminsvc "github.com/mikios34/delivery-backend/admin/service"
	authrepo "github.com/mikios34/delivery-backend/auth/repository"
	authsvc "github.com/mikios34/delivery-backend/auth/service"
	courierrepo "github.com/mikios34/delivery-backend/courier/repository"
	couriersvc "github.com/mikios34/delivery-backend/courier/service"
	customerrepo "github.com/mikios34/delivery-backend/customer/repository"
	customersvc "github.com/mikios34/delivery-backend/customer/service"
	dispatchsvc "github.com/mikios34/delivery-backend/dispatch"
	api "github.com/mikios34/delivery-backend/handler"
	mw "github.com/mikios34/delivery-backend/middleware"
	orderrepo "github.com/mikios34/delivery-backend/order/repository"
	ordersvc "github.com/mikios34/delivery-backend/order/service"
	realtime "github.com/mikios34/delivery-backend/realtime"
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

	// setup realtime hub
	hub := realtime.NewHub()
	wsHandler := api.NewWSHandler(hub).WithCourierLocationHandler(func(courierID string, lat, lng *float64) {
		if id, err := uuid.Parse(courierID); err == nil {
			_ = courierService.UpdateLocation(context.Background(), id, lat, lng)
		}
	})

	// setup order repository + service
	orderRepo := orderrepo.NewGormOrderRepo(db)
	orderService := ordersvc.NewOrderService(orderRepo)
	orderHandler := api.NewOrderHandler(orderService, dispatchsvc.New(orderRepo, courierRepo, hub))
	statusHandler := api.NewOrderStatusHandler(orderService)

	// setup dispatch service (with hub for notifications)
	dispatchService := dispatchsvc.New(orderRepo, courierRepo, hub)

	// background reassign ticker (every 20s, cutoff 20s)
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for range t.C {
			ctx := context.Background()

			// Only run cleanup if there are assigned orders
			count, err := orderRepo.CountAssignedOrders(ctx)
			if err != nil {
				continue // Skip this iteration if count fails
			}
			if count == 0 {
				continue // No assigned orders, skip expensive cleanup
			}

			cutoff := time.Now().Add(-20 * time.Second)
			_, _ = dispatchService.ReassignTimedOut(ctx, cutoff)
		}
	}()

	r.Use(gin.Recovery(), gin.Logger())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})


	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		v1.GET("/guaranty-options", courierHandler.ListGuarantyOptions())
		v1.POST("/couriers/register", courierHandler.RegisterCourier())
		v1.POST("/customers/register", customerHandler.RegisterCustomer())
		v1.POST("/admins/register", adminHandler.RegisterAdmin())
		v1.POST("/login", authHandler.Login())

		// order endpoints
		v1.GET("/order-types", mw.RequireAuth(), orderHandler.ListOrderTypes())
		v1.POST("/orders", mw.RequireAuth(), mw.RequireRoles("customer"), orderHandler.CreateOrder())

		// websocket endpoints
		courierWS := v1.Group("/ws/courier")
		courierWS.Use(mw.RequireAuth(), mw.RequireRoles("courier"))
		courierWS.GET("", wsHandler.CourierSocket())
	}

	// Example protected groups (not yet used by any specific endpoints):
	courierGroup := v1.Group("/courier")
	courierGroup.Use(mw.RequireAuth(), mw.RequireRoles("courier"))
	courierGroup.POST("/availability", courierHandler.SetAvailability())
	courierGroup.POST("/location", courierHandler.UpdateLocation())
	courierGroup.POST("/orders/accept", statusHandler.Accept())
	courierGroup.POST("/orders/decline", statusHandler.Decline())
	courierGroup.POST("/orders/arrived", statusHandler.Arrived())
	courierGroup.POST("/orders/picked", statusHandler.Picked())
	courierGroup.POST("/orders/delivered", statusHandler.Delivered())

	customerGroup := v1.Group("/customer")
	customerGroup.Use(mw.RequireAuth(), mw.RequireRoles("customer"))

	adminGroup := v1.Group("/admin")
	adminGroup.Use(mw.RequireAuth(), mw.RequireRoles("admin"))

	r.Run() // listen and serve on 0.0.0.0:8080
}
