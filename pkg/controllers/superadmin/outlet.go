package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AddOutlets creates a new outlet
func AddOutlets(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		Address    string `json:"address" binding:"required"`
		Phone      string `json:"phone" binding:"required"`
		Email      string `json:"email" binding:"required"`
		StaffCount int    `json:"staffCount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide all outlet details"})
		return
	}

	// Check existing outlet
	var existing models.Outlet
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Outlet already exists"})
		return
	}

	// Create outlet
	outlet := models.Outlet{
		Name:       req.Name,
		Address:    &req.Address,
		Phone:      &req.Phone,
		Email:      &req.Email,
		StaffCount: req.StaffCount,
	}

	if err := database.DB.Create(&outlet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    outlet,
		"message": "Outlet created successfully",
	})
}

// GetOutlets returns all outlets
func GetOutlets(c *gin.Context) {
	var outlets []models.Outlet
	database.DB.Find(&outlets)

	// DEBUG: Log what we fetched
	fmt.Printf("[DEBUG] GetOutlets: Found %d outlets\n", len(outlets))
	for i, o := range outlets {
		fmt.Printf("[DEBUG]   Outlet[%d]: ID=%d, Name='%s', Address=%v, Email=%v\n", i, o.ID, o.Name, o.Address, o.Email)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    outlets,
		"message": "Outlets fetched successfully",
	})
}

// RemoveOutlets deletes an outlet
func RemoveOutlets(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide OutletId to delete"})
		return
	}

	if err := database.DB.Delete(&models.Outlet{}, outletID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Outlet deleted successfully",
	})
}
