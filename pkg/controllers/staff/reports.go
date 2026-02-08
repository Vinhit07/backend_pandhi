package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetSalesTrend returns revenue by dates
func GetSalesTrend(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
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
	database.DB.Where("\"outletId\" = ? AND \"createdAt\" >= ? AND \"createdAt\" <= ? AND status IN ?",
		outletID,
		from,
		to,
		[]models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered},
	).Select("\"totalAmount\", \"createdAt\"").Find(&orders)

	// Group by date
	dailyRevenue := make(map[string]float64)
	for _, order := range orders {
		date := order.CreatedAt.Format("2006-01-02")
		dailyRevenue[date] += order.TotalAmount
	}

	// Convert to sorted slice
	type DailyData struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
	}
	result := []DailyData{}
	for date, revenue := range dailyRevenue {
		result = append(result, DailyData{Date: date, Revenue: revenue})
	}

	c.JSON(http.StatusOK, result)
}

// GetOrderTypeBreakdown returns manual vs app order count
func GetOrderTypeBreakdown(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
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

	var appOrders, manualOrders int64
	database.DB.Model(&models.Order{}).Where("\"outletId\" = ? AND type = ? AND \"createdAt\" >= ? AND \"createdAt\" <= ?",
		outletID, models.OrderTypeApp, from, to).Count(&appOrders)
	database.DB.Model(&models.Order{}).Where("\"outletId\" = ? AND type = ? AND \"createdAt\" >= ? AND \"createdAt\" <= ?",
		outletID, models.OrderTypeManual, from, to).Count(&manualOrders)

	c.JSON(http.StatusOK, gin.H{
		"appOrders":    appOrders,
		"manualOrders": manualOrders,
	})
}

// GetNewCustomersTrend returns new customers by date
func GetNewCustomersTrend(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
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

	var users []models.User
	database.DB.Where("\"outletId\" = ? AND role = ? AND \"createdAt\" >= ? AND \"createdAt\" <= ?",
		outletID, models.RoleCustomer, from, to).Select("\"createdAt\"").Find(&users)

	// Group by date
	dailyNewCustomers := make(map[string]int)
	for _, user := range users {
		date := user.CreatedAt.Format("2006-01-02")
		dailyNewCustomers[date]++
	}

	type DailyData struct {
		Date         string `json:"date"`
		NewCustomers int    `json:"newCustomers"`
	}
	result := []DailyData{}
	for date, count := range dailyNewCustomers {
		result = append(result, DailyData{Date: date, NewCustomers: count})
	}

	c.JSON(http.StatusOK, result)
}

// GetCategoryBreakdown returns order count by category
func GetCategoryBreakdown(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
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

	type CategoryData struct {
		ProductID int
		Quantity  int
	}

	var categoryData []CategoryData
	database.DB.Model(&models.OrderItem{}).
		Select("\"productId\" as product_id, SUM(quantity) as quantity").
		Joins("JOIN \"Order\" ON \"Order\".id = \"OrderItem\".\"orderId\"").
		Where("\"Order\".\"outletId\" = ? AND \"Order\".\"createdAt\" >= ? AND \"Order\".\"createdAt\" <= ? AND \"Order\".status IN ?",
			outletID, from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Group("\"productId\"").
		Scan(&categoryData)

	// Get product categories
	productIDs := make([]int, len(categoryData))
	for i, data := range categoryData {
		productIDs[i] = data.ProductID
	}

	var products []models.Product
	database.DB.Where("id IN ?", productIDs).Select("id, category").Find(&products)

	productCategoryMap := make(map[int]string)
	for _, product := range products {
		productCategoryMap[product.ID] = string(product.Category)
	}

	// Aggregate by category
	categoryTotals := make(map[string]int)
	for _, data := range categoryData {
		category := productCategoryMap[data.ProductID]
		categoryTotals[category] += data.Quantity
	}

	type Result struct {
		Category   string `json:"category"`
		OrderCount int    `json:"orderCount"`
	}
	result := []Result{}
	for category, count := range categoryTotals {
		result = append(result, Result{Category: category, OrderCount: count})
	}

	c.JSON(http.StatusOK, result)
}

// GetDeliveryTimeOrders returns orders by delivery time slot
func GetDeliveryTimeOrders(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
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

	type SlotData struct {
		DeliverySlot string
		Count        int64
	}

	var slotData []SlotData
	database.DB.Model(&models.Order{}).
		Select("\"deliverySlot\" as delivery_slot, COUNT(*) as count").
		Where("\"outletId\" = ? AND \"createdAt\" >= ? AND \"createdAt\" <= ? AND status IN ? AND \"deliverySlot\" IS NOT NULL",
			outletID, from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Group("\"deliverySlot\"").
		Scan(&slotData)

	type Result struct {
		DeliverySlot string `json:"deliverySlot"`
		OrderCount   int64  `json:"orderCount"`
	}
	result := make([]Result, len(slotData))
	for i, data := range slotData {
		result[i] = Result{DeliverySlot: data.DeliverySlot, OrderCount: data.Count}
	}

	c.JSON(http.StatusOK, result)
}

// GetCancellationRefunds returns cancelled orders and refunds by date
func GetCancellationRefunds(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
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

	// Get cancelled orders
	var cancelledOrders []models.Order
	database.DB.Where("\"outletId\" = ? AND \"createdAt\" >= ? AND \"createdAt\" <= ? AND status IN ?",
		outletID, from, to, []models.OrderStatus{models.OrderStatusCancelled, models.OrderStatusPartialCancel}).
		Select("\"createdAt\", status").Find(&cancelledOrders)

	// Get refunds
	var refunds []models.WalletTransaction
	database.DB.
		Joins("JOIN \"Wallet\" ON \"Wallet\".id = \"WalletTransaction\".\"walletId\"").
		Joins("JOIN \"CustomerDetails\" ON \"CustomerDetails\".id = \"Wallet\".\"customerId\"").
		Joins("JOIN \"User\" ON \"User\".id = \"CustomerDetails\".\"userId\"").
		Where("\"User\".\"outletId\" = ? AND \"WalletTransaction\".status = ? AND \"WalletTransaction\".\"createdAt\" >= ? AND \"WalletTransaction\".\"createdAt\" <= ?",
			outletID, models.WalletTransTypeDeduct, from, to).
		Select("\"WalletTransaction\".\"createdAt\"").Find(&refunds)

	// Group by date
	dailyData := make(map[string]struct {
		Cancellations int `json:"cancellations"`
		Refunds       int `json:"refunds"`
	})

	for _, order := range cancelledOrders {
		date := order.CreatedAt.Format("2006-01-02")
		data := dailyData[date]
		data.Cancellations++
		dailyData[date] = data
	}

	for _, refund := range refunds {
		date := refund.CreatedAt.Format("2006-01-02")
		data := dailyData[date]
		data.Refunds++
		dailyData[date] = data
	}

	type Result struct {
		Date          string `json:"date"`
		Cancellations int    `json:"cancellations"`
		Refunds       int    `json:"refunds"`
	}
	result := []Result{}
	for date, data := range dailyData {
		result = append(result, Result{Date: date, Cancellations: data.Cancellations, Refunds: data.Refunds})
	}

	c.JSON(http.StatusOK, result)
}

// GetQuantitySold returns quantity sold by product
func GetQuantitySold(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
		return
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

	type QuantityData struct {
		ProductID int
		Quantity  int
	}

	var quantityData []QuantityData
	database.DB.Model(&models.OrderItem{}).
		Select("\"productId\" as product_id, SUM(quantity) as quantity").
		Joins("JOIN \"Order\" ON \"Order\".id = \"OrderItem\".\"orderId\"").
		Where("\"Order\".\"outletId\" = ? AND \"Order\".\"createdAt\" >= ? AND \"Order\".\"createdAt\" <= ? AND \"Order\".status IN ?",
			outletID, from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Group("\"productId\"").
		Scan(&quantityData)

	// Get product names
	productIDs := make([]int, len(quantityData))
	for i, data := range quantityData {
		productIDs[i] = data.ProductID
	}

	var products []models.Product
	database.DB.Where("id IN ?", productIDs).Select("id, name").Find(&products)

	productNameMap := make(map[int]string)
	for _, product := range products {
		productNameMap[product.ID] = product.Name
	}

	type Result struct {
		ProductID    int    `json:"productId"`
		ProductName  string `json:"productName"`
		QuantitySold int    `json:"quantitySold"`
	}
	result := make([]Result, len(quantityData))
	for i, data := range quantityData {
		result[i] = Result{
			ProductID:    data.ProductID,
			ProductName:  productNameMap[data.ProductID],
			QuantitySold: data.Quantity,
		}
	}

	c.JSON(http.StatusOK, result)
}
