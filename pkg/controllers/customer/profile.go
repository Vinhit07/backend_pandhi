package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetProfile retrieves the customer's profile information
func GetProfile(c *gin.Context) {
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

	// Load customer info
	var userWithCustomer models.User
	if err := database.DB.
		Preload("CustomerInfo").
		First(&userWithCustomer, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error", "error": err.Error()})
		return
	}

	if userWithCustomer.CustomerInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	response := gin.H{
		"id":          userWithCustomer.ID,
		"customerId":  userWithCustomer.CustomerInfo.ID,
		"name":        userWithCustomer.Name,
		"email":       userWithCustomer.Email,
		"phone":       userWithCustomer.Phone,
		"imageUrl":    nil,
		"bio":         userWithCustomer.CustomerInfo.Bio,
		"yearOfStudy": userWithCustomer.CustomerInfo.YearOfStudy,
		"degree":      userWithCustomer.CustomerInfo.Degree,
	}

	if userWithCustomer.ImageURL != nil {
		signedURL, _ := services.GetSignedURL(*userWithCustomer.ImageURL)
		response["imageUrl"] = signedURL
	}

	c.JSON(http.StatusOK, response)
}

// EditProfile updates the customer's profile information
func EditProfile(c *gin.Context) {
	var req struct {
		Name        *string                `json:"name" form:"name"`
		Phone       *string                `json:"phone" form:"phone"`
		Email       *string                `json:"email" form:"email"`
		Bio         *string                `json:"bio" form:"bio"`
		YearOfStudy *string                `json:"yearOfStudy" form:"yearOfStudy"`
		Degree      *models.TypeOfDegree   `json:"degree" form:"degree"`
	}

	if err := c.ShouldBind(&req); err != nil {
		// Check if at least one field is provided
		if req.Name == nil && req.Phone == nil && req.Email == nil && req.Bio == nil && req.YearOfStudy == nil && req.Degree == nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "No updates provided"})
			return
		}
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

	// Load existing user with customer info
	var existingUser models.User
	if err := database.DB.
		Preload("CustomerInfo").
		First(&existingUser, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error", "error": err.Error()})
		return
	}

	if existingUser.CustomerInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Handle image upload
	file, err := c.FormFile("image")
	if err == nil {
		f, err := file.Open()
		if err == nil {
			defer f.Close()
			imageURL, err := services.UploadImageFromReader(f, file.Filename)
			if err == nil {
				// Delete old image
				if existingUser.ImageURL != nil {
					_ = services.DeleteImage(*existingUser.ImageURL)
				}
				// Update image URL
				database.DB.Model(&existingUser).Update("image_url", imageURL)
				existingUser.ImageURL = &imageURL
			}
		}
	}

	// Update user fields
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}

	if len(updates) > 0 {
		database.DB.Model(&existingUser).Updates(updates)
	}

	// Update customer info fields
	customerUpdates := make(map[string]interface{})
	if req.Bio != nil {
		customerUpdates["bio"] = *req.Bio
	}
	if req.YearOfStudy != nil {
		yearInt, err := strconv.Atoi(*req.YearOfStudy)
		if err == nil {
			customerUpdates["year_of_study"] = yearInt
		}
	}
	if req.Degree != nil {
		customerUpdates["degree"] = *req.Degree
	}

	if len(customerUpdates) > 0 {
		database.DB.Model(&existingUser.CustomerInfo).Updates(customerUpdates)
	}

	// Reload user with updated data
	database.DB.Preload("CustomerInfo").First(&existingUser, user.ID)

	responseUser := gin.H{
		"id":          existingUser.ID,
		"name":        existingUser.Name,
		"email":       existingUser.Email,
		"phone":       existingUser.Phone,
		"imageUrl":    nil,
		"bio":         existingUser.CustomerInfo.Bio,
		"yearOfStudy": existingUser.CustomerInfo.YearOfStudy,
		"degree":      existingUser.CustomerInfo.Degree,
	}

	if existingUser.ImageURL != nil {
		signedURL, _ := services.GetSignedURL(*existingUser.ImageURL)
		responseUser["imageUrl"] = signedURL
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    responseUser,
	})
}
