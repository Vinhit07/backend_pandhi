package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"fmt"
	"math"
	"net/http"
	"strconv"
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

	// Calculate order stats
	type OrderStat struct {
		Type         models.OrderType
		DeliverySlot *models.DeliverySlot
		Count        int64
		TotalAmount  float64
	}

	var orderStats []OrderStat
	database.DB.Model(&models.Order{}).
		Select("type, delivery_slot, COUNT(*) as count, COALESCE(SUM(total_amount), 0) as total_amount").
		Where("outlet_id = ? AND status IN ?", outletID, []models.OrderStatus{
			models.OrderStatusDelivered,
			models.OrderStatusPartiallyDelivered,
		}).
		Group("type, delivery_slot").
		Scan(&orderStats)

	totalRevenue := 0.0
	appOrders := int64(0)
	manualOrders := int64(0)
	slotCounts := make(map[string]int64)

	for _, stat := range orderStats {
		totalRevenue += stat.TotalAmount

		if stat.Type == models.OrderTypeApp {
			appOrders += stat.Count
		}
		if stat.Type == models.OrderTypeManual {
			manualOrders += stat.Count
		}

		if stat.DeliverySlot != nil {
			slotKey := string(*stat.DeliverySlot)
			slotCounts[slotKey] += stat.Count
		}
	}

	// Find peak slot
	var peakSlot *string
	maxSlotCount := int64(0)
	for slot, count := range slotCounts {
		if count > maxSlotCount {
			maxSlotCount = count
			slotCopy := slot
			peakSlot = &slotCopy
		}
	}

	// Get best seller
	type BestSellerResult struct {
		ProductID    int
		TotalQuantity int
	}

	var bestSellerResult BestSellerResult
	err := database.DB.Model(&models.OrderItem{}).
		Select("product_id, SUM(quantity) as total_quantity").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.outlet_id = ? AND orders.status IN ?", outletID, []models.OrderStatus{
			models.OrderStatusDelivered,
			models.OrderStatusPartiallyDelivered,
		}).
		Group("product_id").
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
		Select("COALESCE(SUM(total_recharged), 0)").
		Joins("JOIN customer_details ON customer_details.id = wallets.customer_id").
		Joins("JOIN users ON users.id = customer_details.user_id").
		Where("users.outlet_id = ?", outletID).
		Scan(&totalRechargedAmount)

	// Low stock products
	var lowStock []models.Inventory
	database.DB.Where("outlet_id = ?", outletID).
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
		"totalRevenue":          totalRevenue,
		"appOrders":             appOrders,
		"manualOrders":          manualOrders,
		"peakSlot":              peakSlot,
		"bestSellerProduct":     bestSellerProduct,
		"totalRechargedAmount":  totalRechargedAmount,
		"lowStockProducts":      lowStockProducts,
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
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	skip := (page - 1) * limit

	// Count total orders
	var totalOrders int64
	database.DB.Model(&models.Order{}).Where("outlet_id = ?", outletID).Count(&totalOrders)

	// Fetch orders
	var orders []models.Order
	database.DB.Where("outlet_id = ?", outletID).
		Preload("Customer.User").
		Preload("Items.Product").
		Order("created_at DESC").
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
			items[j] = gin.H{
				"name":      item.Product.Name,
				"quantity":  item.Quantity,
				"unitPrice": item.UnitPrice,
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
		Joins("JOIN customer_details ON customer_details.id = tickets.customer_id").
		Joins("JOIN users ON users.id = customer_details.user_id").
		Where("users.outlet_id = ?", user.OutletID).
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
		Where("id = ? AND outlet_id = ?", orderID, outletID).
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
		items[i] = gin.H{
			"id":                 item.ID,
			"productName":        item.Product.Name,
			"productDescription": item.Product.Description,
			"quantity":           item.Quantity,
			"unitPrice":          item.UnitPrice,
			"totalPrice":         float64(item.Quantity) * item.UnitPrice,
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
		Where("id = ? AND outlet_id = ?", req.OrderID, req.OutletID).
		Preload("Items").
		Preload("Customer").
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Order not found for this outlet"})
		return
	}

	// === CANCELLED ===
	if req.Status == "CANCELLED" {
		if order.Status != models.OrderStatusPending {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": fmt.Sprintf("Cannot cancel order. Order status is %s", order.Status),
			})
			return
		}

		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Update order status
			tx.Model(&order).Updates(map[string]interface{}{
				"status":       models.OrderStatusCancelled,
				"delivered_at": nil,
			})

			// Restore stock for all items
			for _, item := range order.Items {
				tx.Model(&models.Inventory{}).
					Where("product_id = ?", item.ProductID).
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
				if err := tx.Where("customer_id = ?", *order.CustomerID).First(&wallet).Error; err == nil {
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
			if tx.Where("order_id = ?", req.OrderID).First(&couponUsage).Error == nil {
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
				if tx.Where("user_id = ? AND consumption_date = ?", order.Customer.UserID, today).
					First(&quota).Error == nil {
					if quota.QuantityUsed >= totalFreeQty {
						tx.Model(&quota).Update("quantity_used", gorm.Expr("quantity_used - ?", totalFreeQty))
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
		if order.Status == models.OrderStatusCancelled {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Cannot mark a cancelled order as delivered."})
			return
		}

		err := database.DB.Transaction(func(tx *gorm.DB) error {
			tx.Model(&models.OrderItem{}).
				Where("order_id = ? AND status != ?", order.ID, models.OrderItemStatusDelivered).
				Update("status", models.OrderItemStatusDelivered)

			now := time.Now()
			tx.Model(&order).Updates(map[string]interface{}{
				"status":       models.OrderStatusDelivered,
				"delivered_at": &now,
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
				"delivered_at": deliveredAt,
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
		if order.Status != models.OrderStatusPartiallyDelivered {
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
				"delivered_at": &now,
			})

			// Restore stock
			for _, item := range undeliveredItems {
				tx.Model(&models.Inventory{}).
					Where("product_id = ?", item.ProductID).
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
				if err := tx.Where("customer_id = ?", *order.CustomerID).First(&wallet).Error; err == nil {
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
				if tx.Where("user_id = ? AND consumption_date = ?", order.Customer.UserID, today).
					First(&quota).Error == nil {
					if quota.QuantityUsed >= totalFreeQty {
						tx.Model(&quota).Update("quantity_used", gorm.Expr("quantity_used - ?", totalFreeQty))
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
