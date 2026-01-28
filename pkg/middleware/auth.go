package middleware

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/utils"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthenticateToken middleware - mirrors Express.js authenticateToken
func AuthenticateToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from cookie or Authorization header
		token := ""
		
		// Check cookie first
		if cookieToken, err := c.Cookie("token"); err == nil && cookieToken != "" {
			token = cookieToken
		}
		
		// If not in cookie, check Authorization header
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}
		}

		// No token provided
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Access denied. No token provided."})
			c.Abort()
			return
		}

		// Verify token
		claims, err := utils.VerifyToken(token)
		if err != nil {
			if err.Error() == "Token is expired" {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Token expired."})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token."})
			}
			c.Abort()
			return
		}

		// Fetch user/admin from database based on role
		if claims.Role == "ADMIN" {
			var admin models.Admin
			if err := database.DB.
				Preload("Outlets.Outlet").
				Preload("Outlets.Permissions").
				First(&admin, claims.ID).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token. Admin not found."})
				c.Abort()
				return
			}

			if !admin.IsVerified {
				c.JSON(http.StatusForbidden, gin.H{"message": "Admin not verified."})
				c.Abort()
				return
			}

			// Set admin in context
			c.Set("admin", admin)
			// Also set as user for role-based checks
			c.Set("user", gin.H{
				"id":       admin.ID,
				"email":    admin.Email,
				"name":     admin.Name,
				"role":     "ADMIN",
				"outlets":  admin.Outlets,
			})
		} else {
			// Regular user (CUSTOMER, STAFF, SUPERADMIN)
			var user models.User
			query := database.DB

			if claims.Role == models.RoleCustomer {
				query = query.Preload("CustomerInfo.Wallet").Preload("CustomerInfo.Cart").Preload("Outlet")
			} else if claims.Role == models.RoleStaff {
				query = query.Preload("StaffInfo.Permissions").Preload("Outlet")
			} else if claims.Role == models.RoleSuperAdmin {
				// SuperAdmin might not have extra details, just load Outlet if needed
				query = query.Preload("Outlet")
			}

			if err := query.First(&user, claims.ID).Error; err != nil {
				// Log the error for debugging
				log.Printf("Error fetching user %d (Role: %s): %v", claims.ID, claims.Role, err)
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token. User not found."})
				c.Abort()
				return
			}

			// Check verification for STAFF
			if user.Role == models.RoleStaff && !user.IsVerified {
				c.JSON(http.StatusForbidden, gin.H{"message": "Staff not verified."})
				c.Abort()
				return
			}

			// Set user in context
			c.Set("user", user)
		}

		c.Next()
	}
}

// AuthorizeRoles middleware - check if user has required role
func AuthorizeRoles(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by AuthenticateToken)
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required."})
			c.Abort()
			return
		}

		// Extract role from user
		var userRole models.Role
		
		// Check if it's a User struct or map
		if user, ok := userInterface.(models.User); ok {
			userRole = user.Role
		} else if userMap, ok := userInterface.(gin.H); ok {
			userRole = userMap["role"].(models.Role)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required."})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		// User doesn't have required role
		c.JSON(http.StatusForbidden, gin.H{"message": "Access denied. Insufficient permissions."})
		c.Abort()
	}
}

// RestrictToSuperAdmin - convenience middleware for SUPERADMIN only
func RestrictToSuperAdmin() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		AuthenticateToken()(c)
		if c.IsAborted() {
			return
		}
		AuthorizeRoles(models.RoleSuperAdmin)(c)
	})
}

// RestrictToStaff - convenience middleware for STAFF only
func RestrictToStaff() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		AuthenticateToken()(c)
		if c.IsAborted() {
			return
		}
		AuthorizeRoles(models.RoleStaff)(c)
	})
}

// RestrictToCustomer - convenience middleware for CUSTOMER only
func RestrictToCustomer() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		AuthenticateToken()(c)
		if c.IsAborted() {
			return
		}
		AuthorizeRoles(models.RoleCustomer)(c)
	})
}

// RestrictToSuperAdminOrAdmin - allow SUPERADMIN or ADMIN
func RestrictToSuperAdminOrAdmin() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		AuthenticateToken()(c)
		if c.IsAborted() {
			return
		}
		AuthorizeRoles(models.RoleSuperAdmin, models.RoleAdmin)(c)
	})
}

// RestrictToSuperAdminOrAdminOrCustomer - allow SUPERADMIN, ADMIN, or CUSTOMER
func RestrictToSuperAdminOrAdminOrCustomer() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		AuthenticateToken()(c)
		if c.IsAborted() {
			return
		}
		AuthorizeRoles(models.RoleSuperAdmin, models.RoleAdmin, models.RoleCustomer)(c)
	})
}

// RestrictToStaffWithPermission - check if staff has specific permission
func RestrictToStaffWithPermission(permissionType models.PermissionType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First authenticate
		AuthenticateToken()(c)
		if c.IsAborted() {
			return
		}

		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required."})
			c.Abort()
			return
		}

		user, ok := userInterface.(models.User)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required."})
			c.Abort()
			return
		}

		// Allow ADMIN or SUPERADMIN
		if user.Role == models.RoleSuperAdmin {
			c.Next()
			return
		}

		// Check if STAFF with permission
		if user.Role != models.RoleStaff {
			c.JSON(http.StatusForbidden, gin.H{"message": "Unauthorized: Must be STAFF, ADMIN, or SUPERADMIN."})
			c.Abort()
			return
		}

		// Check staff permissions
		var staffDetails models.StaffDetails
		if err := database.DB.
			Preload("Permissions", "type = ? AND is_granted = ?", permissionType, true).
			Where("user_id = ?", user.ID).
			First(&staffDetails).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"message": "Unauthorized: " + string(permissionType) + " permission required."})
			c.Abort()
			return
		}

		if len(staffDetails.Permissions) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"message": "Unauthorized: " + string(permissionType) + " permission required."})
			c.Abort()
			return
		}

		c.Next()
	}
}
