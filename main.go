package main

import (
	"backend_pandhi/pkg/config"
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/routes"
	"backend_pandhi/pkg/services"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database
	log.Println("🔌 Initializing database connection...")
	if err := database.InitDatabase(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.CloseDatabase()

	// Run migrations (optional - comment out in production if using manual migrations)
	// DISABLED: Database schema is managed by Prisma
	// if config.IsDevelopment() {
	// 	if err := database.AutoMigrate(); err != nil {
	// 		log.Printf("⚠️ Failed to run migrations: %v", err)
	// 	}
	// }

	// Initialize GCP Storage service
	if err := services.InitGCPStorage(); err != nil {
		log.Printf("⚠️  Warning: GCP Storage initialization failed: %v", err)
	} else {
		log.Println("✅ GCP Storage initialized successfully")
	}

	// Initialize FCM service
	if err := services.InitFCM(); err != nil {
		log.Printf("⚠️  Warning: FCM initialization failed: %v", err)
	} else {
		log.Println("✅ FCM initialized successfully")
	}

	// Initialize Razorpay service
	if err := services.InitRazorpay(); err != nil {
		log.Printf("⚠️  Warning: Razorpay initialization failed: %v", err)
	} else {
		log.Println("✅ Razorpay initialized successfully")
	}

	// Set Gin mode based on environment
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize Gin router
	router := gin.Default()

	// Recovery middleware for panic handling
	router.Use(gin.Recovery())

	// Session middleware (matching Express.js session config)
	store := cookie.NewStore([]byte(config.AppConfig.SessionSecret))
	router.Use(sessions.Sessions("session", store))

	// CORS middleware (matching Express.js CORS config)
	setupCORS(router)

	// JSON body size limit (matching Express.js 10mb limit)
	router.MaxMultipartMemory = 10 << 20 // 10 MB

	// Routes
	setupRoutes(router)

	// Start server
	srv := &http.Server{
		Addr:    ":" + config.AppConfig.Port,
		Handler: router,
	}

	// Server startup in goroutine
	go func() {
		log.Printf("🚀 Server running in %s mode\n", config.AppConfig.Environment)
		log.Printf("📡 Server listening on http://localhost:%s\n", config.AppConfig.Port)
		if config.IsProduction() && config.AppConfig.EC2PublicIP != "" {
			log.Printf("🌐 External access: http://%s:%s\n", config.AppConfig.EC2PublicIP, config.AppConfig.Port)
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("✅ Server exited gracefully")
}

// setupCORS configures CORS middleware matching Express.js configuration
func setupCORS(router *gin.Engine) {
	isProduction := config.IsProduction()

	// Default production origins (matching Express.js)
	defaultProductionOrigins := []string{
		"http://admins.mrkadalai.com.s3-website.ap-south-1.amazonaws.com",
		"http://staffs.mrkadalai.com.s3-website.ap-south-1.amazonaws.com",
		"http://localhost:3000",
		"http://localhost:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
		"http://localhost:8081",
		"http://127.0.0.1:8081",
		"http://localhost:3001",
		"http://127.0.0.1:3001",
	}

	var allowOrigins []string
	if isProduction {
		// Use environment variable or default list
		if config.AppConfig.AllowedOrigins != "" {
			// Parse comma-separated origins
			allowOrigins = parseOrigins(config.AppConfig.AllowedOrigins)
		} else {
			allowOrigins = defaultProductionOrigins
		}
	} else {
		// Development: allow default origins (including localhost) instead of "*" to support credentials
		allowOrigins = defaultProductionOrigins
	}

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Range", "X-Content-Range"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if isProduction {
		corsConfig.AllowOrigins = allowOrigins
	} else {
		// In development, trust any localhost origin
		corsConfig.AllowOriginFunc = func(origin string) bool {
			return true // Allow all origins in development
		}
	}

	router.Use(cors.New(corsConfig))

	if isProduction {
		log.Printf("🔒 CORS enabled for origins: %v\n", allowOrigins)
	} else {
		log.Println("🔓 CORS enabled for all origins (development mode)")
	}
}

// parseOrigins splits comma-separated origin string
func parseOrigins(origins string) []string {
	// Split by comma and trim spaces
	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// setupRoutes sets up all application routes
func setupRoutes(router *gin.Engine) {
	// Root route
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "UPS Backend Server is running...")
	})

	// API routes group
	api := router.Group("/api")
	{
		// Register auth routes
		routes.RegisterAuthRoutes(api)

		// Register customer routes
		routes.RegisterCustomerRoutes(api)

		// Register staff routes
		routes.RegisterStaffRoutes(api)

		// Register SuperAdmin routes
		routes.RegisterSuperAdminRoutes(router)

		// Health check route
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":      "ok",
				"environment": config.AppConfig.Environment,
				"database":    "connected",
			})
		})
	}
}
