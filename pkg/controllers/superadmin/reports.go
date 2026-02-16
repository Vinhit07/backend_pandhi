package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetOutletSalesReport returns sales by product
// GetOutletSalesReport returns sales by product
func GetOutletSalesReport(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int

	// Default query conditions
	whereClause := `"Order"."createdAt" >= ? AND "Order"."createdAt" <= ? AND "Order".status IN (?, ?)`
	args := []interface{}{}

	if outletIDStr != "ALL" {
		var err error
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
		whereClause += ` AND "Order"."outletId" = ?`
		args = append(args, outletID)
	}

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

	// Prepend dates and status to args
	args = append([]interface{}{from, to, models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}, args...)

	type SalesData struct {
		ProductID   int     `gorm:"column:productId" json:"productId"`
		ProductName string  `gorm:"column:productName" json:"productName"`
		Category    string  `gorm:"column:category" json:"category"`
		TotalOrders int     `gorm:"column:totalOrders" json:"totalOrders"`
		Quantity    int     `gorm:"column:quantity" json:"quantity"`
		Revenue     float64 `gorm:"column:revenue" json:"revenue"`
	}
	var sales []SalesData

	err := database.DB.Raw(`
		SELECT 
			"OrderItem"."productId" as "productId",
			"Product".name as "productName",
			COALESCE("Product".category, 'Uncategorized') as "category",
			COUNT(DISTINCT "OrderItem"."orderId") as "totalOrders",
			SUM("OrderItem".quantity) as "quantity",
			SUM("OrderItem".quantity * "OrderItem"."unitPrice") as "revenue"
		FROM "OrderItem"
		JOIN "Order" ON "Order".id = "OrderItem"."orderId"
		JOIN "Product" ON "Product".id = "OrderItem"."productId"
		WHERE `+whereClause+`
		GROUP BY "OrderItem"."productId", "Product".name, "Product".category
	`, args...).Scan(&sales).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sales)
}

// GetOutletRevenueByItems returns revenue by product
func GetOutletRevenueByItems(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int

	query := database.DB.Table(`"OrderItem"`).
		Select(`"OrderItem"."productId", "Product".name as productName, SUM("OrderItem".quantity * "OrderItem"."unitPrice") as revenue`).
		Joins(`JOIN "Order" ON "Order".id = "OrderItem"."orderId"`).
		Joins(`JOIN "Product" ON "Product".id = "OrderItem"."productId"`)

	if outletIDStr != "ALL" {
		var err error
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
		query = query.Where(`"Order"."outletId" = ?`, outletID)
	}

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
		ProductID   int     `json:"productId"`
		ProductName string  `json:"productName"`
		Revenue     float64 `json:"revenue"`
	}
	var revenue []RevenueData

	query.Where(`"Order"."createdAt" >= ? AND "Order"."createdAt" <= ? AND "Order".status IN ?`,
		from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Group(`"OrderItem"."productId", "Product".name`).
		Scan(&revenue)

	c.JSON(http.StatusOK, revenue)
}

// GetRevenueSplit returns revenue by type (APP, MANUAL, WALLET)
// GetRevenueSplit returns revenue by type (APP, MANUAL, WALLET)
func GetRevenueSplit(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	isAll := outletIDStr == "ALL"

	if !isAll {
		var err error
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
	}

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

	// App Order Revenue
	appQuery := database.DB.Model(&models.Order{}).
		Where(`type = ? AND status IN ? AND "createdAt" >= ? AND "createdAt" <= ?`,
			models.OrderTypeApp, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}, from, to)

	if !isAll {
		appQuery = appQuery.Where(`"outletId" = ?`, outletID)
	}
	var appOrderRevenue float64
	appQuery.Select(`COALESCE(SUM("totalAmount"), 0)`).Scan(&appOrderRevenue)

	// Manual Order Revenue
	manualQuery := database.DB.Model(&models.Order{}).
		Where(`type = ? AND status IN ? AND "createdAt" >= ? AND "createdAt" <= ?`,
			models.OrderTypeManual, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}, from, to)

	if !isAll {
		manualQuery = manualQuery.Where(`"outletId" = ?`, outletID)
	}
	var manualOrderRevenue float64
	manualQuery.Select(`COALESCE(SUM("totalAmount"), 0)`).Scan(&manualOrderRevenue)

	// Wallet Recharge Revenue (User -> WalletTransaction)
	// Wallet transactions are linked to wallets, which are linked to users, who have outletId.
	// Join WalletTransaction -> Wallet -> CustomerDetails -> User
	var walletRechargeRevenue float64
	walletQuery := database.DB.Table(`"WalletTransaction"`).
		Joins(`JOIN "Wallet" ON "Wallet".id = "WalletTransaction"."walletId"`).
		Joins(`JOIN "CustomerDetails" ON "CustomerDetails".id = "Wallet"."customerId"`).
		Joins(`JOIN "User" ON "User".id = "CustomerDetails"."userId"`).
		Where(`"WalletTransaction".status = ? AND "WalletTransaction"."createdAt" >= ? AND "WalletTransaction"."createdAt" <= ?`,
			models.WalletTransTypeRecharge, from, to)

	if !isAll {
		walletQuery = walletQuery.Where(`"User"."outletId" = ?`, outletID)
	}

	walletQuery.Select(`COALESCE(SUM("WalletTransaction".amount), 0)`).Scan(&walletRechargeRevenue)

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
	var outletID int

	query := database.DB.Table(`"WalletTransaction"`).Select(`"WalletTransaction"."createdAt", "WalletTransaction".amount`).
		Joins(`JOIN "Wallet" ON "Wallet".id = "WalletTransaction"."walletId"`).
		Joins(`JOIN "CustomerDetails" ON "CustomerDetails".id = "Wallet"."customerId"`).
		Joins(`JOIN "User" ON "User".id = "CustomerDetails"."userId"`)

	if outletIDStr != "ALL" {
		var err error
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
		query = query.Where(`"User"."outletId" = ?`, outletID)
	}

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

	query.Where(`"WalletTransaction".status = ? AND "WalletTransaction"."createdAt" >= ? AND "WalletTransaction"."createdAt" <= ?`,
		models.WalletTransTypeRecharge, from, to).
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
	var outletID int
	isAll := outletIDStr == "ALL"

	if !isAll {
		var err error
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
	}

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
		TotalAmount float64   `gorm:"column:totalAmount"`
		CreatedAt   time.Time `gorm:"column:createdAt"`
	}
	ordersQuery := database.DB.Model(&models.Order{}).
		Select(`"totalAmount", "createdAt"`).
		Where(`status::text IN (?,?) AND "createdAt" >= ? AND "createdAt" <= ?`,
			models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered, yearStart, yearEnd)

	if !isAll {
		ordersQuery = ordersQuery.Where(`"outletId" = ?`, outletID)
	}
	ordersQuery.Scan(&orders)

	// Get expenses
	var expenses []struct {
		Amount    float64   `gorm:"column:amount"`
		CreatedAt time.Time `gorm:"column:createdAt"`
	}
	expensesQuery := database.DB.Model(&models.Expense{}).
		Select(`amount, "createdAt"`).
		Where(`"createdAt" >= ? AND "createdAt" <= ?`, yearStart, yearEnd)

	if !isAll {
		expensesQuery = expensesQuery.Where(`"outletId" = ?`, outletID)
	}
	expensesQuery.Scan(&expenses)

	// DEBUG LOGS
	fmt.Printf("💰 PROFIT/LOSS DEBUG: OutletID=%d, Year=%d\n", outletID, req.Year)
	fmt.Printf("🔍 Orders Found: %d\n", len(orders))
	fmt.Printf("🔍 Expenses Found: %d\n", len(expenses))
	if len(orders) > 0 {
		fmt.Printf("📝 Sample Order: Amount=%.2f, Date=%s\n", orders[0].TotalAmount, orders[0].CreatedAt)
	}

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
		expensesVal := monthly[month]["expenses"].(float64)
		monthly[month] = gin.H{
			"sales":    monthly[month]["sales"],
			"expenses": expensesVal + exp.Amount,
			"profit":   monthly[month]["profit"],
		}
	}

	result := []gin.H{}
	for m := 1; m <= 12; m++ {
		sales := monthly[m]["sales"].(float64)
		expensesVal := monthly[m]["expenses"].(float64)
		profit := sales - expensesVal
		status := "profit"
		if profit < 0 {
			status = "loss"
		}
		result = append(result, gin.H{
			"month":    m,
			"sales":    sales,
			"expenses": expensesVal,
			"profit":   profit,
			"status":   status,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GetCustomerOverview returns new vs returning customers
func GetCustomerOverview(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	isAll := outletIDStr == "ALL"

	if !isAll {
		var err error
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
	}

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
	query := database.DB.Where(`"createdAt" >= ? AND "createdAt" <= ? AND "customerId" IS NOT NULL`, from, to)

	if !isAll {
		query = query.Where(`"outletId" = ?`, outletID)
	}
	query.Find(&orders)

	customerIDs := make(map[int]bool)
	for _, order := range orders {
		if order.CustomerID != nil {
			customerIDs[*order.CustomerID] = true
		}
	}

	newCount := 0
	returningCount := 0
	newRevenue := 0.0
	returningRevenue := 0.0

	for customerID := range customerIDs {
		// Determine if new or returning (has prior order)
		var priorOrder models.Order
		err := database.DB.Where(`"customerId" = ? AND "createdAt" < ?`, customerID, from).First(&priorOrder).Error
		isReturning := err == nil

		if isReturning {
			returningCount++
		} else {
			newCount++
		}

		// Calculate revenue for this customer in the current period from the 'orders' slice
		// We can filter the 'orders' slice we already fetched
		for _, order := range orders {
			if order.CustomerID != nil && *order.CustomerID == customerID {
				if isReturning {
					returningRevenue += order.TotalAmount
				} else {
					newRevenue += order.TotalAmount
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"newCustomers":             newCount,
		"returningCustomers":       returningCount,
		"newCustomerRevenue":       newRevenue,
		"returningCustomerRevenue": returningRevenue,
	})
}

// GetCustomerPerOrder returns customers per order by day
func GetCustomerPerOrder(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	isAll := outletIDStr == "ALL"

	if !isAll {
		var err error
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
	}

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
	query := database.DB.Where(`"createdAt" >= ? AND "createdAt" <= ? AND "customerId" IS NOT NULL AND status IN ?`,
		from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered})

	if !isAll {
		query = query.Where(`"outletId" = ?`, outletID)
	}
	query.Find(&orders)

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
