package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	driverpkg "github.com/mikios34/delivery-backend/driver"
	"github.com/mikios34/delivery-backend/models"
)

// RegisterRoutes registers driver endpoints on the provided router group.
func RegisterRoutes(rg *gin.RouterGroup, svc driverpkg.DriverService) {
	rg.GET("/", func(c *gin.Context) {
		drivers, err := svc.ListDrivers(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, drivers)
	})

	rg.POST("/", func(c *gin.Context) {
		var d models.Driver
		if err := c.ShouldBindJSON(&d); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		created, err := svc.CreateDriver(c, &d)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, created)
	})

	rg.GET("/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id64, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		d, err := svc.GetDriver(context.Background(), uint(id64))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, d)
	})

	rg.DELETE("/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id64, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := svc.DeleteDriver(context.Background(), uint(id64)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
