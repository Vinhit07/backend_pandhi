package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// OutletAddStaff creates a new staff member
func OutletAddStaff(c *gin.Context) {
	var req struct {
		Email       string   `json:"email" binding:"required"`
		Password    string   `json:"password" binding:"required"`
		Name        string   `json:"name" binding:"required"`
		Phone       string   `json:"phone" binding:"required"`
		OutletID    int      `json:"outletId" binding:"required"`
		StaffRole   string   `json:"staffRole" binding:"required"`
		Permissions []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Please provide email, password, fullName, and phone."})
		return
	}

	// Check existing user
	var existing models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User with this email already exists."})
		return
	}

	// Hash password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	passwordStr := string(hashedPassword)

	// Create user with staff info in transaction
	var newUser models.User
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		newUser = models.User{
			Name:     req.Name,
			Email:    req.Email,
			Phone:    &req.Phone,
			Password: &passwordStr,
			OutletID: &req.OutletID,
			Role:     models.RoleStaff,
		}

		if err := tx.Create(&newUser).Error; err != nil {
			return err
		}

		// Create staff details
		staffInfo := models.StaffDetails{
			UserID:    newUser.ID,
			StaffRole: req.StaffRole,
		}
		if err := tx.Create(&staffInfo).Error; err != nil {
			return err
		}

		// Create permissions
		if len(req.Permissions) > 0 {
			for _, permType := range req.Permissions {
				perm := models.StaffPermission{
					StaffID:   staffInfo.ID,
					Type:      models.PermissionType(permType),
					IsGranted: true,
				}
				tx.Create(&perm)
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Staff user created successfully",
		"user": gin.H{
			"id":    newUser.ID,
			"email": newUser.Email,
			"role":  newUser.Role,
		},
	})
}

// OutletStaffPermission updates staff permissions
func OutletStaffPermission(c *gin.Context) {
	var req struct {
		Permission string `json:"permission" binding:"required"`
		Grant      bool   `json:"grant"`
		StaffID    int    `json:"staffId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	// Find existing permission
	var existing models.StaffPermission
	err := database.DB.Where("staff_id = ? AND type = ?", req.StaffID, req.Permission).First(&existing).Error

	if err == nil {
		// Update existing
		database.DB.Model(&existing).Update("is_granted", req.Grant)
		message := "granted"
		if !req.Grant {
			message = "revoked"
		}
		c.JSON(http.StatusOK, gin.H{
			"message":    fmt.Sprintf("Permission %s", message),
			"permission": existing,
		})
	} else {
		// Create new
		perm := models.StaffPermission{
			StaffID:   req.StaffID,
			Type:      models.PermissionType(req.Permission),
			IsGranted: req.Grant,
		}
		database.DB.Create(&perm)
		message := "granted"
		if !req.Grant {
			message = "revoked"
		}
		c.JSON(http.StatusOK, gin.H{
			"message":    fmt.Sprintf("Permission %s", message),
			"permission": perm,
		})
	}
}

// GetOutletStaff returns all staff for an outlet
func GetOutletStaff(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
	}

	var staffDetails []models.StaffDetails
	database.DB.Joins(`JOIN "User" ON "User".id = "StaffDetails"."userId"`).
		Where(`"User"."outletId" = ? AND "User".role = ?`, outletID, models.RoleStaff).
		Preload("User").
		Preload("Permissions").
		Find(&staffDetails)

	// Get signed URLs for images
	staffsWithSignedURLs := make([]gin.H, len(staffDetails))
	for i, staff := range staffDetails {
		imageURL := ""
		if staff.User.ImageURL != nil {
			signedURL, _ := services.GetSignedURL(*staff.User.ImageURL)
			imageURL = signedURL
		}

		staffsWithSignedURLs[i] = gin.H{
			"id":          staff.ID,
			"userId":      staff.UserID,
			"staffRole":   staff.StaffRole,
			"permissions": staff.Permissions,
			"user": gin.H{
				"id":       staff.User.ID,
				"name":     staff.User.Name,
				"email":    staff.User.Email,
				"phone":    staff.User.Phone,
				"role":     staff.User.Role,
				"outletId": staff.User.OutletID,
				"imageUrl": imageURL,
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    staffsWithSignedURLs,
		"message": "Staff members fetched successfully",
	})
}

// OutletUpdateStaff updates staff member details
func OutletUpdateStaff(c *gin.Context) {
	staffIDStr := c.Param("staffId")
	staffID, err := strconv.Atoi(staffIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid staff ID"})
		return
	}

	// Get staff details
	var staffDetails models.StaffDetails
	if err := database.DB.Preload("User").First(&staffDetails, staffID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff member not found"})
		return
	}

	// Handle multipart form
	name := c.PostForm("name")
	email := c.PostForm("email")
	phone := c.PostForm("phone")
	staffRole := c.PostForm("staffRole")

	imageURL := staffDetails.User.ImageURL

	// Handle image upload
	file, _, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()

		// Delete old image
		if imageURL != nil {
			services.DeleteImage(*imageURL)
		}

		// Upload new image
		fileBytes, _ := io.ReadAll(file)
		newImageURL, uploadErr := services.UploadImage(fileBytes, "staff-image.jpg")
		if uploadErr == nil {
			imageURL = &newImageURL
		}
	}

	// Update user
	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if email != "" {
		updates["email"] = email
	}
	if phone != "" {
		updates["phone"] = phone
	}
	if imageURL != nil {
		updates["imageUrl"] = *imageURL
	}

	database.DB.Model(&staffDetails.User).Updates(updates)

	// Update staff role
	if staffRole != "" {
		database.DB.Model(&staffDetails).Update("staff_role", staffRole)
	}

	// Reload
	database.DB.Preload("User").First(&staffDetails, staffID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Staff updated successfully",
		"staff":   staffDetails,
	})
}

// OutletDeleteStaff deletes a staff member
func OutletDeleteStaff(c *gin.Context) {
	staffIDStr := c.Param("staffId")
	staffID, err := strconv.Atoi(staffIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid staff ID"})
		return
	}

	var staffDetails models.StaffDetails
	if err := database.DB.Preload("User").First(&staffDetails, staffID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff member not found"})
		return
	}

	// Delete in transaction
	database.DB.Transaction(func(tx *gorm.DB) error {
		// Delete permissions
		tx.Where("staff_id = ?", staffID).Delete(&models.StaffPermission{})
		// Delete staff details
		tx.Delete(&staffDetails)
		// Delete user
		tx.Delete(&staffDetails.User)
		return nil
	})

	c.JSON(http.StatusOK, gin.H{"message": "Staff member deleted successfully"})
}

// GetStaffById returns single staff details
func GetStaffById(c *gin.Context) {
	staffIDStr := c.Param("staffId")
	staffID, err := strconv.Atoi(staffIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid staff ID"})
		return
	}

	var staff models.StaffDetails
	if err := database.DB.Preload("User").Preload("Permissions").First(&staff, staffID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff member not found"})
		return
	}

	// Get signed URL for image
	if staff.User.ImageURL != nil {
		signedURL, _ := services.GetSignedURL(*staff.User.ImageURL)
		staff.User.ImageURL = &signedURL
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    staff,
		"message": "Staff details fetched successfully",
	})
}
