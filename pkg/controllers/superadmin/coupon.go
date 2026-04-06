package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateCoupon creates a new coupon
func CreateCoupon(c *gin.Context) {
	var req struct {
		Code          string  `json:"code" binding:"required"`
		Description   *string `json:"description"`
		RewardValue   string  `json:"rewardValue" binding:"required"`
		MinOrderValue float64 `json:"minOrderValue" binding:"required"`
		ValidFrom     string  `json:"validFrom" binding:"required"`
		ValidUntil    string  `json:"validUntil" binding:"required"`
		IsActive      *bool   `json:"isActive"`
		UsageLimit    *int    `json:"usageLimit"`
		OutletID      int     `json:"outletId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "code, rewardValue, minOrderValue, validFrom, and validUntil are required"})
		return
	}

	// Parse reward value (percentage)
	var parsedRewardValue float64
	if strings.HasSuffix(req.RewardValue, "%") {
		percentStr := strings.TrimSuffix(req.RewardValue, "%")
		percentage, err := strconv.ParseFloat(percentStr, 64)
		if err != nil || percentage <= 0 || percentage > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "rewardValue must be a valid percentage between 1% and 100%"})
			return
		}
		parsedRewardValue = percentage / 100
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"message": "rewardValue must be provided as a percentage (e.g., '10%')"})
		return
	}

	// Parse and validate dates
	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "validFrom must be in RFC3339 format (e.g., 2024-01-01T10:00:00Z)"})
		return
	}

	validUntil, err := time.Parse(time.RFC3339, req.ValidUntil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "validUntil must be in RFC3339 format (e.g., 2024-01-01T10:00:00Z)"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}
	usageLimit := 0
	if req.UsageLimit != nil {
		usageLimit = *req.UsageLimit
	}

	coupon := models.Coupon{
		Code:          req.Code,
		Description:   desc,
		RewardValue:   parsedRewardValue,
		MinOrderValue: req.MinOrderValue,
		ValidFrom:     validFrom,
		ValidUntil:    validUntil,
		IsActive:      isActive,
		UsageLimit:    usageLimit,
		UsedCount:     0,
		OutletID:      &req.OutletID,
	}

	if err := database.DB.Create(&coupon).Error; err != nil {
		// Check for duplicate key error
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "Coupon_code_key") {
			c.JSON(http.StatusConflict, gin.H{"message": "A coupon with this code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Coupon created successfully",
		"data":    coupon,
	})
}

// GetCoupons returns all coupons for an outlet
func GetCoupons(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	var err error

	var coupons []models.Coupon
	query := database.DB.Model(&models.Coupon{})

	if outletIDStr != "ALL" {
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
		query = query.Where(`"outletId" = ?`, outletID)
	}

	query.Find(&coupons)

	c.JSON(http.StatusOK, gin.H{
		"message": "Coupons fetched successfully",
		"data":    coupons,
	})
}

// DeleteCoupon deletes a coupon
func DeleteCoupon(c *gin.Context) {
	couponIDStr := c.Param("couponId")
	couponID, err := strconv.Atoi(couponIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Valid couponId is required"})
		return
	}

	result := database.DB.Delete(&models.Coupon{}, couponID)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Coupon not found"})
		return
	}

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error", "error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Coupon deleted successfully",
	})
}
