package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetOrderHistory returns historical orders based on query filters
func GetOrderHistory(c *gin.Context) {
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

	// Get outlet ID from user
	if user.OutletID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Staff not assigned to outlet"})
		return
	}

	outletID := *user.OutletID

	// Get query parameters
	status := c.Query("status")
	orderType := c.Query("type")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	query := database.DB.Where("\"outletId\" = ?", outletID)

	// Apply filters
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if orderType != "" {
		query = query.Where("type = ?", orderType)
	}
	if startDate != "" {
		query = query.Where("\"createdAt\" >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("\"createdAt\" <= ?", endDate)
	}

	var orders []models.Order
	query.Preload("Customer.User").
		Preload("Items.Product").
		Preload("Outlet").
		Order("\"createdAt\" DESC").
		Find(&orders)

	formattedOrders := make([]gin.H, len(orders))
	for i, order := range orders {
		customerName := "Walk-in Customer"
		if order.Customer != nil {
			database.DB.Preload("User").First(&order.Customer, order.Customer.ID)
			if order.Customer.User.ID > 0 {
				customerName = order.Customer.User.Name
			}
		}

		items := make([]gin.H, len(order.Items))
		for j, item := range order.Items {
			items[j] = gin.H{
				"id":          item.ID,
				"productName": item.Product.Name,
				"quantity":    item.Quantity,
				"unitPrice":   func() float64 {
					if item.UnitPrice == 0 {
						return item.Product.Price
					}
					return item.UnitPrice
				}(),
				"status":      item.Status,
			}
		}

		formattedOrders[i] = gin.H{
			"id":            order.ID,
			"orderNumber":   "#ORD-" + strconv.Itoa(order.ID),
			"customerName":  customerName,
			"totalAmount":   order.TotalAmount,
			"paymentMethod": order.PaymentMethod,
			"status":        order.Status,
			"type":          order.Type,
			"deliveryDate":  order.DeliveryDate,
			"deliverySlot":  order.DeliverySlot,
			"deliveredAt":   order.DeliveredAt,
			"createdAt":     order.CreatedAt,
			"items":         items,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order history retrieved",
		"orders":  formattedOrders,
		"count":   len(formattedOrders),
	})
}

// GetAvailableDatesAndSlotsForStaff returns available delivery dates and slots
func GetAvailableDatesAndSlotsForStaff(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
	}

	// Verify outlet exists
	var outlet models.Outlet
	if err := database.DB.First(&outlet, outletID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Outlet not found"})
		return
	}

	today := time.Now()
	next30Days := today.AddDate(0, 0, 30)

	var nonAvailable []models.OutletAvailability
	database.DB.Where("\"outletId\" = ? AND date >= ? AND date <= ?",
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
				"date":           dateStr,
				"availableSlots": currentSlots,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Available dates and slots fetched",
		"data":    availableDates,
	})
}
