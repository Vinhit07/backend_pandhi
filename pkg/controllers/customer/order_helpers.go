package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CustomerAppOngoingOrderList retrieves ongoing (PENDING) orders
func CustomerAppOngoingOrderList(c *gin.Context) {
	// Get user
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

	// Get customer
	var customer models.CustomerDetails
	if err := database.DB.Where("user_id = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Fetch ongoing orders
	var orders []models.Order
	if err := database.DB.
		Where("customer_id = ? AND status = ?", customer.ID, models.OrderStatusPending).
		Preload("Items.Product").
		Preload("Outlet").
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if len(orders) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No ongoing orders found",
			"orders":  []interface{}{},
		})
		return
	}

	// Format orders
	formattedOrders := make([]gin.H, len(orders))
	for i, order := range orders {
		items := make([]gin.H, len(order.Items))
		for j, item := range order.Items {
// Helper to get signed URL
			var imageURL *string
			if item.Product.ImageURL != nil {
				signedURL, _ := services.GetSignedURL(*item.Product.ImageURL)
				imageURL = &signedURL
			}

			items[j] = gin.H{
				"id":         item.ID,
				"productId":  item.ProductID,
				"quantity":   item.Quantity,
				"unitPrice":  item.UnitPrice,
				"status":     item.Status,
				"product": gin.H{
					"id":          item.Product.ID,
					"name":        item.Product.Name,
					"description": item.Product.Description,
					"price":       item.Product.Price,
					"imageUrl":    imageURL,
				},
			}
		}

		formattedOrders[i] = gin.H{
			"id":            order.ID,
			"orderNumber":   fmt.Sprintf("#ORD-%06d", order.ID),
			"totalAmount":   order.TotalAmount,
			"paymentMethod": order.PaymentMethod,
			"status":        order.Status,
			"deliveryDate":  order.DeliveryDate,
			"deliverySlot":  order.DeliverySlot,
			"createdAt":     order.CreatedAt,
			"items":         items,
			"outlet": gin.H{
				"id":       order.Outlet.ID,
				"name":     order.Outlet.Name,
				"address":  order.Outlet.Address,
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ongoing orders retrieved",
		"orders":  formattedOrders,
	})
}

// CustomerAppOrderHistory retrieves completed orders
func CustomerAppOrderHistory(c *gin.Context) {
	// Get user
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

	// Get customer
	var customer models.CustomerDetails
	if err := database.DB.Where("user_id = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Fetch completed orders
	var orders []models.Order
	if err := database.DB.
		Where("customer_id = ? AND status IN ?", customer.ID, []models.OrderStatus{
			models.OrderStatusDelivered,
			models.OrderStatusCancelled,
			models.OrderStatusPartiallyDelivered,
			models.OrderStatusPartialCancel,
		}).
		Preload("Items.Product").
		Preload("Outlet").
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if len(orders) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No order history found",
			"orders":  []interface{}{},
		})
		return
	}

	// Format orders (same as ongoing)
	formattedOrders := make([]gin.H, len(orders))
	for i, order := range orders {
		items := make([]gin.H, len(order.Items))
		for j, item := range order.Items {
// Helper to get signed URL
			var imageURL *string
			if item.Product.ImageURL != nil {
				signedURL, _ := services.GetSignedURL(*item.Product.ImageURL)
				imageURL = &signedURL
			}

			items[j] = gin.H{
				"id":         item.ID,
				"productId":  item.ProductID,
				"quantity":   item.Quantity,
				"unitPrice":  item.UnitPrice,
				"status":     item.Status,
				"product": gin.H{
					"id":          item.Product.ID,
					"name":        item.Product.Name,
					"description": item.Product.Description,
					"price":       item.Product.Price,
					"imageUrl":    imageURL,
				},
			}
		}

		formattedOrders[i] = gin.H{
			"id":            order.ID,
			"orderNumber":   fmt.Sprintf("#ORD-%06d", order.ID),
			"totalAmount":   order.TotalAmount,
			"paymentMethod": order.PaymentMethod,
			"status":        order.Status,
			"deliveryDate":  order.DeliveryDate,
			"deliverySlot":  order.DeliverySlot,
			"deliveredAt":   order.DeliveredAt,
			"createdAt":     order.CreatedAt,
			"items":         items,
			"outlet": gin.H{
				"id":      order.Outlet.ID,
				"name":    order.Outlet.Name,
				"address": order.Outlet.Address,
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order history retrieved",
		"orders":  formattedOrders,
	})
}

// CustomerAppCancelOrder cancels a pending order
func CustomerAppCancelOrder(c *gin.Context) {
	orderIDStr := c.Param("orderId")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid order ID"})
		return
	}

	// Get user
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

	// Get customer
	var customer models.CustomerDetails
	if err := database.DB.Where("user_id = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Fetch order
	var order models.Order
	if err := database.DB.
		Where("id = ? AND customer_id = ?", orderID, customer.ID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Order not found"})
		return
	}

	// Check if order can be cancelled
	if order.Status != models.OrderStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Only pending orders can be cancelled",
			"status":  order.Status,
		})
		return
	}

	// Update order status
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&order).Update("status", models.OrderStatusCancelled).Error; err != nil {
			return err
		}

		// Refund if paid via wallet
		if order.PaymentMethod == "WALLET" {
			var wallet models.Wallet
			if err := tx.Where("customer_id = ?", customer.ID).First(&wallet).Error; err != nil {
				return err
			}

			// Credit wallet
			wallet.Balance += order.TotalAmount
			if err := tx.Save(&wallet).Error; err != nil {
				return err
			}

			// Create transaction record
			transaction := models.WalletTransaction{
				WalletID:    wallet.ID,
				Amount:      order.TotalAmount,
				Status:      models.TransactionTypeCredit,
				Description: fmt.Sprintf("Refund for order #%d", order.ID),
				CreatedAt:   time.Now(),
			}
			if err := tx.Create(&transaction).Error; err != nil {
				return err
			}
		}

		// Restore inventory
		var orderItems []models.OrderItem
		if err := tx.Where("order_id = ?", order.ID).Find(&orderItems).Error; err != nil {
			return err
		}

		for _, item := range orderItems {
			// Find inventory
			var inventory models.Inventory
			if err := tx.Where("product_id = ? AND outlet_id = ?", item.ProductID, order.OutletID).First(&inventory).Error; err == nil {
				// Add stock back
				inventory.Quantity += item.Quantity
				if err := tx.Save(&inventory).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to cancel order and process refund",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order cancelled successfully",
		"orderId": order.ID,
		"status":  models.OrderStatusCancelled,
	})
}

// CreateRazorpayOrder creates a Razorpay order for payment
func CreateRazorpayOrder(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Amount is required"})
		return
	}

	// Get user to associate with order
	userInterface, exists := c.Get("user")
	userID := 0
	if exists {
		if user, ok := userInterface.(models.User); ok {
			userID = user.ID
		}
	}

	// Create order with reference notes
	order, err := services.CreateRazorpayOrder(req.Amount, "INR", fmt.Sprintf("order_%d_%d", userID, time.Now().Unix()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create Razorpay order", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
	})
}

// VerifyRazorpayPayment verifies Razorpay payment
func VerifyRazorpayPayment(c *gin.Context) {
	var req struct {
		RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
		RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
		RazorpaySignature string `json:"razorpay_signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing payment details"})
		return
	}

	isValid := services.VerifyPaymentSignature(req.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature)
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid payment signature"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Payment verified successfully",
	})
}
