package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

//OutletCurrentOrder returns current orders for notification purposes
func OutletCurrentOrder(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
	}

	// Fetch pending/in-progress orders
	var orders []models.Order
	database.DB.
		Where("outlet_id = ? AND status IN ?", outletID, []models.OrderStatus{
			models.OrderStatusPending,
			models.OrderStatusPartiallyDelivered,
		}).
		Preload("Customer.User").
		Preload("Items.Product").
		Order("created_at DESC").
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
				"productName": item.Product.Name,
				"quantity":    item.Quantity,
				"status":      item.Status,
			}
		}

		formattedOrders[i] = gin.H{
			"id":            order.ID,
			"customerName":  customerName,
			"totalAmount":   order.TotalAmount,
			"paymentMethod": order.PaymentMethod,
			"status":        order.Status,
			"type":          order.Type,
			"deliverySlot":  order.DeliverySlot,
			"createdAt":     order.CreatedAt,
			"items":         items,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Current orders retrieved",
		"orders":  formattedOrders,
		"count":   len(formattedOrders),
	})
}
