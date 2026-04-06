package auth

import (
	"backend_pandhi/pkg/config"
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"backend_pandhi/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SendOTP generates and sends an OTP to the given email
func SendOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Valid email is required"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Generate OTP
	code, err := services.GenerateOTP(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate OTP"})
		return
	}

	// Send OTP via email
	if err := services.SendOTPEmail(email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send OTP"})
		return
	}

	// Check if user exists (to inform frontend whether to show name field)
	var existingUser models.User
	userExists := database.DB.Where("email = ?", email).First(&existingUser).Error == nil

	c.JSON(http.StatusOK, gin.H{
		"message":    "OTP sent successfully",
		"email":      email,
		"userExists": userExists,
	})
}

// VerifyOTP validates the OTP and authenticates/creates the user
func VerifyOTPHandler(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		OTP   string `json:"otp" binding:"required"`
		Name  string `json:"name"` // Required only for new users
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and OTP are required"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Verify OTP
	valid, errMsg := services.VerifyOTP(email, req.OTP)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"message": errMsg})
		return
	}

	// OTP is valid — find or create user
	var user models.User
	err := database.DB.Where("email = ?", email).First(&user).Error

	if err != nil {
		// User doesn't exist — create new user
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = strings.Split(email, "@")[0] // Use email prefix as name
		}

		emptyStr := ""
		outletID := 1
		user = models.User{
			Name:     name,
			Email:    email,
			Password: &emptyStr, // No password for OTP users
			Role:     "CUSTOMER",
			OutletID: &outletID, // Default outlet
			Phone:    &emptyStr,
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
			return
		}

		// Create customer details
		customerDetails := models.CustomerDetails{
			UserID: user.ID,
		}
		database.DB.Create(&customerDetails)

		// Create wallet for new customer
		wallet := models.Wallet{
			CustomerID:     customerDetails.ID,
			Balance:        0,
			TotalRecharged: 0,
			TotalUsed:      0,
		}
		database.DB.Create(&wallet)

		// Create cart for new customer
		cart := models.Cart{
			CustomerID: customerDetails.ID,
		}
		database.DB.Create(&cart)
	}

	// Load relationships
	database.DB.
		Preload("CustomerInfo.Wallet").
		Preload("CustomerInfo.Cart").
		Preload("Outlet").
		First(&user, user.ID)

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate token"})
		return
	}

	// Set cookie
	c.SetCookie(
		"token",
		token,
		7*24*60*60,
		"/",
		"",
		config.AppConfig.CookieSecure == "true",
		true,
	)

	// Build response
	response := gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"phone":    user.Phone,
		"role":     user.Role,
		"outletId": user.OutletID,
		"outlet":   user.Outlet,
		"imageUrl": nil,
	}

	if user.CustomerInfo != nil {
		response["customerDetails"] = gin.H{
			"id":          user.CustomerInfo.ID,
			"yearOfStudy": user.CustomerInfo.YearOfStudy,
			"wallet":      user.CustomerInfo.Wallet,
			"cart":        user.CustomerInfo.Cart,
		}
	}

	jsonResponse := gin.H{
		"message": "Authentication successful",
		"user":    response,
	}

	if strings.TrimSpace(config.AppConfig.EnableMobileTokenReturn) == "true" {
		jsonResponse["token"] = token
	}

	c.JSON(http.StatusOK, jsonResponse)
}
