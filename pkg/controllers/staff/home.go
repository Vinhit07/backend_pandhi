package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetHomeDetails returns dashboard overview statistics
func GetHomeDetails(c *gin.Context) {
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
	
	log.Printf("GetHomeDetails called for outletID: %d", outletID)

	// Calculate order stats for all delivered orders
	var statsResult struct {
		TotalRevenue float64
		OrderCount   int64
	}
	database.DB.Model(&models.Order{}).
		Select("COALESCE(SUM(\"totalAmount\"), 0) as total_revenue, COUNT(*) as order_count").
		Where("\"outletId\" = ? AND status IN ?",
			outletID,
			[]models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered},
		).
		Scan(&statsResult)

	// Debug: Check total orders in DB
	var totalOrderCount int64
	database.DB.Model(&models.Order{}).Where("\"outletId\" = ?", outletID).Count(&totalOrderCount)
	log.Printf("Dashboard Debug - OutletID: %d, Total Orders in DB: %d, Delivered Orders: %d, Revenue: %.2f", 
		outletID, totalOrderCount, statsResult.OrderCount, statsResult.TotalRevenue)

	totalRevenue := statsResult.TotalRevenue

	// Get order counts by type for all delivered orders
	type OrderTypeCount struct {
		Type  models.OrderType
		Count int64
	}
	var typeCounts []OrderTypeCount
	database.DB.Model(&models.Order{}).
		Select("type, COUNT(*) as count").
		Where("\"outletId\" = ? AND status IN ?",
			outletID,
			[]models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered},
		).
		Group("type").
		Scan(&typeCounts)

	log.Printf("Dashboard Stats - Type Counts: %+v", typeCounts)

	appOrders := int64(0)
	manualOrders := int64(0)
	for _, tc := range typeCounts {
		if tc.Type == models.OrderTypeApp {
			appOrders = tc.Count
		}
		if tc.Type == models.OrderTypeManual {
			manualOrders = tc.Count
		}
	}

	// Get delivery slot counts for peak time (all time, not just today)
	type SlotCount struct {
		DeliverySlot string
		Count        int64
	}
	var slotCounts []SlotCount
	database.DB.Model(&models.Order{}).
		Select("\"deliverySlot\", COUNT(*) as count").
		Where("\"outletId\" = ? AND status IN ? AND \"deliverySlot\" IS NOT NULL AND \"deliverySlot\" != ''",
			outletID,
			[]models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered},
		).
		Group("\"deliverySlot\"").
		Order("count DESC").
		Limit(1).
		Scan(&slotCounts)

	// Find peak slot
	var peakSlot *string
	if len(slotCounts) > 0 {
		peakSlot = &slotCounts[0].DeliverySlot
	}

	// Get best seller
	type BestSellerResult struct {
		ProductID     int
		TotalQuantity int
	}

	var bestSellerResult BestSellerResult
	err := database.DB.Model(&models.OrderItem{}).
		Select("\"productId\", SUM(quantity) as total_quantity").
		Joins("JOIN \"Order\" ON \"Order\".id = \"OrderItem\".\"orderId\"").
		Where("\"Order\".\"outletId\" = ? AND \"Order\".status IN ?", outletID, []models.OrderStatus{
			models.OrderStatusDelivered,
			models.OrderStatusPartiallyDelivered,
		}).
		Group("\"productId\"").
		Order("total_quantity DESC").
		Limit(1).
		Scan(&bestSellerResult).Error

	var bestSellerProduct *gin.H
	if err == nil && bestSellerResult.ProductID > 0 {
		var product models.Product
		if err := database.DB.First(&product, bestSellerResult.ProductID).Error; err == nil {
			bestSellerProduct = &gin.H{
				"id":           product.ID,
				"name":         product.Name,
				"imageUrl":     product.ImageURL,
				"quantitySold": bestSellerResult.TotalQuantity,
			}
		}
	}

	// Total wallet recharge
	var totalRechargedAmount float64
	database.DB.Model(&models.Wallet{}).
		Select("COALESCE(SUM(\"totalRecharged\"), 0)").
		Joins("JOIN \"CustomerDetails\" ON \"CustomerDetails\".id = \"Wallet\".\"customerId\"").
		Joins("JOIN \"User\" ON \"User\".id = \"CustomerDetails\".\"userId\"").
		Where("\"User\".\"outletId\" = ?", outletID).
		Scan(&totalRechargedAmount)

	// Low stock products
	var lowStock []models.Inventory
	database.DB.Where(map[string]interface{}{"outletId": outletID}).
		Where("quantity < threshold").
		Preload("Product").
		Find(&lowStock)

	lowStockProducts := make([]gin.H, len(lowStock))
	for i, inv := range lowStock {
		lowStockProducts[i] = gin.H{
			"productId": inv.Product.ID,
			"name":      inv.Product.Name,
			"imageUrl":  inv.Product.ImageURL,
			"quantity":  inv.Quantity,
			"threshold": inv.Threshold,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"totalRevenue":         totalRevenue,
		"appOrders":            appOrders,
		"manualOrders":         manualOrders,
		"peakSlot":             peakSlot,
		"bestSellerProduct":    bestSellerProduct,
		"totalRechargedAmount": totalRechargedAmount,
		"lowStockProducts":     lowStockProducts,
	})
}

// RecentOrders returns paginated recent orders for an outlet
func RecentOrders(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status") // Get status filter (e.g., "pending", "delivered", etc.)
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	skip := (page - 1) * limit

	// Build query with optional status filter
	query := database.DB.Model(&models.Order{}).Where("\"outletId\" = ?", outletID)
	if status != "" {
		// Convert status to uppercase to match OrderStatus enum
		query = query.Where("UPPER(status) = ?", strings.ToUpper(status))
	}

	// Count total orders
	var totalOrders int64
	query.Count(&totalOrders)

	// Fetch orders
	var orders []models.Order
	query.Preload("Customer.User").
		Preload("Items.Product").
		Order("\"createdAt\" DESC").
		Limit(limit).
		Offset(skip).
		Find(&orders)

	formatted := make([]gin.H, len(orders))
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
			unitPrice := item.UnitPrice
			if unitPrice == 0 {
				unitPrice = item.Product.Price
			}
			items[j] = gin.H{
				"name":      item.Product.Name,
				"quantity":  item.Quantity,
				"unitPrice": unitPrice,
			}
		}

		formatted[i] = gin.H{
			"billNumber":   order.ID,
			"customerName": customerName,
			"orderType":    order.Type,
			"paymentMode":  order.PaymentMethod,
			"status":       order.Status,
			"items":        items,
			"totalAmount":  order.TotalAmount,
			"createdAt":    order.CreatedAt,
			"deliveryDate": order.DeliveryDate,
			"deliverySlot": order.DeliverySlot,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Recent orders fetched successfully",
		"orders":      formatted,
		"total":       totalOrders,
		"currentPage": page,
		"totalPages":  int(math.Ceil(float64(totalOrders) / float64(limit))),
	})
}

// GetTicketsCount returns ticket count for staff's outlet
func GetTicketsCount(c *gin.Context) {
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

	var ticketCount int64
	database.DB.Model(&models.Ticket{}).
		Joins("JOIN \"CustomerDetails\" ON \"CustomerDetails\".id = \"Ticket\".\"customerId\"").
		Joins("JOIN \"User\" ON \"User\".id = \"CustomerDetails\".\"userId\"").
		Where("\"User\".\"outletId\" = ?", user.OutletID).
		Count(&ticketCount)

	c.JSON(http.StatusOK, gin.H{
		"count":   ticketCount,
		"message": "Ticket count fetched successfully",
	})
}

// GetOrder returns single order details
func GetOrder(c *gin.Context) {
	orderIDStr := c.Param("orderId")
	outletIDStr := c.Param("outletId")

	orderID, err1 := strconv.Atoi(orderIDStr)
	outletID, err2 := strconv.Atoi(outletIDStr)

	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide valid orderId and outletId"})
		return
	}

	var order models.Order
	if err := database.DB.
		Where("id = ? AND \"outletId\" = ?", orderID, outletID).
		Preload("Customer.User").
		Preload("Outlet").
		Preload("Items.Product").
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Order not found or does not belong to this outlet"})
		return
	}

	customerName := "Guest"
	if order.Customer != nil && order.Customer.User.ID > 0 {
		customerName = order.Customer.User.Name
	}

	items := make([]gin.H, len(order.Items))
	for i, item := range order.Items {
		unitPrice := item.UnitPrice
		if unitPrice == 0 {
			unitPrice = item.Product.Price
		}
		items[i] = gin.H{
			"id":                 item.ID,
			"productName":        item.Product.Name,
			"productDescription": item.Product.Description,
			"quantity":           item.Quantity,
			"unitPrice":          unitPrice,
			"totalPrice":         float64(item.Quantity) * unitPrice,
			"itemStatus":         item.Status,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"order": gin.H{
			"orderId":      order.ID,
			"customerName": customerName,
			"outletName":   order.Outlet.Name,
			"orderStatus":  order.Status,
			"totalPrice":   order.TotalAmount,
			"createdAt":    order.CreatedAt,
			"items":        items,
		},
	})
}

// UpdateOrder updates order status with stock management and refunds
func UpdateOrder(c *gin.Context) {
	var req struct {
		OrderID      int    `json:"orderId" binding:"required"`
		OrderItemIDs []int  `json:"orderItemIds"`
		Status       string `json:"status" binding:"required"`
		OutletID     int    `json:"outletId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide orderId, status, and outletId"})
		return
	}

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

	// Verify staff outlet matches request outlet
	if user.OutletID != nil && *user.OutletID != req.OutletID {
		c.JSON(http.StatusForbidden, gin.H{"message": "You can only update orders for your assigned outlet"})
		return
	}

	// Fetch order with relationships
	var order models.Order
	if err := database.DB.
		Where("id = ? AND \"outletId\" = ?", req.OrderID, req.OutletID).
		Preload("Items").
		Preload("Customer").
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Order not found for this outlet"})
		return
	}

	// === CANCELLED ===
	if req.Status == "CANCELLED" {
		if order.Status != string(models.OrderStatusPending) {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": fmt.Sprintf("Cannot cancel order. Order status is %s", order.Status),
			})
			return
		}

		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Update order status
			tx.Model(&order).Updates(map[string]interface{}{
				"status":       models.OrderStatusCancelled,
				"deliveredAt": nil,
			})

			// Restore stock for all items
			for _, item := range order.Items {
				tx.Model(&models.Inventory{}).
					Where("\"productId\" = ?", item.ProductID).
					Update("quantity", gorm.Expr("quantity + ?", item.Quantity))

				tx.Create(&models.StockHistory{
					ProductID: item.ProductID,
					OutletID:  order.OutletID,
					Quantity:  item.Quantity,
					Action:    models.StockActionAdd,
				})
			}

			// Refund logic for APP orders
			if order.Type == models.OrderTypeApp && order.CustomerID != nil {
				var wallet models.Wallet
				if err := tx.Where("\"customerId\" = ?", *order.CustomerID).First(&wallet).Error; err == nil {
					now := time.Now()
					tx.Model(&wallet).Updates(map[string]interface{}{
						"balance": gorm.Expr("balance + ?", order.TotalAmount),
					})

					tx.Create(&models.WalletTransaction{
						WalletID: wallet.ID,
						Amount:   order.TotalAmount,
						Method:   order.PaymentMethod,
						Status:   models.WalletTransTypeRecharge,
					})
					_ = now
				}
			}

			// Refund coupon
			var couponUsage models.CouponUsage
			if tx.Where("\"orderId\" = ?", req.OrderID).First(&couponUsage).Error == nil {
				tx.Delete(&couponUsage)
				tx.Model(&models.Coupon{}).
					Where("id = ?", couponUsage.CouponID).
					Update("used_count", gorm.Expr("used_count - 1"))
			}

			// Restore quota for free items
			totalFreeQty := 0
			for _, item := range order.Items {
				if item.FreeQuantity > 0 {
					totalFreeQty += item.FreeQuantity
				}
			}

			if totalFreeQty > 0 && order.Customer != nil {
				today := time.Now().Truncate(24 * time.Hour)
				var quota models.UserFreeQuota
				if tx.Where("\"userId\" = ? AND \"consumptionDate\" = ?", order.Customer.UserID, today).
					First(&quota).Error == nil {
					if quota.QuantityUsed >= totalFreeQty {
						tx.Model(&quota).Update("\"quantityUsed\"", gorm.Expr("\"quantityUsed\" - ?", totalFreeQty))
					}
				}
			}

			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to cancel order", "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Order cancelled and stock updated"})
		return
	}

	// === DELIVERED ===
	if req.Status == "DELIVERED" {
		if order.Status == string(models.OrderStatusCancelled) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Cannot mark a cancelled order as delivered."})
			return
		}

		err := database.DB.Transaction(func(tx *gorm.DB) error {
			tx.Model(&models.OrderItem{}).
				Where("\"orderId\" = ? AND status != ?", order.ID, models.OrderItemStatusDelivered).
				Update("status", models.OrderItemStatusDelivered)

			now := time.Now()
			tx.Model(&order).Updates(map[string]interface{}{
				"status":       models.OrderStatusDelivered,
				"deliveredAt": &now,
			})
			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update order"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "All items and order marked DELIVERED"})
		return
	}

	// === PARTIALLY_DELIVERED ===
	if req.Status == "PARTIALLY_DELIVERED" {
		if len(req.OrderItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Provide at least one orderItemId to deliver"})
			return
		}

		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Update selected items
			tx.Model(&models.OrderItem{}).
				Where("id IN ?", req.OrderItemIDs).
				Update("status", models.OrderItemStatusDelivered)

			// Check if all items delivered
			var updatedOrder models.Order
			tx.Preload("Items").First(&updatedOrder, order.ID)

			allDelivered := true
			for _, item := range updatedOrder.Items {
				if item.Status != models.OrderItemStatusDelivered {
					allDelivered = false
					break
				}
			}

			status := models.OrderStatusPartiallyDelivered
			var deliveredAt *time.Time
			if allDelivered {
				status = models.OrderStatusDelivered
				now := time.Now()
				deliveredAt = &now
			}

			tx.Model(&order).Updates(map[string]interface{}{
				"status":       status,
				"deliveredAt": deliveredAt,
			})
			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update order"})
			return
		}

		message := "Selected items delivered, order marked PARTIALLY_DELIVERED"
		if len(req.OrderItemIDs) == 1 {
			message = "Order marked PARTIALLY_DELIVERED; one item delivered"
		}

		c.JSON(http.StatusOK, gin.H{"message": message})
		return
	}

	// === PARTIAL_CANCEL ===
	if req.Status == "PARTIAL_CANCEL" {
		if order.Status != string(models.OrderStatusPartiallyDelivered) {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": fmt.Sprintf("Cannot partially cancel order. Order status is %s", order.Status),
			})
			return
		}

		undeliveredItems := []models.OrderItem{}
		for _, item := range order.Items {
			if item.Status == models.OrderItemStatusNotDelivered {
				undeliveredItems = append(undeliveredItems, item)
			}
		}

		if len(undeliveredItems) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "No undelivered items to cancel"})
			return
		}

		refundAmount := 0.0
		for _, item := range undeliveredItems {
			refundAmount += float64(item.Quantity) * item.UnitPrice
		}

		err := database.DB.Transaction(func(tx *gorm.DB) error {
			now := time.Now()
			tx.Model(&order).Updates(map[string]interface{}{
				"status":       models.OrderStatusDelivered,
				"deliveredAt": &now,
			})

			// Restore stock
			for _, item := range undeliveredItems {
				tx.Model(&models.Inventory{}).
					Where("\"productId\" = ?", item.ProductID).
					Update("quantity", gorm.Expr("quantity + ?", item.Quantity))

				tx.Create(&models.StockHistory{
					ProductID: item.ProductID,
					OutletID:  order.OutletID,
					Quantity:  item.Quantity,
					Action:    models.StockActionAdd,
				})
			}

			// Refund for APP orders
			if order.Type == models.OrderTypeApp && order.CustomerID != nil && refundAmount > 0 {
				var wallet models.Wallet
				if err := tx.Where("\"customerId\" = ?", *order.CustomerID).First(&wallet).Error; err == nil {
					tx.Model(&wallet).Update("balance", gorm.Expr("balance + ?", refundAmount))

					tx.Create(&models.WalletTransaction{
						WalletID: wallet.ID,
						Amount:   refundAmount,
						Method:   order.PaymentMethod,
						Status:   models.WalletTransTypeRecharge,
					})
				}
			}

			// Restore quota
			totalFreeQty := 0
			for _, item := range undeliveredItems {
				if item.FreeQuantity > 0 {
					totalFreeQty += item.FreeQuantity
				}
			}

			if totalFreeQty > 0 && order.Customer != nil {
				today := time.Now().Truncate(24 * time.Hour)
				var quota models.UserFreeQuota
				if tx.Where("\"userId\" = ? AND \"consumptionDate\" = ?", order.Customer.UserID, today).
					First(&quota).Error == nil {
					if quota.QuantityUsed >= totalFreeQty {
						tx.Model(&quota).Update("\"quantityUsed\"", gorm.Expr("\"quantityUsed\" - ?", totalFreeQty))
					}
				}
			}

			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to cancel items"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Undelivered items cancelled, stock restored, and ₹%.2f refunded", refundAmount),
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid status value"})
}
