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

// GetRechargeHistory returns wallet recharge history for an outlet
func GetRechargeHistory(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
	}

	// Fetch wallet transactions for customers in this outlet
	var transactions []models.WalletTransaction
	database.DB.
		Joins("JOIN \"Wallet\" ON \"Wallet\".id = \"WalletTransaction\".\"walletId\"").
		Joins("JOIN \"CustomerDetails\" ON \"CustomerDetails\".id = \"Wallet\".\"customerId\"").
		Joins("JOIN \"User\" ON \"User\".id = \"CustomerDetails\".\"userId\"").
		Where("\"User\".\"outletId\" = ? AND \"WalletTransaction\".status = ?", outletID, models.WalletTransTypeRecharge).
		Preload("Wallet.Customer.User").
		Order("\"WalletTransaction\".\"createdAt\" DESC").
		Find(&transactions)

	formattedTransactions := make([]gin.H, len(transactions))
	for i, tx := range transactions {
		customerName := "Unknown"
		if tx.Wallet.Customer.ID > 0 {
			database.DB.Preload("User").First(&tx.Wallet.Customer, tx.Wallet.Customer.ID)
			if tx.Wallet.Customer.User.ID > 0 {
				customerName = tx.Wallet.Customer.User.Name
			}
		}

		formattedTransactions[i] = gin.H{
			"id":           tx.ID,
			"customerName": customerName,
			"amount":       tx.Amount,
			"method":       tx.Method,
			"createdAt":    tx.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Recharge history fetched",
		"transactions": formattedTransactions,
	})
}

// AddRecharge manually adds wallet balance (cash recharge by staff)
func AddRecharge(c *gin.Context) {
	var req struct {
		CustomerID int     `json:"customerId" binding:"required"`
		Amount     float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide valid customerId and amount"})
		return
	}

	// Find or create wallet
	var wallet models.Wallet
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where(map[string]interface{}{"customerId": req.CustomerID}).First(&wallet)
		if result.Error == gorm.ErrRecordNotFound {
			// Create wallet if doesn't exist
			wallet = models.Wallet{
				CustomerID:     req.CustomerID,
				Balance:        req.Amount,
				TotalRecharged: req.Amount,
				TotalUsed:      0,
			}
			now := time.Now()
			wallet.LastRecharged = &now
			tx.Create(&wallet)
		} else if result.Error == nil {
			// Update existing wallet
			now := time.Now()
			tx.Model(&wallet).Updates(map[string]interface{}{
				"balance":        wallet.Balance + req.Amount,
				"totalRecharged": wallet.TotalRecharged + req.Amount,
				"lastRecharged":  &now,
			})
			wallet.Balance += req.Amount
			wallet.TotalRecharged += req.Amount
		} else {
			return result.Error
		}

		// Create transaction record
		tx.Create(&models.WalletTransaction{
			WalletID: wallet.ID,
			Amount:   req.Amount,
			Method:   models.PaymentMethodCash,
			Status:   models.WalletTransTypeRecharge,
		})

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to recharge wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wallet recharged successfully",
		"wallet":  wallet,
	})
}
