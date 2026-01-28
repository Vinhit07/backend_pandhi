package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetCart retrieves the user's cart with all items and product details
func GetCart(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "User not found."})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid user data."})
		return
	}

	// Get customer details
	var customer models.CustomerDetails
	if err := database.DB.Where("user_id = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Get cart with items and products
	var cart models.Cart
	err := database.DB.
		Preload("Items.Product.Inventory").
		Where("customer_id = ?", customer.ID).
		First(&cart).Error

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"cart": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"cart": cart})
}

// UpdateCartItem adds or removes items from the cart
func UpdateCartItem(c *gin.Context) {
	var req struct {
		ProductID int    `json:"productId" binding:"required"`
		Quantity  int    `json:"quantity" binding:"required"`
		Action    string `json:"action" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input: productId, quantity, and valid action are required"})
		return
	}

	if req.Quantity <= 0 || (req.Action != "add" && req.Action != "remove") {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input: productId, quantity, and valid action are required"})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "User not found."})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid user data."})
		return
	}

	// Get customer details
	var customer models.CustomerDetails
	if err := database.DB.Where("user_id = ?", user.ID).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Get cart
	var cart models.Cart
	if err := database.DB.Where("customer_id = ?", customer.ID).First(&cart).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Cart not found, please contact support"})
		return
	}

	// Check if item exists in cart
	var existingCartItem models.CartItem
	itemExists := database.DB.
		Where("cart_id = ? AND product_id = ?", cart.ID, req.ProductID).
		First(&existingCartItem).Error == nil

	if req.Action == "add" {
		if itemExists {
			// Update quantity
			database.DB.Model(&existingCartItem).Update("quantity", existingCartItem.Quantity+req.Quantity)
		} else {
			// Create new cart item
			newItem := models.CartItem{
				CartID:    cart.ID,
				ProductID: req.ProductID,
				Quantity:  req.Quantity,
			}
			database.DB.Create(&newItem)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Product added to cart"})
		return
	}

	if req.Action == "remove" {
		if !itemExists {
			c.JSON(http.StatusNotFound, gin.H{"message": "Item not found in cart"})
			return
		}

		if req.Quantity > existingCartItem.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Cannot remove " + string(rune(req.Quantity)) + " item(s), only " + string(rune(existingCartItem.Quantity)) + " in cart",
			})
			return
		}

		if req.Quantity == existingCartItem.Quantity {
			// Delete item completely
			database.DB.Delete(&existingCartItem)
			c.JSON(http.StatusOK, gin.H{"message": "Item completely removed from cart"})
		} else {
			// Reduce quantity
			database.DB.Model(&existingCartItem).Update("quantity", existingCartItem.Quantity-req.Quantity)
			c.JSON(http.StatusOK, gin.H{"message": "Item quantity reduced"})
		}
		return
	}
}
