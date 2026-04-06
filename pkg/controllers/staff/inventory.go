package staff

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetStocks returns current inventory levels for an outlet
func GetStocks(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide outletId"})
		return
	}

	var products []models.Product
	if err := database.DB.
		Where(map[string]interface{}{"outletId": outletID}).
		Preload("Inventory").
		Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if len(products) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No products found for this outlet."})
		return
	}

	stockInfo := make([]gin.H, len(products))
	for i, prod := range products {
		quantity := 0
		threshold := 0
		if prod.Inventory != nil {
			quantity = prod.Inventory.Quantity
			threshold = prod.Inventory.Threshold
		}

		stockInfo[i] = gin.H{
			"id":        prod.ID,
			"name":      prod.Name,
			"category":  prod.Category,
			"price":     prod.Price,
			"quantity":  quantity,
			"threshold": threshold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"stocks": stockInfo})
}

// AddStock adds inventory quantity
func AddStock(c *gin.Context) {
	var req struct {
		ProductID     int `json:"productId" binding:"required"`
		OutletID      int `json:"outletId" binding:"required"`
		AddedQuantity int `json:"addedQuantity" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Required fields are missing"})
		return
	}

	if req.AddedQuantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Added quantity must be greater than 0"})
		return
	}

	// Find inventory
	var inventory models.Inventory
	if err := database.DB.Where(map[string]interface{}{"productId": req.ProductID}).First(&inventory).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Product inventory not found"})
		return
	}

	// Update inventory
	if err := database.DB.Model(&inventory).
		Update("quantity", inventory.Quantity+req.AddedQuantity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update stock"})
		return
	}

	// Create stock history
	database.DB.Create(&models.StockHistory{
		ProductID: req.ProductID,
		OutletID:  req.OutletID,
		Quantity:  req.AddedQuantity,
		Action:    models.StockActionAdd,
	})

	inventory.Quantity += req.AddedQuantity

	c.JSON(http.StatusOK, gin.H{
		"message":          "Stock updated successfully",
		"updatedInventory": inventory,
	})
}

// DeductStock removes inventory quantity
func DeductStock(c *gin.Context) {
	var req struct {
		ProductID int `json:"productId" binding:"required"`
		OutletID  int `json:"outletId" binding:"required"`
		Quantity  int `json:"quantity" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide valid productId, outletId, and quantity."})
		return
	}

	if req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Quantity must be greater than 0"})
		return
	}

	// Find inventory
	var inventory models.Inventory
	if err := database.DB.
		Where(map[string]interface{}{"productId": req.ProductID, "outletId": req.OutletID}).
		First(&inventory).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Inventory record not found."})
		return
	}

	// Check sufficient stock
	if inventory.Quantity < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Insufficient stock available."})
		return
	}

	// Update inventory
	newQuantity := inventory.Quantity - req.Quantity
	database.DB.Model(&inventory).Update("quantity", newQuantity)

	// Create stock history
	database.DB.Create(&models.StockHistory{
		ProductID: req.ProductID,
		OutletID:  req.OutletID,
		Quantity:  req.Quantity,
		Action:    models.StockActionRemove,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":         "Stock deducted successfully",
		"currentQuantity": newQuantity,
	})
}

// StockHistory returns stock movement history for date range
func StockHistory(c *gin.Context) {
	var req struct {
		OutletID  int    `json:"outletId" binding:"required"`
		StartDate string `json:"startDate" binding:"required"`
		EndDate   string `json:"endDate" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "outletId, startDate, and endDate are required."})
		return
	}

	// Parse dates
	from, err1 := time.Parse("2006-01-02", req.StartDate)
	if err1 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid startDate format. Use YYYY-MM-DD"})
		return
	}

	to, err2 := time.Parse("2006-01-02", req.EndDate)
	if err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid endDate format. Use YYYY-MM-DD"})
		return
	}

	// Set end time to end of day
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())

	// Fetch history
	var history []models.StockHistory
	database.DB.
		Where("\"outletId\" = ? AND action IN ? AND timestamp >= ? AND timestamp <= ?",
			req.OutletID,
			[]models.StockAction{models.StockActionAdd, models.StockActionRemove},
			from,
			to,
		).
		Preload("Product").
		Order("timestamp DESC").
		Find(&history)

	c.JSON(http.StatusOK, gin.H{
		"message": "Stock history fetched",
		"history": history,
	})
}
