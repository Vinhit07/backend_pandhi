package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AddManualOrder creates a manual/phone order with inventory deduction
func AddManualOrder(c *gin.Context) {
	var req struct {
		OutletID      int     `json:"outletId" binding:"required"`
		TotalAmount   float64 `json:"totalAmount" binding:"required"`
		PaymentMethod string  `json:"paymentMethod" binding:"required"`
		Status        string  `json:"status"`
		Items         []struct {
			ProductID int     `json:"productId" binding:"required"`
			Quantity  int     `json:"quantity" binding:"required"`
			UnitPrice float64 `json:"unitPrice" binding:"required"`
		} `json:"items" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing required fields"})
		return
	}

	// Validate inventory
	for _, item := range req.Items {
		var inventory models.Inventory
		if err := database.DB.
			Where("outlet_id = ? AND product_id = ?", req.OutletID, item.ProductID).
			First(&inventory).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Inventory not found for product ID " + string(rune(item.ProductID)),
			})
			return
		}

		if inventory.Quantity < item.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Insufficient inventory for product ID " + string(rune(item.ProductID)),
			})
			return
		}
	}

	// Create order in transaction
	var createdOrder models.Order
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		today := time.Now().Truncate(24 * time.Hour)
		now := time.Now()

		order := models.Order{
			OutletID:      req.OutletID,
			TotalAmount:   req.TotalAmount,
			PaymentMethod: models.PaymentMethod(req.PaymentMethod),
			Status:        string(models.OrderStatusDelivered),
			Type:          models.OrderTypeManual,
			CustomerID:    nil,
			DeliveryDate:  &today,
			IsPreOrder:    false,
			DeliveredAt:   &now,
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		// Create order items
		for _, item := range req.Items {
			orderItem := models.OrderItem{
				OrderID:   order.ID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				Status:    models.OrderItemStatusDelivered,
			}
			tx.Create(&orderItem)

			// Deduct inventory
			tx.Model(&models.Inventory{}).
				Where("outlet_id = ? AND product_id = ?", req.OutletID, item.ProductID).
				Update("quantity", gorm.Expr("quantity - ?", item.Quantity))
		}

		tx.Preload("Items").First(&order, order.ID)
		createdOrder = order
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Manual order created",
		"order":   createdOrder,
	})
}

// GetProducts returns available products with stock for manual orders
func GetProducts(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide a valid outletId"})
		return
	}

	// Fetch products with inventory > 0
	var inventories []models.Inventory
	database.DB.Where("outlet_id = ? AND quantity > 0", outletID).
		Preload("Product").
		Find(&inventories)

	availableProducts := make([]gin.H, len(inventories))
	for i, inv := range inventories {
		availableProducts[i] = gin.H{
			"id":                inv.Product.ID,
			"name":              inv.Product.Name,
			"description":       inv.Product.Description,
			"price":             inv.Product.Price,
			"imageUrl":          inv.Product.ImageURL,
			"category":          inv.Product.Category,
			"quantityAvailable": inv.Quantity,
		}
	}

	c.JSON(http.StatusOK, gin.H{"products": availableProducts})
}
