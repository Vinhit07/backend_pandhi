package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateScheduledNotification creates a scheduled notification
func CreateScheduledNotification(c *gin.Context) {
	var req struct {
		Title         string  `json:"title" binding:"required"`
		Message       string  `json:"message" binding:"required"`
		Priority      *string `json:"priority"`
		ImageURL      *string `json:"imageUrl"`
		ScheduledDate string  `json:"scheduledDate" binding:"required"`
		ScheduledTime string  `json:"scheduledTime" binding:"required"`
		OutletID      int     `json:"outletId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Title, message, scheduled date, scheduled time, and outlet ID are required"})
		return
	}

	scheduledAtStr := req.ScheduledDate + "T" + req.ScheduledTime + "+05:30"
	scheduledAt, err := time.Parse("2006-01-02T15:04:05Z07:00", scheduledAtStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid date/time format"})
		return
	}

	if scheduledAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Scheduled time must be in the future"})
		return
	}

	priority := "MEDIUM"
	if req.Priority != nil {
		priority = *req.Priority
	}

	notification := models.ScheduledNotification{
		Message:     req.Message,
		Priority:    models.Priority(priority),
		ImageURL:    req.ImageURL,
		ScheduledAt: scheduledAt,
		OutletID:    req.OutletID,
		IsSent:      false,
	}

	database.DB.Create(&notification)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Notification scheduled successfully",
		"data":    notification,
	})
}

// GetScheduledNotifications returns scheduled notifications for an outlet
func GetScheduledNotifications(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var notifications []models.ScheduledNotification
	database.DB.Where("outlet_id = ?", outletID).Find(&notifications)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notifications,
	})
}

// CancelScheduledNotification cancels a scheduled notification
func CancelScheduledNotification(c *gin.Context) {
	notificationIDStr := c.Param("notificationId")
	notificationID, _ := strconv.Atoi(notificationIDStr)

	database.DB.Delete(&models.ScheduledNotification{}, notificationID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification cancelled successfully",
	})
}

// SendImmediateNotification sends notification to all outlet customers
func SendImmediateNotification(c *gin.Context) {
	var req struct {
		Title    string `json:"title" binding:"required"`
		Message  string `json:"message" binding:"required"`
		OutletID int    `json:"outletId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Title, message, and outlet ID are required"})
		return
	}

	// Get device tokens for customers
	var deviceTokens []models.UserDeviceToken
	database.DB.Joins("JOIN users ON users.id = user_device_tokens.user_id").
		Where("users.outlet_id = ? AND users.role = ? AND user_device_tokens.is_active = ?",
			req.OutletID, models.RoleCustomer, true).
		Find(&deviceTokens)

	if len(deviceTokens) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "No device tokens found for this outlet"})
		return
	}

	tokens := []string{}
	for _, dt := range deviceTokens {
		tokens = append(tokens, dt.DeviceToken)
	}

	// Send notifications
	data := map[string]string{
		"outletId": strconv.Itoa(req.OutletID),
		"type":     "immediate",
	}
	results, err := services.SendBulkPushNotifications(tokens, req.Title, req.Message, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to send notification", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Notification sent to %d devices", len(results)),
		"data": gin.H{
			"sentCount":    len(results),
			"totalDevices": len(tokens),
		},
	})
}

// GetNotificationStats returns notification statistics
func GetNotificationStats(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var total int64
	database.DB.Model(&models.ScheduledNotification{}).Where("outlet_id = ?", outletID).Count(&total)

	var sent int64
	database.DB.Model(&models.ScheduledNotification{}).Where("outlet_id = ? AND is_sent = ?", outletID, true).Count(&sent)

	pending := total - sent

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total":   total,
			"sent":    sent,
			"pending": pending,
		},
	})
}

// TestFCMService tests FCM service status
func TestFCMService(c *gin.Context) {
	status := services.GetServiceStatus()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "FCM Service Status",
		"data":    status,
	})
}

// TestSingleDeviceNotification tests notification to single device
func TestSingleDeviceNotification(c *gin.Context) {
	var req struct {
		DeviceToken string `json:"deviceToken" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Message     string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Device token, title, and message are required"})
		return
	}

	data := map[string]string{
		"type":      "test",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	result, err := services.SendPushNotification(req.DeviceToken, req.Title, req.Message, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to send test notification", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test notification sent",
		"data":    result,
	})
}

// GetLowStockNotifications returns low stock items (from dashboard controller in Express)
func GetLowStockNotifications(c *gin.Context) {
	var inventory []models.Inventory
	database.DB.Preload("Product").
		Where("quantity < threshold").
		Find(&inventory)

	lowStock := make([]gin.H, len(inventory))
	for i, inv := range inventory {
		lowStock[i] = gin.H{
			"productId":   inv.ProductID,
			"productName": inv.Product.Name,
			"quantity":    inv.Quantity,
			"threshold":   inv.Threshold,
			"outletId":    inv.OutletID,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(lowStock),
		"data":    lowStock,
	})
}
