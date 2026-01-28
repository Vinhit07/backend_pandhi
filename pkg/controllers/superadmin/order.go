package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// OutletTotalOrders returns all orders for an outlet with customer and item details
func OutletTotalOrders(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
	}

	var orders []models.Order
	database.DB.Where("outlet_id = ?", outletID).
		Preload("Customer.User").
		Preload("Items.Product").
		Order("created_at DESC").
		Find(&orders)

	formatted := make([]gin.H, len(orders))
	for i, order := range orders {
		customerName := "WalkIn"
		var customerPhone *string

		if order.Customer != nil {
			database.DB.Preload("User").First(&order.Customer, order.Customer.ID)
			if order.Customer.User.ID > 0 {
				customerName = order.Customer.User.Name
				customerPhone = order.Customer.User.Phone
			}
		}

		items := make([]gin.H, len(order.Items))
		for j, item := range order.Items {
			items[j] = gin.H{
				"productName": item.Product.Name,
				"quantity":    item.Quantity,
				"unitPrice":   item.UnitPrice,
				"totalPrice":  item.UnitPrice * float64(item.Quantity),
			}
		}

		formatted[i] = gin.H{
			"orderId":       order.ID,
			"orderTime":     order.CreatedAt,
			"totalAmount":   order.TotalAmount,
			"paymentMethod": order.PaymentMethod,
			"status":        order.Status,
			"customerName":  customerName,
			"customerPhone": customerPhone,
			"deliveryDate":  order.DeliveryDate,
			"deliverySlot":  order.DeliverySlot,
			"type":          order.Type,
			"items":         items,
		}
	}

	c.JSON(http.StatusOK, formatted)
}
