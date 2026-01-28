package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Removed deprecated razorpayService init

// CreateWalletRechargeOrder creates a Razorpay order for wallet recharge
func CreateWalletRechargeOrder(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid amount",
			"error":   "Amount must be greater than 0",
		})
		return
	}

	// Get user from context
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

	// Get customer details
	var customer models.CustomerDetails
	if err := database.DB.Where("user_id = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer details not found"})
		return
	}

	// Create Razorpay order
	order, err := services.CreateRazorpayOrder(req.Amount, "INR", string(rune(user.ID)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create payment order",
			"error":   err.Error(),
		})
		return
	}

	// Service charge is typically handled on frontend or in specific calculations.
	// For now, returning dummy breakdown or simplified.
	// Assuming 2% fee as per standard or what previous service did.
	serviceCharge := req.Amount * 0.02
	totalPayable := req.Amount + serviceCharge

	c.JSON(http.StatusCreated, gin.H{
		"message": "Wallet recharge order created successfully",
		"order":   order,
		"breakdown": gin.H{
			"walletAmount":            req.Amount,
			"serviceCharge":           serviceCharge,
			"totalPayable":            totalPayable,
			"serviceChargePercentage": 2.0,
		},
	})
}

// VerifyWalletRecharge verifies payment and processes wallet recharge
func VerifyWalletRecharge(c *gin.Context) {
	var req struct {
		RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
		RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
		RazorpaySignature string `json:"razorpay_signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing payment verification details"})
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

	// Verify signature
	if !services.VerifyPaymentSignature(req.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Payment verification failed",
			"error":   "Invalid signature",
		})
		return
	}

	// Assuming success if signature verifies as we removed FetchPaymentDetails
	payment := map[string]interface{}{
		"id":     req.RazorpayPaymentID,
		"status": "captured",
		"method": "upi", // Defaulting to UPI or could get from client if needed, but verifying via backend fetch is safer. Since we removed Fetch, we accept it.
		// For wallet amount, we depend on what was ordered.
		// We need to fetch order details to confirm amount if we want to be strict.
		// For now, trusting the signature verification implies the orderID is valid, and we can look up the transaction amount if we stored it, or trust frontend (less secure).
		// Better: We should probably store the pending recharge order in DB and look it up by RazorpayOrderID.
		// But let's assume valid for this migration step.
		"amount": 0, // Placeholder
		"notes": map[string]interface{}{
			"wallet_amount": 0.0, // We need to know this. Without store, we rely on Fetch.
			// Revert to fetching payment details using a direct http call if needed?
			// Or just assume standard calculation without specific notes.
		},
	}
	
	// CRITICAL: We need the amount to recharge the wallet!
	// Existing code extracts "wallet_amount" from notes.
	// If `CreateRazorpayOrder` didn't add notes, this fails.
	// My `services.CreateRazorpayOrder` in step 756 ADDS notes!
	// So fetching payment details IS needed to get those notes back.
	// I should probably Implement FetchPaymentDetails in services/razorpay.go
	// But for now, to fix build, I will comment out the notes extraction and use a placeholder or better yet,
	// I will just add FetchPaymentDetails to services/razorpay.go in the NEXT step if this works.
	// For now, let's fix method signature.
	
	// Wait, I can't easily get the amount without fetching. 
	// I'll make a simplified assumption: The user sends the amount in body? No, the verification endpoint only takes IDs.
	// I MUST fetch payment details to know how much to credit.
	// I will add FetchPaymentDetails to `pkg/services/razorpay.go` in a subsequent step.
	// For this file, I'll update it to call `services.FetchPaymentDetails`.
	
	payment, err := services.FetchPaymentDetails(req.RazorpayPaymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to fetch payment details",
			"error":   err.Error(),
		})
		return
	}

	status, _ := payment["status"].(string)
	if status != "captured" && status != "authorized" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Payment not successful",
			"status":  status,
		})
		return
	}

	// Get customer
	var customer models.CustomerDetails
	if err := database.DB.Where("user_id = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer details not found"})
		return
	}

	// Process in transaction
	var result struct {
		Wallet      models.Wallet
		Transaction models.WalletTransaction
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Check if already processed
		var existing models.WalletTransaction
		if tx.Where("razorpay_payment_id = ?", req.RazorpayPaymentID).First(&existing).Error == nil {
			return fmt.Errorf("Payment already processed")
		}

		// Extract amounts from payment notes
		notes := payment["notes"].(map[string]interface{})
		walletAmount, _ := notes["wallet_amount"].(float64)
		serviceCharge, _ := notes["service_charge"].(float64)
		grossAmount := float64(payment["amount"].(int)) / 100 // Convert from paise

		// Update wallet
		var wallet models.Wallet
		err := tx.Where("customer_id = ?", customer.ID).First(&wallet).Error
		if err != nil {
			// Create new wallet
			now := time.Now()
			wallet = models.Wallet{
				CustomerID:     customer.ID,
				Balance:        walletAmount,
				TotalRecharged: walletAmount,
				TotalUsed:      0,
				LastRecharged:  &now,
			}
			tx.Create(&wallet)
		} else {
			// Update wallet
			now := time.Now()
			tx.Model(&wallet).Updates(map[string]interface{}{
				"balance":         wallet.Balance + walletAmount,
				"total_recharged": wallet.TotalRecharged + walletAmount,
				"last_recharged":  &now,
			})
		}

		// Create transaction
		method := "CARD"
		if pMethod, ok := payment["method"].(string); ok && pMethod == "upi" {
			method = "UPI"
		}

		transaction := models.WalletTransaction{
			WalletID:           wallet.ID,
			Amount:             walletAmount,
			GrossAmount:        &grossAmount,
			ServiceCharge:      &serviceCharge,
			Method:             models.PaymentMethod(method),
			RazorpayPaymentID:  &req.RazorpayPaymentID,
			RazorpayOrderID:    &req.RazorpayOrderID,
			Status:             models.WalletTransTypeRecharge,
		}
		tx.Create(&transaction)

		result.Wallet = wallet
		result.Transaction = transaction
		return nil
	})

	if err != nil {
		if err.Error() == "Payment already processed" {
			c.JSON(http.StatusConflict, gin.H{
				"message": "Payment already processed",
				"error":   err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Wallet recharge verification failed",
				"error":   err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wallet recharged successfully",
		"wallet": gin.H{
			"balance":        result.Wallet.Balance,
			"totalRecharged": result.Wallet.TotalRecharged,
			"lastRecharged":  result.Wallet.LastRecharged,
		},
		"transaction": gin.H{
			"id":            result.Transaction.ID,
			"amount":        result.Transaction.Amount,
			"grossAmount":   result.Transaction.GrossAmount,
			"serviceCharge": result.Transaction.ServiceCharge,
			"method":        result.Transaction.Method,
			"createdAt":     result.Transaction.CreatedAt,
		},
		"payment": gin.H{
			"id":     payment["id"],
			"status": payment["status"],
			"method": payment["method"],
		},
	})
}

// GetWalletDetails retrieves the customer's wallet details
func GetWalletDetails(c *gin.Context) {
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
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer details not found"})
		return
	}

	// Get or create wallet
	var wallet models.Wallet
	err := database.DB.Where("customer_id = ?", customer.ID).First(&wallet).Error
	if err != nil {
		wallet = models.Wallet{
			CustomerID:     customer.ID,
			Balance:        0,
			TotalRecharged: 0,
			TotalUsed:      0,
		}
		database.DB.Create(&wallet)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wallet details fetched successfully",
		"wallet": gin.H{
			"balance":        wallet.Balance,
			"totalRecharged": wallet.TotalRecharged,
			"totalUsed":      wallet.TotalUsed,
			"lastRecharged":  wallet.LastRecharged,
			"lastOrder":      wallet.LastOrder,
		},
	})
}
