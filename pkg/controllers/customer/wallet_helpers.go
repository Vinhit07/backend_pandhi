package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RechargeWallet handles legacy cash wallet recharge (manual)
func RechargeWallet(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid amount"})
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

	var customer models.CustomerDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Legacy cash recharge - staff/admin would credit wallet manually
	c.JSON(http.StatusOK, gin.H{
		"message": "Legacy cash recharge endpoint - requires staff approval",
		"amount":  req.Amount,
	})
}

// RecentTrans retrieves recent wallet transactions
func RecentTrans(c *gin.Context) {
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

	var customer models.CustomerDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	var wallet models.Wallet
	if err := database.DB.Where("\"customerId\" = ?", customer.ID).First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Wallet not found"})
		return
	}

	var transactions []models.WalletTransaction
	database.DB.Where("\"walletId\" = ?", wallet.ID).
		Order("\"createdAt\" DESC").
		Limit(50).
		Find(&transactions)

	result := make([]gin.H, len(transactions))
	for i, tx := range transactions {
		// Determine description based on transaction type
		description := "Transaction"
		switch tx.Status {
		case models.WalletTransTypeRecharge:
			description = "Wallet Recharge"
		case models.WalletTransTypeDeduct:
			description = "Order Payment"
		case models.TransactionTypeCredit:
			description = "Refund"
		}
		if tx.Description != "" {
			description = tx.Description
		}

		result[i] = gin.H{
			"id":          tx.ID,
			"amount":      tx.Amount,
			"method":      tx.Method,
			"status":      tx.Status,
			"description": description,
			"createdAt":   tx.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Recent transactions retrieved",
		"transactions": result,
	})
}

// GetRechargeHistory retrieves wallet recharge history
func GetRechargeHistory(c *gin.Context) {
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

	var customer models.CustomerDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	var wallet models.Wallet
	if err := database.DB.Where("\"customerId\" = ?", customer.ID).First(&wallet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Wallet not found"})
		return
	}

	var transactions []models.WalletTransaction
	database.DB.Where("\"walletId\" = ? AND status = ?", wallet.ID, models.WalletTransTypeRecharge).
		Order("\"createdAt\" DESC").
		Find(&transactions)

	result := make([]gin.H, len(transactions))
	for i, tx := range transactions {
		result[i] = gin.H{
			"id":        tx.ID,
			"amount":    tx.Amount,
			"method":    tx.Method,
			"status":    tx.Status,
			"createdAt": tx.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Recharge history retrieved",
		"rechargeHistory": result,
	})
}

// GetServiceChargeBreakdown returns service charge breakdown (currently 0%)
func GetServiceChargeBreakdown(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid amount"})
		return
	}

	breakdown := gin.H{
		"walletAmount":  req.Amount,
		"serviceCharge": 0.0,
		"totalPayable":  req.Amount,
		"chargePercent": 0.0,
	}

	c.JSON(http.StatusOK, breakdown)
}
