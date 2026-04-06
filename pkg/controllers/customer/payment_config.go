package customer

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// GetRazorpayKey returns the Razorpay key ID for frontend checkout
func GetRazorpayKey(c *gin.Context) {
	keyID := os.Getenv("RAZORPAY_KEY_ID")
	if keyID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Razorpay not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"keyId": keyID,
	})
}
