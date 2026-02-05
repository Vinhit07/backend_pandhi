package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetDashboardOverview returns overall statistics
func GetDashboardOverview(c *gin.Context) {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	// Optional date filtering via POST body
	c.ShouldBindJSON(&req)

	// Build date filter if provided
	var dateFilter func(*gorm.DB) *gorm.DB
	if req.From != "" && req.To != "" {
		from, _ := time.Parse("2006-01-02", req.From)
		to, _ := time.Parse("2006-01-02", req.To)
		to = to.Add(23*time.Hour + 59*time.Minute)
		dateFilter = func(db *gorm.DB) *gorm.DB {
			return db.Where(`"createdAt" >= ? AND "createdAt" <= ?`, from, to)
		}
	} else {
		dateFilter = func(db *gorm.DB) *gorm.DB { return db }
	}

	var totalActiveOutlets int64
	database.DB.Model(&models.Outlet{}).Where(`"isActive" = ?`, true).Count(&totalActiveOutlets)

	var totalRevenue float64
	database.DB.Model(&models.Order{}).
		Scopes(dateFilter).
		Where("status IN ?", []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Select("COALESCE(SUM(\"totalAmount\"), 0)").Scan(&totalRevenue)

	var totalCustomers int64
	database.DB.Model(&models.CustomerDetails{}).Count(&totalCustomers)

	var totalOrders int64
	database.DB.Model(&models.Order{}).Scopes(dateFilter).Count(&totalOrders)

	// Top performing outlet
	type OutletRevenue struct {
		OutletID    int
		TotalAmount float64
	}
	var topOutlet OutletRevenue
	database.DB.Model(&models.Order{}).
		Scopes(dateFilter).
		Select(`"outletId", SUM("totalAmount") as "totalAmount"`).
		Group(`"outletId"`).
		Order(`"totalAmount" DESC`).
		Limit(1).Scan(&topOutlet)

	var topOutletDetails *models.Outlet
	if topOutlet.OutletID > 0 {
		topOutletDetails = &models.Outlet{}
		database.DB.Select("id, name").First(topOutletDetails, topOutlet.OutletID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"totalActiveOutlets":  totalActiveOutlets,
			"totalRevenue":        totalRevenue,
			"totalCustomers":      totalCustomers,
			"totalOrders":         totalOrders,
			"topPerformingOutlet": topOutletDetails,
		},
		"message": "Dashboard overview fetched successfully",
	})
}
