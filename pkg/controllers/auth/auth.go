package auth

import (
	"backend_pandhi/pkg/config"
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"backend_pandhi/pkg/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CustomerSignup handles customer registration
func CustomerSignup(c *gin.Context) {
	var req struct {
		Name           string  `json:"name" binding:"required"`
		Email          string  `json:"email" binding:"required,email"`
		Password       string  `json:"password" binding:"required"`
		RetypePassword string  `json:"retypePassword" binding:"required"`
		OutletID       *int    `json:"outletId"`
		Phone          *string `json:"phone"`
		YearOfStudy    *int    `json:"yearOfStudy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Name, email, password, and retype password are required"})
		return
	}

	if req.Password != req.RetypePassword {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Passwords do not match"})
		return
	}

	// Default OutletID to 1 if not provided
	var outletID int = 1
	if req.OutletID != nil {
		outletID = *req.OutletID
	}

	// Check if user exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User already exists"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Create user with customer details, wallet, and cart in a transaction
	user := models.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   &hashedPassword,
		Role:       models.RoleCustomer,
		Phone:      req.Phone,
		OutletID:   &outletID,
		IsVerified: true, // Auto-verify customers
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Create user
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Create customer details
		customerDetails := models.CustomerDetails{
			UserID:      user.ID,
			YearOfStudy: req.YearOfStudy,
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

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
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
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Set cookie
	c.SetCookie(
		"token",
		token,
		7*24*60*60, // 7 days
		"/",
		"",
		config.AppConfig.CookieSecure == "true",
		true, // httpOnly
	)

	// Prepare response
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
		"message": "Customer created successfully",
		"user":    response,
	}

	// Add token to response if mobile mode enabled
	if strings.TrimSpace(config.AppConfig.EnableMobileTokenReturn) == "true" {
		jsonResponse["token"] = token
	}

	c.JSON(http.StatusCreated, jsonResponse)
}

// CustomerSignIn handles customer login
func CustomerSignIn(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and password are required"})
		return
	}

	// Find user
	var user models.User
	if err := database.DB.
		Preload("CustomerInfo.Wallet").
		Preload("CustomerInfo.Cart").
		Preload("Outlet").
		Where("email = ?", req.Email).
		First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid customer credentials"})
		return
	}

	if user.Role != models.RoleCustomer {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid customer credentials"})
		return
	}

	// Verify password
	if user.Password == nil || utils.ComparePassword(*user.Password, req.Password) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
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

	// Prepare response
	response := gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"phone":    user.Phone,
		"role":     user.Role,
		"outletId": user.OutletID,
		"outlet":   user.Outlet,
	}

	if user.ImageURL != nil {
		signedURL, _ := services.GetSignedURL(*user.ImageURL)
		response["imageUrl"] = signedURL
	} else {
		response["imageUrl"] = nil
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
		"message": "Customer login successful",
		"user":    response,
	}

	// Add token to response if mobile mode enabled
	if strings.TrimSpace(config.AppConfig.EnableMobileTokenReturn) == "true" {
		jsonResponse["token"] = token
	}

	c.JSON(http.StatusOK, jsonResponse)
}

// StaffSignup handles staff registration
func StaffSignup(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Email          string `json:"email" binding:"required,email"`
		Password       string `json:"password" binding:"required"`
		RetypePassword string `json:"retypePassword" binding:"required"`
		Phone          string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Name, email, password, and retype password are required"})
		return
	}

	if req.Password != req.RetypePassword {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Passwords do not match"})
		return
	}

	// Check if user exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User already exists"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Handle file uploads for aadhar and pan
	var aadharUrl, panUrl string

	// Handle Aadhar upload
	file, err := c.FormFile("aadhar")
	if err == nil {
		f, err := file.Open()
		if err == nil {
			defer f.Close()
			url, err := services.UploadImageFromReader(f, file.Filename)
			if err == nil {
				aadharUrl = url
			}
		}
	}

	// Handle PAN upload
	file, err = c.FormFile("pan")
	if err == nil {
		f, err := file.Open()
		if err == nil {
			defer f.Close()
			url, err := services.UploadImageFromReader(f, file.Filename)
			if err == nil {
				panUrl = url
			}
		}
	}

	// Create user with staff details in a transaction
	user := models.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   &hashedPassword,
		Role:       models.RoleStaff,
		Phone:      &req.Phone,
		IsVerified: false,
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Create user
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Create staff details
		staffDetails := models.StaffDetails{
			UserID:    user.ID,
			StaffRole: "Staff",
		}
		if err := tx.Create(&staffDetails).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Load relationships
	database.DB.
		Preload("StaffInfo.Permissions").
		Preload("Outlet").
		First(&user, user.ID)

	// Prepare response
	response := gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"phone":      user.Phone,
		"role":       user.Role,
		"outletId":   user.OutletID,
		"outlet":     user.Outlet,
		"isVerified": user.IsVerified,
	}

	if user.StaffInfo != nil {
		response["staffDetails"] = gin.H{
			"id":          user.StaffInfo.ID,
			"staffRole":   user.StaffInfo.StaffRole,
			"aadharUrl":   user.StaffInfo.AadharURL,
			"panUrl":      user.StaffInfo.PanURL,
			"permissions": user.StaffInfo.Permissions,
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Staff signup successful. Awaiting SuperAdmin verification.",
		"user":    response,
		"documentsUploaded": gin.H{
			"aadhar": aadharUrl != "",
			"pan":    panUrl != "",
		},
	})
}

// StaffSignIn handles staff login
func StaffSignIn(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and password are required"})
		return
	}

	// Find user
	var user models.User
	if err := database.DB.
		Preload("StaffInfo.Permissions").
		Preload("Outlet").
		Where("email = ?", req.Email).
		First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid staff credentials"})
		return
	}

	if user.Role != models.RoleStaff {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid staff credentials"})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"message": "Staff not verified. Contact SuperAdmin."})
		return
	}

	// Verify password
	if user.Password == nil || utils.ComparePassword(*user.Password, req.Password) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid staff credentials"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
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

	// Prepare response
	response := gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"phone":    user.Phone,
		"role":     user.Role,
		"outletId": user.OutletID,
		"outlet":   user.Outlet,
	}

	if user.ImageURL != nil {
		signedURL, _ := services.GetSignedURL(*user.ImageURL)
		response["imageUrl"] = signedURL
	} else {
		response["imageUrl"] = nil
	}

	if user.StaffInfo != nil {
		response["staffDetails"] = gin.H{
			"id":          user.StaffInfo.ID,
			"staffRole":   user.StaffInfo.StaffRole,
			"permissions": user.StaffInfo.Permissions,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Staff login successful",
		"user":    response,
		"token":   token,
	})
}

// AdminSignup handles admin registration
func AdminSignup(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Email          string `json:"email" binding:"required,email"`
		Password       string `json:"password" binding:"required"`
		RetypePassword string `json:"retypePassword" binding:"required"`
		Phone          string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Name, email, password, and retype password are required"})
		return
	}

	if req.Password != req.RetypePassword {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Passwords do not match"})
		return
	}

	// Check if admin exists
	var existingAdmin models.Admin
	if err := database.DB.Where("email = ?", req.Email).First(&existingAdmin).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Admin already exists"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Handle file uploads for aadhar and pan
	var aadharUrl, panUrl string

	// Handle Aadhar upload
	file, err := c.FormFile("aadhar")
	if err == nil {
		f, err := file.Open()
		if err == nil {
			defer f.Close()
			url, err := services.UploadImageFromReader(f, file.Filename)
			if err == nil {
				aadharUrl = url // Using same UploadImageFromReader since it's generic
			}
		}
	}

	// Handle PAN upload
	file, err = c.FormFile("pan")
	if err == nil {
		f, err := file.Open()
		if err == nil {
			defer f.Close()
			url, err := services.UploadImageFromReader(f, file.Filename)
			if err == nil {
				panUrl = url
			}
		}
	}

	// Create admin
	admin := models.Admin{
		Name:       req.Name,
		Email:      req.Email,
		Password:   hashedPassword,
		IsVerified: false,
		Phone:      &req.Phone,
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin signup successful. Awaiting SuperAdmin verification.",
		"adminId": admin.ID,
		"documentsUploaded": gin.H{
			"aadhar": aadharUrl != "",
			"pan":    panUrl != "",
		},
	})
}

// AdminSignIn handles admin login
func AdminSignIn(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and password are required"})
		return
	}

	// Find admin
	var admin models.Admin
	if err := database.DB.
		Preload("Outlets.Outlet").
		Preload("Outlets.Permissions").
		Where("email = ?", req.Email).
		First(&admin).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	if !admin.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"message": "Admin not verified. Contact SuperAdmin."})
		return
	}

	// Verify password
	if utils.ComparePassword(admin.Password, req.Password) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(admin.ID, admin.Email, "ADMIN")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
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

	// Prepare outlets response
	outlets := make([]gin.H, 0)
	for _, outlet := range admin.Outlets {
		outlets = append(outlets, gin.H{
			"outletId":    outlet.OutletID,
			"outlet":      outlet.Outlet,
			"permissions": outlet.Permissions,
		})
	}

	response := gin.H{
		"id":         admin.ID,
		"name":       admin.Name,
		"email":      admin.Email,
		"role":       "ADMIN",
		"isVerified": admin.IsVerified,
		"outlets":    outlets,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin login successful",
		"admin":   response,
		"token":   token,
	})
}

// SuperAdminSignIn handles superadmin login
func SuperAdminSignIn(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and password are required"})
		return
	}

	// Find user
	var user models.User
	if err := database.DB.
		Preload("Outlet").
		Where("email = ?", req.Email).
		First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	if user.Role != models.RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"message": "Access denied. Only SuperAdmin can log in here."})
		return
	}

	// Verify password
	if user.Password == nil || utils.ComparePassword(*user.Password, req.Password) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
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

	response := gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"phone":    user.Phone,
		"role":     user.Role,
		"outletId": user.OutletID,
		"outlet":   user.Outlet,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "SuperAdmin login successful",
		"user":    response,
		"token":   token,
	})
}

// SignOut handles user logout
func SignOut(c *gin.Context) {
	c.SetCookie(
		"token",
		"",
		-1,
		"/",
		"",
		config.IsProduction(),
		true,
	)
	c.JSON(http.StatusOK, gin.H{"message": "Signed out successfully"})
}

// CheckAuth verifies if user is authenticated and returns user details
func CheckAuth(c *gin.Context) {
	// Get token from cookie or header
	token := ""
	if cookieToken, err := c.Cookie("token"); err == nil {
		token = cookieToken
	}
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 {
			token = parts[1]
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Not authenticated"})
		return
	}

	// Verify token
	claims, err := utils.VerifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
		return
	}

	// Check role and fetch user data
	if claims.Role == "ADMIN" {
		var admin models.Admin
		if err := database.DB.
			Preload("Outlets.Outlet").
			Preload("Outlets.Permissions").
			First(&admin, claims.ID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Admin not found"})
			return
		}

		if !admin.IsVerified {
			c.JSON(http.StatusForbidden, gin.H{"message": "Admin not verified"})
			return
		}

		response := gin.H{
			"id":         admin.ID,
			"name":       admin.Name,
			"email":      admin.Email,
			"role":       "ADMIN",
			"isVerified": admin.IsVerified,
			"outlets":    admin.Outlets,
		}

		c.JSON(http.StatusOK, gin.H{"user": response})
	} else {
		// Regular user (CUSTOMER, STAFF, SUPERADMIN)
		userID, err := strconv.Atoi(strconv.Itoa(claims.ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid token payload"})
			return
		}

		var user models.User
		if err := database.DB.
			Preload("CustomerInfo.Wallet").
			Preload("CustomerInfo.Cart").
			Preload("StaffInfo.Permissions").
			Preload("Outlet").
			First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
			return
		}

		response := gin.H{
			"id":       user.ID,
			"name":     user.Name,
			"email":    user.Email,
			"phone":    user.Phone,
			"role":     user.Role,
			"outletId": user.OutletID,
			"outlet":   user.Outlet,
		}

		if user.ImageURL != nil {
			signedURL, _ := services.GetSignedURL(*user.ImageURL)
			response["imageUrl"] = signedURL
		} else {
			response["imageUrl"] = nil
		}

		if user.CustomerInfo != nil {
			response["customerDetails"] = gin.H{
				"id":          user.CustomerInfo.ID,
				"yearOfStudy": user.CustomerInfo.YearOfStudy,
				"wallet":      user.CustomerInfo.Wallet,
				"cart":        user.CustomerInfo.Cart,
			}
		}

		if user.StaffInfo != nil {
			response["staffDetails"] = gin.H{
				"id":          user.StaffInfo.ID,
				"staffRole":   user.StaffInfo.StaffRole,
				"permissions": user.StaffInfo.Permissions,
			}
		}

		c.JSON(http.StatusOK, gin.H{"user": response})
	}
}
