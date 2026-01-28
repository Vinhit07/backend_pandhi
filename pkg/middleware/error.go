package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorMiddleware provides centralized error handling
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there were any errors
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Log the error
			log.Printf("Error: %v", err.Err)

			// Send error response
			var statusCode int
			var message string

			// Determine status code and message based on error type
			if err.Meta != nil {
				if code, ok := err.Meta.(int); ok {
					statusCode = code
				}
			}

			// Default to 500 if no status code set
			if statusCode == 0 {
				statusCode = http.StatusInternalServerError
			}

			// Get error message
			message = err.Error()
			if message == "" {
				message = "Internal server error"
			}

			// Send JSON response (matching Express.js error format)
			c.JSON(statusCode, gin.H{
				"message": message,
			})
		}
	}
}

// RecoveryMiddleware handles panics and prevents server crashes
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				
				// Send error response
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// NotFoundHandler handles 404 errors
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Route not found",
		})
	}
}
