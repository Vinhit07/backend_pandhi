package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetOutletSalesReport returns sales by product
func GetOutletSalesReport(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var req struct {
		From string `json:"from" binding:"required"`
		To   string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "from and to dates are required"})
		return
	}

	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)

	type SalesData struct {
		ProductID   int
		ProductName string
		TotalOrders int
	}
	var sales []SalesData
	database.DB.Table("order_items").
		Select("order_items.product_id, products.name as product_name, SUM(order_items.quantity) as total_orders").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.outlet_id = ? AND orders.created_at >= ? AND orders.created_at <= ? AND orders.status IN ?",
			outletID, from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Group("order_items.product_id, products.name").
		Scan(&sales)

	c.JSON(http.StatusOK, sales)
}

// GetOutletRevenueByItems returns revenue by product
func GetOutletRevenueByItems(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var req struct {
		From string `json:"from" binding:"required"`
		To   string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "from and to dates are required"})
		return
	}

	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)

	type RevenueData struct {
		ProductID   int
		ProductName string
		Revenue     float64
	}
	var revenue []RevenueData
	database.DB.Table("order_items").
		Select("order_items.product_id, products.name as product_name, SUM(order_items.quantity * order_items.unit_price) as revenue").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Joins("JOIN products ON products.id = order_items.product_id").
		Where("orders.outlet_id = ? AND orders.created_at >= ? AND orders.created_at <= ? AND orders.status IN ?",
			outletID, from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Group("order_items.product_id, products.name").
		Scan(&revenue)

	c.JSON(http.StatusOK, revenue)
}

// GetRevenueSplit returns revenue by type (APP, MANUAL, WALLET)
func GetRevenueSplit(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var req struct {
		From string `json:"from" binding:"required"`
		To   string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "from and to dates are required"})
		return
	}

	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)

	var appOrderRevenue float64
	database.DB.Model(&models.Order{}).
		Where("outlet_id = ? AND type = ? AND status IN ? AND created_at >= ? AND created_at <= ?",
			outletID, models.OrderTypeApp, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}, from, to).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&appOrderRevenue)

	var manualOrderRevenue float64
	database.DB.Model(&models.Order{}).
		Where("outlet_id = ? AND type = ? AND status IN ? AND created_at >= ? AND created_at <= ?",
			outletID, models.OrderTypeManual, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}, from, to).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&manualOrderRevenue)

	var walletRechargeRevenue float64
	database.DB.Model(&models.WalletTransaction{}).
		Where("status = ? AND created_at >= ? AND created_at <= ?",
			models.WalletTransTypeRecharge, from, to).
		Select("COALESCE(SUM(amount), 0)").Scan(&walletRechargeRevenue)

	totalRevenue := appOrderRevenue + manualOrderRevenue + walletRechargeRevenue

	c.JSON(http.StatusOK, gin.H{
		"revenueByAppOrder":       appOrderRevenue,
		"revenueByManualOrder":    manualOrderRevenue,
		"revenueByWalletRecharge": walletRechargeRevenue,
		"totalRevenue":            totalRevenue,
	})
}

// GetWalletRechargeByDay returns daily wallet recharge revenue
func GetWalletRechargeByDay(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var req struct {
		From string `json:"from" binding:"required"`
		To   string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "from and to dates are required"})
		return
	}

	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)

	type DailyRecharge struct {
		CreatedAt time.Time
		Amount    float64
	}
	var recharges []DailyRecharge
	database.DB.Table("wallet_transactions").
		Select("created_at, amount").
		Joins("JOIN wallets ON wallets.id = wallet_transactions.wallet_id").
		Joins("JOIN customer_details ON customer_details.id = wallets.customer_id").
		Joins("JOIN users ON users.id = customer_details.user_id").
		Where("users.outlet_id = ? AND wallet_transactions.status = ? AND wallet_transactions.created_at >= ? AND wallet_transactions.created_at <= ?",
			outletID, models.WalletTransTypeRecharge, from, to).
		Scan(&recharges)

	dailyRevenue := make(map[string]float64)
	for _, r := range recharges {
		date := r.CreatedAt.Format("2006-01-02")
		dailyRevenue[date] += r.Amount
	}

	result := []gin.H{}
	for date, revenue := range dailyRevenue {
		result = append(result, gin.H{"date": date, "revenue": revenue})
	}

	c.JSON(http.StatusOK, result)
}

// GetProfitLossTrends returns monthly profit/loss for a year
func GetProfitLossTrends(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var req struct {
		Year int `json:"year" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "year is required"})
		return
	}

	yearStart := time.Date(req.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(req.Year, 12, 31, 23, 59, 59, 999, time.UTC)

	// Get orders
	var orders []struct {
		TotalAmount float64
		CreatedAt   time.Time
	}
	database.DB.Model(&models.Order{}).
		Select("total_amount, created_at").
		Where("outlet_id = ? AND status IN ? AND created_at >= ? AND created_at <= ?",
			outletID, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}, yearStart, yearEnd).
		Scan(&orders)

	// Get expenses
	var expenses []struct {
		Amount    float64
		CreatedAt time.Time
	}
	database.DB.Model(&models.Expense{}).
		Select("amount, created_at").
		Where("outlet_id = ? AND created_at >= ? AND created_at <= ?",
			outletID, yearStart, yearEnd).
		Scan(&expenses)

	// Aggregate by month
	monthly := make(map[int]gin.H)
	for m := 1; m <= 12; m++ {
		monthly[m] = gin.H{"sales": 0.0, "expenses": 0.0, "profit": 0.0}
	}

	for _, order := range orders {
		month := int(order.CreatedAt.Month())
		sales := monthly[month]["sales"].(float64)
		monthly[month] = gin.H{
			"sales":    sales + order.TotalAmount,
			"expenses": monthly[month]["expenses"],
			"profit":   monthly[month]["profit"],
		}
	}

	for _, exp := range expenses {
		month := int(exp.CreatedAt.Month())
		expenses := monthly[month]["expenses"].(float64)
		monthly[month] = gin.H{
			"sales":    monthly[month]["sales"],
			"expenses": expenses + exp.Amount,
			"profit":   monthly[month]["profit"],
		}
	}

	result := []gin.H{}
	for m := 1; m <= 12; m++ {
		sales := monthly[m]["sales"].(float64)
		expenses := monthly[m]["expenses"].(float64)
		profit := sales - expenses
		status := "profit"
		if profit < 0 {
			status = "loss"
		}
		result = append(result, gin.H{
			"month":    m,
			"sales":    sales,
			"expenses": expenses,
			"profit":   profit,
			"status":   status,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GetCustomerOverview returns new vs returning customers
func GetCustomerOverview(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var req struct {
		From string `json:"from" binding:"required"`
		To   string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "from and to dates are required"})
		return
	}

	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)

	// Get orders in period
	var orders []models.Order
	database.DB.Where("outlet_id = ? AND created_at >= ? AND created_at <= ? AND customer_id IS NOT NULL",
		outletID, from, to).
		Find(&orders)

	customerIDs := make(map[int]bool)
	for _, order := range orders {
		if order.CustomerID != nil {
			customerIDs[*order.CustomerID] = true
		}
	}

	newCount := 0
	returningCount := 0
	for customerID := range customerIDs {
		var priorOrder models.Order
		err := database.DB.Where("customer_id = ? AND created_at < ?", customerID, from).First(&priorOrder).Error
		if err == nil {
			returningCount++
		} else {
			newCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"newCustomers":       newCount,
		"returningCustomers": returningCount,
	})
}

// GetCustomerPerOrder returns customers per order by day
func GetCustomerPerOrder(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var req struct {
		From string `json:"from" binding:"required"`
		To   string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "from and to dates are required"})
		return
	}

	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)

	var orders []models.Order
	database.DB.Where("outlet_id = ? AND created_at >= ? AND created_at <= ? AND customer_id IS NOT NULL AND status IN ?",
		outletID, from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Find(&orders)

	grouped := make(map[string]map[int]bool)
	orderCounts := make(map[string]int)

	for _, order := range orders {
		date := order.CreatedAt.Format("2006-01-02")
		if grouped[date] == nil {
			grouped[date] = make(map[int]bool)
		}
		if order.CustomerID != nil {
			grouped[date][*order.CustomerID] = true
		}
		orderCounts[date]++
	}

	result := []gin.H{}
	for date, customers := range grouped {
		ordersForDate := orderCounts[date]
		customersPerOrder := 0.0
		if ordersForDate > 0 {
			customersPerOrder = float64(len(customers)) / float64(ordersForDate)
		}
		result = append(result, gin.H{
			"date":              date,
			"customersPerOrder": customersPerOrder,
		})
	}

	c.JSON(http.StatusOK, result)
}
