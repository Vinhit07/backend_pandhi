package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetStocks returns inventory for an outlet
func GetStocks(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide outletId"})
		return
	}

	var products []models.Product
	database.DB.Where(`"outletId" = ?`, outletID).Preload("Inventory").Find(&products)

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

	// Find inventory
	var inventory models.Inventory
	if err := database.DB.Where(`"productId" = ?`, req.ProductID).First(&inventory).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Product inventory not found"})
		return
	}

	// Update inventory
	database.DB.Model(&inventory).Update("quantity", inventory.Quantity+req.AddedQuantity)

	// Create history
	database.DB.Create(&models.StockHistory{
		ProductID: req.ProductID,
		OutletID:  req.OutletID,
		Quantity:  req.AddedQuantity,
		Action:    models.StockActionAdd,
	})

	// Reload
	database.DB.First(&inventory, inventory.ID)

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

	if err := c.ShouldBindJSON(&req); err != nil || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide valid productId, outletId, and quantity."})
		return
	}

	// Find inventory
	var inventory models.Inventory
	if err := database.DB.Where(`"productId" = ?`, req.ProductID).First(&inventory).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Inventory record not found."})
		return
	}

	if inventory.Quantity < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Insufficient stock available"})
		return
	}

	// Update inventory
	database.DB.Model(&inventory).Update("quantity", inventory.Quantity-req.Quantity)

	// Create history
	database.DB.Create(&models.StockHistory{
		ProductID: req.ProductID,
		OutletID:  req.OutletID,
		Quantity:  req.Quantity,
		Action:    models.StockActionRemove,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":         "Stock deducted successfully",
		"currentQuantity": inventory.Quantity - req.Quantity,
	})
}

// StockHistory returns stock movement history
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

	from, _ := time.Parse("2006-01-02", req.StartDate)
	to, _ := time.Parse("2006-01-02", req.EndDate)
	to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	var history []models.StockHistory
	database.DB.Where(`"outletId" = ? AND action IN ? AND timestamp >= ? AND timestamp <= ?`,
		req.OutletID, []models.StockAction{models.StockActionAdd, models.StockActionRemove}, from, to).
		Preload("Product").
		Order("timestamp DESC").
		Find(&history)

	c.JSON(http.StatusOK, gin.H{
		"message": "Stock history fetched",
		"history": history,
	})
}
