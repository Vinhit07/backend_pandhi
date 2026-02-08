package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetStaffProfile retrieves staff profile information
func GetStaffProfile(c *gin.Context) {
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

	// Get staff details with outlet
	var staff models.StaffDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).
		Preload("Outlet").
		First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff not found"})
		return
	}

	// Get user with outlet
	database.DB.Preload("Outlet").First(&user, user.ID)

	var outlet *gin.H
	if user.Outlet != nil {
		outlet = &gin.H{
			"id":      user.Outlet.ID,
			"name":    user.Outlet.Name,
			"address": user.Outlet.Address,
		}
	}

	response := gin.H{
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"phone":       user.Phone,
		"imageUrl":    nil,
		"designation": staff.StaffRole,
		"outlet":      outlet,
	}

	if user.ImageURL != nil {
		signedURL, _ := services.GetSignedURL(*user.ImageURL)
		response["imageUrl"] = signedURL
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Staff profile fetched successfully",
		"profile": response,
	})
}

// UpdateStaffProfile updates staff profile information
func UpdateStaffProfile(c *gin.Context) {
	var req struct {
		Name        *string `json:"name"`
		Phone       *string `json:"phone"`
		Designation *string `json:"designation"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
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

	// Check if at least one field is provided
	if req.Name == nil && req.Phone == nil && req.Designation == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No updates provided"})
		return
	}

	// Get staff details
	var staff models.StaffDetails
	if err := database.DB.Where("\"userId\" = ?", user.ID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff not found"})
		return
	}

	// Update user
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}

	if len(updates) > 0 {
		database.DB.Model(&user).Updates(updates)
	}

	// Update staff designation
	if req.Designation != nil {
		database.DB.Model(&staff).Update("staff_role", *req.Designation)
	}

	// Reload data
	database.DB.Preload("Outlet").First(&user, user.ID)
	database.DB.First(&staff, staff.ID)

	var outlet *gin.H
	if user.Outlet != nil {
		outlet = &gin.H{
			"id":      user.Outlet.ID,
			"name":    user.Outlet.Name,
			"address": user.Outlet.Address,
		}
	}

	response := gin.H{
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"phone":       user.Phone,
		"imageUrl":    nil,
		"designation": staff.StaffRole,
		"outlet":      outlet,
	}

	if user.ImageURL != nil {
		signedURL, _ := services.GetSignedURL(*user.ImageURL)
		response["imageUrl"] = signedURL
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"profile": response,
	})
}

// UploadStaffImage uploads profile image
func UploadStaffImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No image uploaded"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to handle image file"})
		return
	}
	defer f.Close()

	// Upload to GCP
	imageURL, err := services.UploadImageFromReader(f, file.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to upload image", "error": err.Error()})
		return
	}

	userInterface, _ := c.Get("user")
	user := userInterface.(models.User)

	// Delete old image if exists
	if user.ImageURL != nil {
		_ = services.DeleteImage(*user.ImageURL)
	}

	// Update user record
	if err := database.DB.Model(&user).Update("image_url", imageURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update user profile"})
		return
	}

	signedURL, _ := services.GetSignedURL(imageURL)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Image uploaded successfully",
		"imageUrl": signedURL,
	})
}

// DeleteStaffImage deletes profile image
func DeleteStaffImage(c *gin.Context) {
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

	if user.ImageURL == nil || *user.ImageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No image to delete"})
		return
	}

	// Delete from GCP Storage
	if err := services.DeleteImage(*user.ImageURL); err != nil {
		// Log error but continue
	}

	// Clear image URL
	database.DB.Model(&user).Update("image_url", nil)

	c.JSON(http.StatusOK, gin.H{
		"message": "Image deleted successfully",
		"user": gin.H{
			"id":       user.ID,
			"name":     user.Name,
			"imageUrl": nil,
		},
	})
}
