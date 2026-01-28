package routes

import (
	"backend_pandhi/pkg/controllers/auth"
	"backend_pandhi/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers all authentication routes
func RegisterAuthRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth")
	{
		// Customer auth
		authGroup.POST("/signup", auth.CustomerSignup)
		authGroup.POST("/signin", auth.CustomerSignIn)

		// Staff auth
		authGroup.POST("/staff-signup", auth.StaffSignup) // TODO: Add uploadDocuments middleware
		authGroup.POST("/staff-signin", auth.StaffSignIn)

		// Admin auth
		authGroup.POST("/admin-signup", auth.AdminSignup) // TODO: Add uploadDocuments middleware
		authGroup.POST("/admin-signin", auth.AdminSignIn)

		// SuperAdmin auth
		authGroup.POST("/superadmin-signin", auth.SuperAdminSignIn)

		// Protected routes
		authGroup.GET("/me", middleware.AuthenticateToken(), auth.CheckAuth)

		// Sign out
		authGroup.POST("/signout", auth.SignOut)
	}
}
