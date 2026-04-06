package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StandardResponse represents a standard API response structure
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SuccessResponse sends a successful response with data
func SuccessResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// SuccessResponseWithData sends a successful response with data only (no message)
func SuccessResponseWithData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    data,
	})
}

// CreatedResponse sends a 201 created response
func CreatedResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusCreated, StandardResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// ErrorResponse sends an error response
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, StandardResponse{
		Success: false,
		Error:   message,
	})
}

// BadRequestResponse sends a 400 bad request response
func BadRequestResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

// UnauthorizedResponse sends a 401 unauthorized response
func UnauthorizedResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, message)
}

// ForbiddenResponse sends a 403 forbidden response
func ForbiddenResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, message)
}

// NotFoundResponse sends a 404 not found response
func NotFoundResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

// InternalServerErrorResponse sends a 500 internal server error response
func InternalServerErrorResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}
