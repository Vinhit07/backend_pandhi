package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetOutletCustomers returns customers for an outlet with wallet and order stats
func GetOutletCustomers(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
	}

	// Get users with customer info
	var users []models.User
	database.DB.Where("outlet_id = ? AND role = ?", outletID, models.RoleCustomer).
		Preload("CustomerInfo.Wallet").
		Preload("CustomerInfo.Orders").
		Find(&users)

	formattedCustomers := make([]gin.H, len(users))
	for i, user := range users {
		var customerID *int
		var walletID *int
		var yearOfStudy *int
		var walletBalance float64
		var totalOrders int64
		var totalPurchase float64
		var lastOrderDate *string

		if user.CustomerInfo != nil {
			customerID = &user.CustomerInfo.ID
			if user.CustomerInfo.YearOfStudy != nil {
				yearOfStudy = user.CustomerInfo.YearOfStudy
			}

			if user.CustomerInfo.Wallet != nil {
				walletID = &user.CustomerInfo.Wallet.ID
				walletBalance = user.CustomerInfo.Wallet.Balance
			}

			// Calculate order stats
			totalOrders = int64(len(user.CustomerInfo.Orders))
			for _, order := range user.CustomerInfo.Orders {
				totalPurchase += order.TotalAmount
			}

			// Get last order date
			if len(user.CustomerInfo.Orders) > 0 {
				latest := user.CustomerInfo.Orders[0].CreatedAt
				for _, order := range user.CustomerInfo.Orders {
					if order.CreatedAt.After(latest) {
						latest = order.CreatedAt
					}
				}
				dateStr := latest.Format("2006-01-02T15:04:05Z")
				lastOrderDate = &dateStr
			}
		}

		formattedCustomers[i] = gin.H{
			"customerId":        customerID,
			"walletId":          walletID,
			"name":              user.Name,
			"email":             user.Email,
			"yearOfStudy":       yearOfStudy,
			"phoneNo":           user.Phone,
			"walletBalance":     walletBalance,
			"totalOrders":       totalOrders,
			"totalPurchaseCost": totalPurchase,
			"lastOrderDate":     lastOrderDate,
		}
	}

	message := "Customers retrieved successfully"
	if len(formattedCustomers) == 0 {
		message = "No customers found for this outlet"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   message,
		"customers": formattedCustomers,
	})
}
