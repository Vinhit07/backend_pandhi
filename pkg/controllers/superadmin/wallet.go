package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetCustomersWithWallet returns customers with wallet details
func GetCustomersWithWallet(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	var err error

	query := database.DB.Where("role = ?", models.RoleCustomer).
		Preload("CustomerInfo.Wallet")

	if outletIDStr != "ALL" {
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Provide outletId"})
			return
		}
		query = query.Where(`"outletId" = ?`, outletID)
		fmt.Printf("[DEBUG] GetCustomersWithWallet - OutletID: %d\n", outletID)
	}

	var users []models.User
	query.Find(&users)
	fmt.Printf("[DEBUG] GetCustomersWithWallet - Found %d users\n", len(users))

	formatted := make([]gin.H, len(users))
	for i, user := range users {
		var customerId *int
		var walletId *int
		balance := 0.0
		totalRecharged := 0.0
		totalUsed := 0.0
		var lastRecharged *time.Time
		var lastOrder *time.Time

		if user.CustomerInfo != nil {
			customerId = &user.CustomerInfo.ID
			if user.CustomerInfo.Wallet != nil {
				walletId = &user.CustomerInfo.Wallet.ID
				balance = user.CustomerInfo.Wallet.Balance
				totalRecharged = user.CustomerInfo.Wallet.TotalRecharged
				totalUsed = user.CustomerInfo.Wallet.TotalUsed
				lastRecharged = user.CustomerInfo.Wallet.LastRecharged
				lastOrder = user.CustomerInfo.Wallet.LastOrder
			}
		}

		formatted[i] = gin.H{
			"userId":         user.ID,
			"name":           user.Name,
			"email":          user.Email,
			"phone":          user.Phone,
			"customerId":     customerId,
			"walletId":       walletId,
			"balance":        balance,
			"totalRecharged": totalRecharged,
			"totalUsed":      totalUsed,
			"lastRecharged":  lastRecharged,
			"lastOrder":      lastOrder,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Customers with wallet fetched successfully",
		"count":   len(formatted),
		"data":    formatted,
	})
}

// GetRechargeHistoryByOutlet returns recharge history for outlet
func GetRechargeHistoryByOutlet(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	var err error

	query := database.DB.Where("role = ?", models.RoleCustomer).
		Preload("CustomerInfo.Wallet.Transactions", func(db *gorm.DB) *gorm.DB {
			return db.Order(`"createdAt" DESC`)
		})

	if outletIDStr != "ALL" {
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Provide outletId"})
			return
		}
		query = query.Where(`"outletId" = ?`, outletID)
		fmt.Printf("[DEBUG] GetRechargeHistoryByOutlet - OutletID: %d\n", outletID)
	}

	var users []models.User
	query.Find(&users)
	fmt.Printf("[DEBUG] GetRechargeHistoryByOutlet - Found %d users\n", len(users))

	history := []gin.H{}
	for _, user := range users {
		if user.CustomerInfo != nil && user.CustomerInfo.Wallet != nil {
			for _, txn := range user.CustomerInfo.Wallet.Transactions {
				history = append(history, gin.H{
					"customerName": user.Name,
					"rechargeId":   txn.ID,
					"amount":       txn.Amount,
					"date":         txn.CreatedAt,
					"method":       txn.Method,
					"status":       txn.Status,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recharge history fetched successfully",
		"count":   len(history),
		"data":    history,
	})
}

// GetOrdersPaidViaWallet returns all wallet-paid orders
func GetOrdersPaidViaWallet(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	var err error

	fmt.Println("[DEBUG] GetOrdersPaidViaWallet - Start")
	var orders []models.Order
	query := database.DB.Where(`"paymentMethod" = ?`, models.PaymentMethodWallet).
		Preload("Customer.User").
		Order(`"createdAt" DESC`)

	if outletIDStr != "ALL" && outletIDStr != "" {
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
		query = query.Where(`"outletId" = ?`, outletID)
		fmt.Printf("[DEBUG] GetOrdersPaidViaWallet - OutletID: %d\n", outletID)
	}

	query.Find(&orders)
	fmt.Printf("[DEBUG] GetOrdersPaidViaWallet - Found %d orders\n", len(orders))

	result := make([]gin.H, len(orders))
	for i, order := range orders {
		customerName := "Unknown"
		if order.Customer != nil {
			database.DB.Preload("User").First(&order.Customer, order.Customer.ID)
			if order.Customer.User.ID > 0 {
				customerName = order.Customer.User.Name
			}
		}

		result[i] = gin.H{
			"orderId":       order.ID,
			"customerName":  customerName,
			"orderTotal":    order.TotalAmount,
			"orderDate":     order.CreatedAt,
			"paymentMethod": order.PaymentMethod,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Orders paid via wallet fetched successfully",
		"count":   len(result),
		"data":    result,
	})
}
