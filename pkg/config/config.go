package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	Port        string
	Environment string

	// Database
	DatabaseURL string

	// JWT
	JWTSecret    string
	JWTExpiresIn string

	// Session
	SessionSecret string

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string

	// Razorpay
	RazorpayKeyID     string
	RazorpayKeySecret string

	// Twilio
	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioPhoneNumber string

	// Security
	CookieSecure string

	// GCP Storage
	GCPProjectID                 string
	GCPBucketName                string
	GoogleApplicationCredentials string

	// Mobile Auth
	EnableMobileTokenReturn string

	// EC2
	EC2PublicIP string

	// Allowed Origins
	AllowedOrigins string
}

var AppConfig *Config

// LoadConfig loads environment variables into Config struct
func LoadConfig() {
	// Load .env file if it exists (optional in production)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	AppConfig = &Config{
		Port:                         getEnv("PORT", "5500"),
		Environment:                  getEnv("NODE_ENV", "development"),
		DatabaseURL:                  getEnv("DATABASE_URL", ""),
		JWTSecret:                    getEnv("JWT_SECRET", ""),
		JWTExpiresIn:                 getEnv("JWT_EXPIRES_IN", "7d"),
		SessionSecret:                getEnv("SESSION_SECRET", ""),
		GoogleClientID:               getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:           getEnv("GOOGLE_CLIENT_SECRET", ""),
		RazorpayKeyID:                getEnv("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:            getEnv("RAZORPAY_KEY_SECRET", ""),
		TwilioAccountSID:             getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:              getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioPhoneNumber:            getEnv("TWILIO_PHONE_NUMBER", ""),
		CookieSecure:                 getEnv("COOKIE_SECURE", "false"),
		GCPProjectID:                 getEnv("GCP_PROJECT_ID", ""),
		GCPBucketName:                getEnv("GCP_BUCKET_NAME", ""),
		GoogleApplicationCredentials: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		EnableMobileTokenReturn:      getEnv("ENABLE_MOBILE_TOKEN_RETURN", "false"),
		EC2PublicIP:                  getEnv("EC2_PUBLIC_IP", ""),
		AllowedOrigins:               getEnv("ALLOWED_ORIGINS", ""),
	}

	// Validate required config
	if AppConfig.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	log.Println("✅ Configuration loaded successfully")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsProduction returns true if running in production mode
func IsProduction() bool {
	return AppConfig.Environment == "production"
}

// IsDevelopment returns true if running in development mode
func IsDevelopment() bool {
	return AppConfig.Environment == "development" || AppConfig.Environment == ""
}
