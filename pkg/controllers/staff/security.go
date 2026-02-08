package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"bytes"
	"image/png"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// ChangePassword updates staff password
func ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Current password, new password, and confirm password are required",
		})
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"message": "New passwords do not match"})
		return
	}

	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "New password must be at least 6 characters long"})
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

	// Verify current password
	if user.Password == nil || *user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Password not set"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to hash password"})
		return
	}

	// Update password
	database.DB.Model(&user).Update("password", string(hashedPassword))

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// Get2FAStatus returns 2FA status
func Get2FAStatus(c *gin.Context) {
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

	var staff models.StaffDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "2FA status fetched successfully",
		"twoFactorEnabled":   staff.TwoFactorEnabled,
		"twoFactorEnabledAt": staff.TwoFactorEnabledAt,
	})
}

// Generate2FASetup generates QR code for 2FA setup
func Generate2FASetup(c *gin.Context) {
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

	var staff models.StaffDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff not found"})
		return
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "HungerBox",
		AccountName: user.Email,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate 2FA key"})
		return
	}

	// Store secret in DB (but not enabled yet)
	secret := key.Secret()
	if err := database.DB.Model(&staff).Update("two_factor_secret", secret).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save 2FA secret"})
		return
	}

	// Generate QR code image
	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate QR code"})
		return
	}
	png.Encode(&buf, img)

	c.JSON(http.StatusOK, gin.H{
		"message":   "2FA setup generated",
		"secret":    secret,
		"otpAuthUrl": key.URL(),
	})
}

// Enable2FA enables 2FA after token verification
func Enable2FA(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Token is required"})
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

	var staff models.StaffDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff not found"})
		return
	}

	if staff.TwoFactorSecret == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "2FA setup not initiated"})
		return
	}

	// Verify token
	valid := totp.Validate(req.Token, *staff.TwoFactorSecret)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 2FA token"})
		return
	}

	// Generate backup codes as strings
	backupCodes := make([]string, 10)
	for i := 0; i < 10; i++ {
		// Simple random 8-digit codes
		key, _ := totp.Generate(totp.GenerateOpts{Issuer: "Backup", AccountName: "Code"})
		backupCodes[i] = key.Secret()[0:8]
	}

	// Enable 2FA
	now := time.Now()
	updates := map[string]interface{}{
		"two_factor_enabled":      true,
		"two_factor_enabled_at":   now,
		"two_factor_backup_codes": backupCodes, // Gorm handles JSON serialization for StringArray
	}

	if err := database.DB.Model(&staff).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to enable 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "2FA enabled successfully",
		"backupCodes": backupCodes,
	})
}

// Disable2FA disables 2FA
func Disable2FA(c *gin.Context) {
	var req struct {
		CurrentPassword string  `json:"currentPassword" binding:"required"`
		Token           *string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Current password is required to disable 2FA"})
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

	var staff models.StaffDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff not found"})
		return
	}

	if !staff.TwoFactorEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"message": "2FA is not enabled"})
		return
	}

	// Verify current password
	if user.Password == nil || *user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Password not set"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Current password is incorrect"})
		return
	}

	// Verify TOTP token if provided and 2FA is enabled
	if staff.TwoFactorEnabled && req.Token != nil && *req.Token != "" {
		if staff.TwoFactorSecret != nil {
			valid := totp.Validate(*req.Token, *staff.TwoFactorSecret)
			if !valid {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 2FA token"})
				return
			}
		}
	}

	// Disable 2FA
	database.DB.Model(&staff).Updates(map[string]interface{}{
		"two_factor_enabled":      false,
		"two_factor_secret":       nil,
		"two_factor_backup_codes": nil,
		"two_factor_enabled_at":   nil,
	})

	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled successfully"})
}

// GetBackupCodesCount returns remaining backup codes count
func GetBackupCodesCount(c *gin.Context) {
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

	var staff models.StaffDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff not found"})
		return
	}

	if !staff.TwoFactorEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"message": "2FA is not enabled"})
		return
	}

	backupCodesCount := 0
	if staff.TwoFactorBackupCodes != nil {
		backupCodesCount = len(*staff.TwoFactorBackupCodes)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Backup codes count fetched successfully",
		"remainingCodes": backupCodesCount,
		"totalCodes":     10,
	})
}
