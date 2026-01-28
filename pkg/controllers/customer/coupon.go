package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetCoupons retrieves all available (unused by customer) coupons for an outlet
func GetCoupons(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "User not found."})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid user data."})
		return
	}

	// Get current time
	currentTime := time.Now()

	// Fetch coupons
	var coupons []models.Coupon
	if err := database.DB.
		Where("outlet_id = ? AND is_active = ? AND valid_from <= ? AND valid_until >= ?",
			outletID, true, currentTime, currentTime).
		Preload("Usages", "user_id = ?", user.ID).
		Order("created_at DESC").
		Find(&coupons).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch coupons", "error": err.Error()})
		return
	}

	// Filter unused coupons
	var unusedCoupons []gin.H
	for _, coupon := range coupons {
		if len(coupon.Usages) == 0 {
			unusedCoupons = append(unusedCoupons, gin.H{
				"id":                     coupon.ID,
				"code":                   coupon.Code,
				"description":            coupon.Description,
				"rewardValue":            coupon.RewardValue,
				"minOrderValue":          coupon.MinOrderValue,
				"validFrom":              coupon.ValidFrom,
				"validUntil":             coupon.ValidUntil,
				"isActive":               coupon.IsActive,
				"usageLimit":             coupon.UsageLimit,
				"usedCount":              coupon.UsedCount,
				"usageType":              coupon.UsageType,
				"outletId":               coupon.OutletID,
				"createdAt":              coupon.CreatedAt,
				"isCurrentlyValid":       true,
				"validFromIST":           formatDateForIST(coupon.ValidFrom),
				"validUntilIST":          formatDateForIST(coupon.ValidUntil),
				"isUsedByCustomer":       false,
				"remainingUses":          coupon.UsageLimit - coupon.UsedCount,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Available coupons fetched successfully",
		"coupons":        unusedCoupons,
		"totalAvailable": len(unusedCoupons),
		"currentTimeIST": formatDateForIST(currentTime),
		"timezone":       "IST (UTC+5:30)",
		"note":           "Only unused coupons by this customer are returned",
	})
}

// ApplyCoupon validates and applies a coupon to the cart
func ApplyCoupon(c *gin.Context) {
	var req struct {
		Code         string  `json:"code" binding:"required"`
		CurrentTotal float64 `json:"currentTotal" binding:"required"`
		OutletID     int     `json:"outletId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing or invalid fields: code, currentTotal, and outletId are required"})
		return
	}

	if req.CurrentTotal < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing or invalid fields: code, currentTotal, and outletId are required"})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "User not found."})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid user data."})
		return
	}

	// Get customer details with cart
	var customer models.CustomerDetails
	if err := database.DB.
		Preload("Cart.Items.Product").
		Where("user_id = ?", user.ID).
		First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	if customer.Cart == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Cart not found for customer"})
		return
	}

	// Calculate cart total
	var calculatedTotal float64
	for _, item := range customer.Cart.Items {
		calculatedTotal += float64(item.Quantity) * item.Product.Price
	}

	if math.Abs(calculatedTotal-req.CurrentTotal) > 0.01 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message":         "Provided currentTotal does not match calculated cart total",
			"calculatedTotal": calculatedTotal,
			"providedTotal":   req.CurrentTotal,
		})
		return
	}

	// Fetch coupon
	var coupon models.Coupon
	if err := database.DB.Where("code = ?", req.Code).First(&coupon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Invalid or inactive coupon"})
		return
	}

	if !coupon.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"message": "Invalid or inactive coupon"})
		return
	}

	// Check coupon validity
	currentTime := time.Now()
	if currentTime.Before(coupon.ValidFrom) || currentTime.After(coupon.ValidUntil) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message":           "Coupon is not valid for the current date and time",
			"currentTimeIST":    formatDateForIST(currentTime),
			"couponValidFrom":   formatDateForIST(coupon.ValidFrom),
			"couponValidUntil":  formatDateForIST(coupon.ValidUntil),
			"timezone":          "IST (UTC+5:30)",
		})
		return
	}

	// Check outlet
	if coupon.OutletID != nil && *coupon.OutletID != req.OutletID {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Coupon is not valid for the selected outlet"})
		return
	}

	// Check if already used
	var existingUsage models.CouponUsage
	if err := database.DB.
		Where("user_id = ? AND coupon_id = ?", user.ID, coupon.ID).
		First(&existingUsage).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Coupon already used by this customer"})
		return
	}

	// Check usage limit
	if coupon.UsedCount >= coupon.UsageLimit {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Coupon usage limit reached"})
		return
	}

	// Check minimum order value
	if req.CurrentTotal < coupon.MinOrderValue {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Minimum order value of ₹" + strconv.FormatFloat(coupon.MinOrderValue, 'f', 2, 64) + " required. Your current order value is ₹" + strconv.FormatFloat(req.CurrentTotal, 'f', 2, 64),
		})
		return
	}

	// Calculate discount
	var discount float64
	if coupon.RewardValue > 0 {
		if coupon.RewardValue <= 1 {
			// Percentage discount (0.25 = 25%)
			discount = req.CurrentTotal * coupon.RewardValue
		} else if coupon.RewardValue <= req.CurrentTotal {
			// Fixed amount discount
			discount = coupon.RewardValue
		} else {
			// Fixed amount exceeds total, cap at total
			discount = req.CurrentTotal
		}
	}

	// Calculate final amount (minimum 0)
	totalAfterDiscount := math.Max(0, req.CurrentTotal-discount)

	c.JSON(http.StatusOK, gin.H{
		"message":            "Coupon applied successfully",
		"discount":           discount,
		"totalAfterDiscount": totalAfterDiscount,
	})
}

// formatDateForIST formats a time in IST (UTC+5:30)
func formatDateForIST(t time.Time) string {
	// Add 5 hours 30 minutes to UTC time
	ist := t.Add(5*time.Hour + 30*time.Minute)
	return ist.Format("2006-01-02 15:04:05")
}
