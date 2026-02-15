package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetProductsAndStocks fetches all products with their stock availability and user's remaining quota
func GetProductsAndStocks(c *gin.Context) {
	// Get user from context (set by AuthenticateToken middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User not found in request."})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid user data."})
		return
	}

	if user.OutletID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Outlet ID not found in request."})
		return
	}

	outletID := *user.OutletID

	// Fetch all products for the outlet
	var products []models.Product
	if err := database.DB.Where("\"outletId\" = ?", outletID).Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Calculate remaining quota for the user
	remainingQuota := 5 // Default quota
	userID := user.ID

	// Get today's date at midnight
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	var quota models.UserFreeQuota
	err := database.DB.Where("\"userId\" = ? AND \"consumptionDate\" = ?", userID, today).First(&quota).Error
	if err == nil {
		// Quota record exists, calculate remaining
		remainingQuota = 5 - quota.QuantityUsed
		if remainingQuota < 0 {
			remainingQuota = 0
		}
	}

	// Build response with product details
	productsWithDetails := make([]gin.H, 0, len(products))

	for _, product := range products {
		// Fetch inventory for this product
		var inventory models.Inventory
		inventoryExists := database.DB.Where("\"productId\" = ?", product.ID).First(&inventory).Error == nil

		// Get signed URL for image
		imageURL := ""
		if product.ImageURL != nil {
			url, _ := services.GetSignedURL(*product.ImageURL)
			imageURL = url
		}

		availableQuantity := 0
		isAvailable := false
		if inventoryExists {
			availableQuantity = inventory.Quantity
			isAvailable = inventory.Quantity > 0
		}

		productsWithDetails = append(productsWithDetails, gin.H{
			"id":                     product.ID,
			"name":                   product.Name,
			"description":            product.Description,
			"price":                  product.Price,
			"imageUrl":               imageURL,
			"outletId":               product.OutletID,
			"category":               product.Category,
			"minValue":               product.MinValue,
			"isVeg":                  product.IsVeg,
			"ratingSum30d":           product.RatingSum30d,
			"ratingCount30d":         product.RatingCount30d,
			"trendScore":             product.TrendScore,
			"ratingSumLifetime":      product.RatingSumLifetime,
			"ratingCountLifetime":    product.RatingCountLifetime,
			"averageRatingLifetime":  product.AverageRatingLifetime,
			"companyPaid":            product.CompanyPaid,
			"availableQuantity":      availableQuantity,
			"remainingQuota":         remainingQuota,
			"isAvailable":            isAvailable,
		})
	}

	c.JSON(http.StatusOK, gin.H{"products": productsWithDetails})
}

// GetCurrentQuota returns the user's current remaining free quota for today
func GetCurrentQuota(c *gin.Context) {
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

	// Get today's date at midnight
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// Check if quota record exists for today
	var quota models.UserFreeQuota
	err := database.DB.Where("user_id = ? AND consumption_date = ?", user.ID, today).First(&quota).Error

	remainingQuota := 5 // Default quota
	quantityUsed := 0

	if err == nil {
		// Quota record exists
		quantityUsed = quota.QuantityUsed
		remainingQuota = 5 - quantityUsed
		if remainingQuota < 0 {
			remainingQuota = 0
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"remainingQuota": remainingQuota,
		"quantityUsed":   quantityUsed,
		"totalQuota":     5,
	})
}

// GetAvailableDatesAndSlotsForCustomer returns available delivery dates and time slots
func GetAvailableDatesAndSlotsForCustomer(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	if outletIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Outlet ID is required"})
		return
	}

	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Outlet ID"})
		return
	}

	today := time.Now()
	next30Days := today.AddDate(0, 0, 30)

	var nonAvailable []models.OutletAvailability
	database.DB.Where("outlet_id = ? AND date >= ? AND date <= ?",
		outletID, today, next30Days).Find(&nonAvailable)

	allSlots := []string{"SLOT_11_12", "SLOT_12_13", "SLOT_13_14", "SLOT_14_15", "SLOT_15_16", "SLOT_16_17"}

	availableDates := []gin.H{}
	for d := today; d.Before(next30Days) || d.Equal(next30Days); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")

		// Find non-availability for this date
		var nonAvailEntry *models.OutletAvailability
		for i := range nonAvailable {
			if nonAvailable[i].Date.Format("2006-01-02") == dateStr {
				nonAvailEntry = &nonAvailable[i]
				break
			}
		}

		// Calculate available slots
		currentSlots := []string{}
		if nonAvailEntry != nil {
			for _, slot := range allSlots {
				isBlocked := false
				for _, blockedSlot := range nonAvailEntry.NonAvailableSlots {
					if slot == blockedSlot {
						isBlocked = true
						break
					}
				}
				if !isBlocked {
					currentSlots = append(currentSlots, slot)
				}
			}
		} else {
			currentSlots = allSlots
		}

		if len(currentSlots) > 0 {
			availableDates = append(availableDates, gin.H{
				"date":  dateStr,
				"slots": currentSlots,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Available dates and slots fetched",
		"data":    availableDates,
	})
}

// GetOutlets returns all active outlets
func GetOutlets(c *gin.Context) {
	var outlets []models.Outlet
	if err := database.DB.Where("is_active = ?", true).Find(&outlets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"outlets": outlets})
}
