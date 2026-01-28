package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"

	"github.com/razorpay/razorpay-go"
)

var (
	razorpayClient *razorpay.Client
)

// InitRazorpay initializes the Razorpay client
func InitRazorpay() error {
	keyID := os.Getenv("RAZORPAY_KEY_ID")
	keySecret := os.Getenv("RAZORPAY_KEY_SECRET")

	if keyID == "" || keySecret == "" {
		fmt.Println("Warning: RAZORPAY_KEY_ID or RAZORPAY_KEY_SECRET not set")
		return nil // Don't fail init, just warn
	}

	razorpayClient = razorpay.NewClient(keyID, keySecret)
	return nil
}

// CalculateGrossAmount calculates the gross amount customer needs to pay
func CalculateGrossAmount(amount float64) (float64, float64, float64) {
	// No service charge - customer pays exactly the wallet amount
	grossAmount := math.Round(amount*100) / 100
	serviceCharge := 0.0
	return amount, grossAmount, serviceCharge
}

// CreateRazorpayOrder creates a Razorpay order
func CreateRazorpayOrder(amount float64, currency, receiptID string) (map[string]interface{}, error) {
	if razorpayClient == nil {
		return nil, fmt.Errorf("Razorpay client not initialized")
	}

	// Amount in paise
	amountInPaise := math.Round(amount * 100)

	data := map[string]interface{}{
		"amount":   amountInPaise,
		"currency": currency,
		"notes": map[string]interface{}{
			"receipt_id": receiptID,
			// Add wallet amount note implicitly for wallet recharge logic compatibility
			"wallet_amount": amount,
		},
		"receipt": fmt.Sprintf("receipt_%v", receiptID),
	}

	body, err := razorpayClient.Order.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create razorpay order: %v", err)
	}

	return body, nil
}

// FetchPaymentDetails fetches payment details from Razorpay
func FetchPaymentDetails(paymentID string) (map[string]interface{}, error) {
	if razorpayClient == nil {
		return nil, fmt.Errorf("Razorpay client not initialized")
	}

	body, err := razorpayClient.Payment.Fetch(paymentID, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment details: %v", err)
	}

	return body, nil
}

// VerifyPaymentSignature verifies the Razorpay payment signature
func VerifyPaymentSignature(orderID, paymentID, signature string) bool {
	keySecret := os.Getenv("RAZORPAY_KEY_SECRET")
	if keySecret == "" {
		return false
	}

	data := orderID + "|" + paymentID
	h := hmac.New(sha256.New, []byte(keySecret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return expectedSignature == signature
}


