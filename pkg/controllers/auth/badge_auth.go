package auth

import (
	"backend_pandhi/pkg/config"
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// VerifyBadge handles the initial registration step with a badge ID
func VerifyBadge(c *gin.Context) {
	var req struct {
		BadgeID string `json:"badgeId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "badgeId is required"})
		return
	}

	// The user's manually created PostgreSQL function `verify_badge_access`
	// RETURNS TABLE (target_email text, exists_bool boolean)
	var badgeResult []struct {
		TargetEmail string `gorm:"column:target_email"`
		ExistsBool  bool   `gorm:"column:exists_bool"`
	}

	// Use SELECT * FROM because it's a set-returning function
	err := database.DB.Raw("SELECT * FROM verify_badge_access(?)", req.BadgeID).Scan(&badgeResult).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database query failed", "error": err.Error()})
		return
	}

	// If no rows returned, it means the badge was either not found or already claimed
	if len(badgeResult) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid or already claimed Badge ID"})
		return
	}

	if badgeResult[0].TargetEmail == "" {
		fmt.Printf("ERROR: Database row found, but TargetEmail was empty. Raw results: %+v\n", badgeResult)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Database column mapping failed in Go backend"})
		return
	}

	// Because the user's schema doesn't store Name in badges table anymore, we don't have a pre-linked name
	// We'll just pass an empty string for name so the user can fill it in the UI.
	email := badgeResult[0].TargetEmail
	name := ""

	// Trigger Supabase Auth OTP
	supabaseURL := config.AppConfig.SupabaseURL
	supabaseKey := config.AppConfig.SupabaseServiceKey

	// Fallback mechanism to ensure variables are fetched even if main config skipped them
	if supabaseURL == "" || supabaseKey == "" {
		envMap, _ := godotenv.Read(".env")
		if supabaseURL == "" {
			supabaseURL = envMap["SUPABASE_URL"]
		}
		if supabaseKey == "" {
			supabaseKey = envMap["SUPABASE_SERVICE_ROLE_KEY"]
		}
	}

	if supabaseURL == "" || supabaseKey == "" {
		fmt.Printf("Supabase configuration missing. URL length: %d, Key length: %d\n", len(supabaseURL), len(supabaseKey))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Supabase configuration missing"})
		return
	}

	otpReqBody, _ := json.Marshal(map[string]interface{}{
		"email":       email,
		"create_user": true,
	})

	httpClient := &http.Client{}
	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/v1/otp", supabaseURL), bytes.NewBuffer(otpReqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create Supabase request"})
		return
	}

	httpReq.Header.Set("apikey", supabaseKey)
	httpReq.Header.Set("Authorization", "Bearer "+supabaseKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil || (httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated) {
		if httpResp != nil {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to trigger OTP", "details": string(bodyBytes)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to trigger OTP"})
		}
		return
	}
	defer httpResp.Body.Close()

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent successfully",
		"email":   email,
		"name":    name,
	})
}

// VerifyBadgeOTP proxies the OTP verification through the backend to Supabase
func VerifyBadgeOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "email and token are required"})
		return
	}

	supabaseURL := config.AppConfig.SupabaseURL
	supabaseKey := config.AppConfig.SupabaseServiceKey

	// Fallback mechanism
	if supabaseURL == "" || supabaseKey == "" {
		envMap, _ := godotenv.Read(".env")
		if supabaseURL == "" {
			supabaseURL = envMap["SUPABASE_URL"]
		}
		if supabaseKey == "" {
			supabaseKey = envMap["SUPABASE_SERVICE_ROLE_KEY"]
		}
	}

	if supabaseURL == "" || supabaseKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Supabase configuration missing"})
		return
	}

	// Call Supabase verify endpoint
	verifyBody, _ := json.Marshal(map[string]interface{}{
		"email": req.Email,
		"token": req.Token,
		"type":  "email",
	})

	httpClient := &http.Client{}
	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/v1/verify", supabaseURL), bytes.NewBuffer(verifyBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create verify request"})
		return
	}

	httpReq.Header.Set("apikey", supabaseKey)
	httpReq.Header.Set("Authorization", "Bearer "+supabaseKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to verify OTP"})
		return
	}
	defer httpResp.Body.Close()

	bodyBytes, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode != http.StatusOK {
		var errData map[string]interface{}
		json.Unmarshal(bodyBytes, &errData)
		msg := "Invalid OTP"
		if m, ok := errData["msg"].(string); ok {
			msg = m
		} else if m, ok := errData["message"].(string); ok {
			msg = m
		}
		c.JSON(http.StatusBadRequest, gin.H{"message": msg})
		return
	}

	// Return the Supabase session data
	var sessionData map[string]interface{}
	json.Unmarshal(bodyBytes, &sessionData)

	c.JSON(http.StatusOK, gin.H{
		"message": "OTP verified successfully",
		"session": sessionData,
	})
}

// ProfileSetup completes the registration by updating auth.users and local tables
func ProfileSetup(c *gin.Context) {
	var req struct {
		BadgeID        string `json:"badgeId" binding:"required"`
		Email          string `json:"email" binding:"required,email"`
		Name           string `json:"name" binding:"required"`
		Password       string `json:"password" binding:"required"`
		RetypePassword string `json:"retypePassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request fields"})
		return
	}

	if req.Password != req.RetypePassword {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Passwords do not match"})
		return
	}

	// Check if badge is valid again just in case
	var badge models.Badge
	if err := database.DB.Where("id = ?", req.BadgeID).First(&badge).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Badge ID"})
		return
	}

	if badge.IsClaimed {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Badge already claimed"})
		return
	}

	// 1. Get user ID from Supabase via Admin API (DB connection can't access auth schema via PgBouncer)
	supabaseURL := config.AppConfig.SupabaseURL
	supabaseKey := config.AppConfig.SupabaseServiceKey

	// Fallback mechanism
	if supabaseURL == "" || supabaseKey == "" {
		envMap, _ := godotenv.Read(".env")
		if supabaseURL == "" {
			supabaseURL = envMap["SUPABASE_URL"]
		}
		if supabaseKey == "" {
			supabaseKey = envMap["SUPABASE_SERVICE_ROLE_KEY"]
		}
	}

	if supabaseURL == "" || supabaseKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Supabase configuration missing"})
		return
	}

	// Look up user by email via Supabase Admin API
	httpClient := &http.Client{}
	listReq, err := http.NewRequest("GET", fmt.Sprintf("%s/auth/v1/admin/users?filter=%s", supabaseURL, req.Email), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create Supabase lookup request"})
		return
	}
	listReq.Header.Set("apikey", supabaseKey)
	listReq.Header.Set("Authorization", "Bearer "+supabaseKey)

	listResp, err := httpClient.Do(listReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to query Supabase users"})
		return
	}
	defer listResp.Body.Close()

	listBody, _ := io.ReadAll(listResp.Body)
	var usersResponse struct {
		Users []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"users"`
	}
	json.Unmarshal(listBody, &usersResponse)

	var supabaseUserID string
	for _, u := range usersResponse.Users {
		if u.Email == req.Email {
			supabaseUserID = u.ID
			break
		}
	}

	if supabaseUserID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Could not locate user in Supabase auth."})
		return
	}

	// 2. Call Supabase Admin API to update password and user_metadata
	updateReqBody, _ := json.Marshal(map[string]interface{}{
		"password": req.Password,
		"user_metadata": map[string]interface{}{
			"full_name": req.Name,
		},
		"email_confirm": true, // ensure they are confirmed
	})

	httpReq, err := http.NewRequest("PUT", fmt.Sprintf("%s/auth/v1/admin/users/%s", supabaseURL, supabaseUserID), bytes.NewBuffer(updateReqBody))
	if err == nil {
		httpReq.Header.Set("apikey", supabaseKey)
		httpReq.Header.Set("Authorization", "Bearer "+supabaseKey)
		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := httpClient.Do(httpReq)
		if err != nil || (httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated && httpResp.StatusCode != 204) {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update Supabase auth profile", "details": string(bodyBytes)})
			return
		}
		if httpResp != nil {
			httpResp.Body.Close()
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to construct Supabase update request"})
		return
	}

	// Hash password for local fallback (since original system uses models.User password)
	hashedPassword, _ := utils.HashPassword(req.Password)
	var outletID int = 1

	// 3. Database Transaction: Insert User, CustomerDetails, Wallet, Cart, and update Badge
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Insert internal User record
		user := models.User{
			Name:       req.Name,
			Email:      req.Email,
			Password:   &hashedPassword,
			Role:       models.RoleCustomer,
			OutletID:   &outletID,
			IsVerified: true,
			BadgeID:    &req.BadgeID,
		}

		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Create customer details
		customerDetails := models.CustomerDetails{
			UserID: user.ID,
		}
		if err := tx.Create(&customerDetails).Error; err != nil {
			return err
		}

		// Create wallet
		wallet := models.Wallet{
			CustomerID:     customerDetails.ID,
			Balance:        0,
			TotalRecharged: 0,
			TotalUsed:      0,
		}
		if err := tx.Create(&wallet).Error; err != nil {
			return err
		}

		// Create cart
		cart := models.Cart{
			CustomerID: customerDetails.ID,
		}
		if err := tx.Create(&cart).Error; err != nil {
			return err
		}

		// Mark Badge as claimed
		badge.IsClaimed = true
		if err := tx.Save(&badge).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error during final profile setup"})
		return
	}

	// Generate and return local JWT to immediately log them in
	var createdUser models.User
	database.DB.Where("email = ?", req.Email).First(&createdUser)

	token, err := utils.GenerateToken(createdUser.ID, createdUser.Email, createdUser.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate session token"})
		return
	}

	// Set cookie
	c.SetCookie("token", token, 7*24*60*60, "/", "", config.AppConfig.CookieSecure == "true", true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile setup successfully",
		"token":   token,
		"user": gin.H{
			"id":       createdUser.ID,
			"name":     createdUser.Name,
			"email":    createdUser.Email,
			"role":     createdUser.Role,
			"outletId": createdUser.OutletID,
			"badgeId":  createdUser.BadgeID,
		},
	})
}
