package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetProducts returns all products for an outlet
func GetProducts(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, _ := strconv.Atoi(outletIDStr)

	var products []models.Product
	query := database.DB.Preload("Inventory").Order("name ASC")
	if outletID > 0 {
		query = query.Where(`"outletId" = ?`, outletID)
	}
	query.Find(&products)

	// Generate signed URLs
	productsWithSignedURLs := make([]gin.H, len(products))
	for i, product := range products {
		imageURL := ""
		if product.ImageURL != nil {
			signedURL, _ := services.GetSignedURL(*product.ImageURL)
			imageURL = signedURL
		}

		productsWithSignedURLs[i] = gin.H{
			"id":          product.ID,
			"name":        product.Name,
			"description": product.Description,
			"price":       product.Price,
			"imageUrl":    imageURL,
			"outletId":    product.OutletID,
			"category":    product.Category,
			"minValue":    product.MinValue,
			"isVeg":       product.IsVeg,
			"companyPaid": product.CompanyPaid,
			"inventory":   product.Inventory,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    productsWithSignedURLs,
	})
}

// AddProduct creates a new product with image upload
func AddProduct(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	priceStr := c.PostForm("price")
	outletIDStr := c.PostForm("outletId")
	category := c.PostForm("category")
	thresholdStr := c.PostForm("threshold")
	minValueStr := c.PostForm("minValue")
	isVegStr := c.PostForm("isVeg")
	companyPaidStr := c.PostForm("companyPaid")

	if name == "" || description == "" || priceStr == "" || outletIDStr == "" || category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide all the fields"})
		return
	}

	price, _ := strconv.ParseFloat(priceStr, 64)
	outletID, _ := strconv.Atoi(outletIDStr)
	threshold, _ := strconv.Atoi(thresholdStr)
	if threshold == 0 {
		threshold = 10
	}
	minValue, _ := strconv.Atoi(minValueStr)

	isVeg := true
	if isVegStr == "false" {
		isVeg = false
	}

	companyPaid := companyPaidStr == "true"

	crtName := strings.ToLower(name)

	// Check existing
	var existing models.Product
	if err := database.DB.Where("name = ?", crtName).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Product already available"})
		return
	}

	// Handle image upload
	var imageURL *string
	file, _, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()
		fileBytes, _ := io.ReadAll(file)
		uploadedURL, uploadErr := services.UploadImage(fileBytes, "product-image.jpg")
		if uploadErr == nil {
			imageURL = &uploadedURL
		}
	}

	// Create product in transaction
	var newProduct models.Product
	database.DB.Transaction(func(tx *gorm.DB) error {
		newProduct = models.Product{
			Name:        crtName,
			Description: &description,
			Price:       price,
			ImageURL:    imageURL,
			OutletID:    outletID,
			Category:    models.Category(category),
			MinValue:    &minValue,
			IsVeg:       isVeg,
			CompanyPaid: companyPaid,
		}

		if err := tx.Create(&newProduct).Error; err != nil {
			return err
		}

		// Create inventory
		inventory := models.Inventory{
			ProductID: newProduct.ID,
			OutletID:  outletID,
			Threshold: threshold,
			Quantity:  minValue,
		}
		tx.Create(&inventory)

		// Create stock history
		stockHistory := models.StockHistory{
			ProductID: newProduct.ID,
			OutletID:  outletID,
			Quantity:  minValue,
			Action:    models.StockActionAdd,
		}
		tx.Create(&stockHistory)

		return nil
	})

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"name":     newProduct.Name,
			"price":    newProduct.Price,
			"minValue": newProduct.MinValue,
			"imageUrl": newProduct.ImageURL,
		},
		"message": "Product created successfully",
	})
}

// DeleteProduct deletes a product
func DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide productID"})
		return
	}

	result := database.DB.Delete(&models.Product{}, id)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "No product found with that id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Product deleted successfully",
	})
}

// UpdateProduct updates product details with image upload
func UpdateProduct(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid product ID"})
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")
	priceStr := c.PostForm("price")
	category := c.PostForm("category")
	thresholdStr := c.PostForm("threshold")
	minValueStr := c.PostForm("minValue")
	outletIDStr := c.PostForm("outletId")
	isVegStr := c.PostForm("isVeg")
	companyPaidStr := c.PostForm("companyPaid")

	if name == "" || description == "" || priceStr == "" || category == "" || outletIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Missing required fields"})
		return
	}

	price, _ := strconv.ParseFloat(priceStr, 64)
	if price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Price must be greater than 0"})
		return
	}

	// Get existing product
	var existingProduct models.Product
	if err := database.DB.Preload("Inventory").First(&existingProduct, productID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Product not found"})
		return
	}

	crtName := strings.ToLower(name)
	minValue, _ := strconv.Atoi(minValueStr)
	threshold, _ := strconv.Atoi(thresholdStr)
	if threshold == 0 {
		threshold = 10
	}
	outletID, _ := strconv.Atoi(outletIDStr)

	// Check duplicate
	var duplicate models.Product
	if err := database.DB.Where("name = ? AND id != ?", crtName, productID).First(&duplicate).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Product with this name already exists"})
		return
	}

	// Handle image update
	imageURL := existingProduct.ImageURL
	file, _, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()
		fileBytes, _ := io.ReadAll(file)
		uploadedURL, uploadErr := services.UploadImage(fileBytes, "product-image.jpg")
		if uploadErr == nil {
			imageURL = &uploadedURL
		}
	}

	isVeg := existingProduct.IsVeg
	if isVegStr != "" {
		isVeg = isVegStr != "false"
	}

	companyPaid := existingProduct.CompanyPaid
	if companyPaidStr == "true" {
		companyPaid = true
	}

	// Update in transaction
	database.DB.Transaction(func(tx *gorm.DB) error {
		tx.Model(&existingProduct).Updates(map[string]interface{}{
			"name":        crtName,
			"description": description,
			"price":       price,
			"imageUrl":    imageURL,
			"category":    category,
			"minValue":    minValue,
			"outletId":    outletID,
			"isVeg":       isVeg,
			"companyPaid": companyPaid,
		})

		tx.Model(&existingProduct.Inventory).Updates(map[string]interface{}{
			"threshold": threshold,
			"outletId":  outletID,
		})

		tx.Create(&models.StockHistory{
			ProductID: productID,
			OutletID:  outletID,
			Quantity:  existingProduct.Inventory.Quantity,
			Action:    models.StockActionUpdate,
		})

		return nil
	})

	// Reload
	var productWithInventory models.Product
	database.DB.Preload("Inventory").First(&productWithInventory, productID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Product updated successfully",
		"data":    productWithInventory,
	})
}
