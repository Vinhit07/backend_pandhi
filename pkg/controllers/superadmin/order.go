package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// OutletTotalOrders returns all orders for an outlet with customer and item details
func OutletTotalOrders(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	var err error

	query := database.DB.Preload("Customer.User").
		Preload("Items.Product").
		Order(`"createdAt" DESC`)

	if outletIDStr != "ALL" {
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
		query = query.Where(`"outletId" = ?`, outletID)
		log.Printf("🔍 [OutletTotalOrders] Fetching orders for outlet ID: %d", outletID)
	} else {
		log.Printf("🔍 [OutletTotalOrders] Fetching orders for ALL outlets")
	}

	var orders []models.Order
	result := query.Find(&orders)

	if result.Error != nil {
		log.Printf("❌ [OutletTotalOrders] Database error: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch orders", "error": result.Error.Error()})
		return
	}

	log.Printf("📦 [OutletTotalOrders] Found %d orders in database", len(orders))

	if len(orders) == 0 {
		log.Printf("⚠️  [OutletTotalOrders] No orders found for outlet ID: %d", outletID)
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	formatted := make([]gin.H, len(orders))
	for i, order := range orders {
		log.Printf("🔄 [OutletTotalOrders] Processing order #%d (ID: %d)", i+1, order.ID)

		customerName := "WalkIn"
		var customerPhone *string

		if order.Customer != nil {
			database.DB.Preload("User").First(&order.Customer, order.Customer.ID)
			if order.Customer.User.ID > 0 {
				customerName = order.Customer.User.Name
				customerPhone = order.Customer.User.Phone
			}
		}

		log.Printf("   Customer: %s, Items count: %d", customerName, len(order.Items))

		items := make([]gin.H, len(order.Items))
		for j, item := range order.Items {
			// Use Product.Price if UnitPrice is 0 (same logic as staff order history)
			price := item.UnitPrice
			log.Printf("   [Item %d] ProductName: %s, UnitPrice: %.2f, Product.ID: %d, Product.Price: %.2f",
				j, item.Product.Name, item.UnitPrice, item.Product.ID, item.Product.Price)

			if price == 0 && item.Product.ID > 0 {
				price = item.Product.Price
				log.Printf("   [Item %d] Using Product.Price fallback: %.2f", j, price)
			}

			items[j] = gin.H{
				"productName": item.Product.Name,
				"quantity":    item.Quantity,
				"unitPrice":   price,
				"totalPrice":  price * float64(item.Quantity),
			}
			log.Printf("   [Item %d] Final item: unitPrice=%.2f, totalPrice=%.2f", j, price, price*float64(item.Quantity))
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

	log.Printf("✅ [OutletTotalOrders] Returning %d formatted orders", len(formatted))
	if len(formatted) > 0 {
		log.Printf("📋 [OutletTotalOrders] Sample response: %+v", formatted[0])
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    formatted,
		"message": "Orders fetched successfully",
	})
}
