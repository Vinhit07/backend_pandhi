package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateProductStats recalculates product rating statistics
func UpdateProductStats(productID int) error {
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	// Get 30-day feedback
	var feedback30d []models.Feedback
	database.DB.Where("product_id = ? AND created_at >= ?", productID, thirtyDaysAgo).
		Find(&feedback30d)

	// Calculate weighted scores for 30 days
	var totalWeightedSum30d float64
	for _, f := range feedback30d {
		weightedScore := (f.RatingOverall * 0.4) +
			(f.RatingTaste * 0.3) +
			(f.RatingQuality * 0.2) +
			(f.RatingQuantity * 0.1)
		totalWeightedSum30d += weightedScore
	}

	ratingCount30d := len(feedback30d)
	trendScore := 0.0
	if ratingCount30d > 0 {
		trendScore = totalWeightedSum30d / float64(ratingCount30d)
	}

	// Get lifetime feedback
	var feedbackLifetime []models.Feedback
	database.DB.Where("product_id = ?", productID).Find(&feedbackLifetime)

	// Calculate weighted scores for lifetime
	var totalWeightedSumLifetime float64
	for _, f := range feedbackLifetime {
		weightedScore := (f.RatingOverall * 0.4) +
			(f.RatingTaste * 0.3) +
			(f.RatingQuality * 0.2) +
			(f.RatingQuantity * 0.1)
		totalWeightedSumLifetime += weightedScore
	}

	ratingCountLifetime := len(feedbackLifetime)
	averageRatingLifetime := 0.0
	if ratingCountLifetime > 0 {
		averageRatingLifetime = totalWeightedSumLifetime / float64(ratingCountLifetime)
	}

	// Update product
	return database.DB.Model(&models.Product{}).Where("id = ?", productID).Updates(map[string]interface{}{
		"rating_sum_30d":          totalWeightedSum30d,
		"rating_count_30d":        ratingCount30d,
		"trend_score":             trendScore,
		"rating_sum_lifetime":     totalWeightedSumLifetime,
		"rating_count_lifetime":   ratingCountLifetime,
		"average_rating_lifetime": averageRatingLifetime,
	}).Error
}

// SubmitFeedback submits feedback for order items
func SubmitFeedback(c *gin.Context) {
	var req struct {
		OrderID int `json:"orderId" binding:"required"`
		Items   []struct {
			ProductID      int     `json:"productId" binding:"required"`
			RatingOverall  float64 `json:"ratingOverall" binding:"required"`
			RatingTaste    float64 `json:"ratingTaste"`
			RatingQuality  float64 `json:"ratingQuality"`
			RatingQuantity float64 `json:"ratingQuantity"`
			Comment        string  `json:"comment"`
		} `json:"items" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "No feedback items provided"})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "User not found."})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid user data."})
		return
	}

	// Verify order exists and belongs to user
	var order models.Order
	if err := database.DB.
		Joins("JOIN customer_details ON customer_details.id = orders.customer_id").
		Where("orders.id = ? AND customer_details.user_id = ?", req.OrderID, user.ID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Order not found or unauthorized"})
		return
	}

	// Check for existing feedback
	productIDs := make([]int, len(req.Items))
	for i, item := range req.Items {
		productIDs[i] = item.ProductID
	}

	var existingFeedback []models.Feedback
	database.DB.Where("order_id = ? AND product_id IN ?", req.OrderID, productIDs).
		Find(&existingFeedback)

	if len(existingFeedback) > 0 {
		alreadyRatedProducts := make([]int, len(existingFeedback))
		for i, f := range existingFeedback {
			alreadyRatedProducts[i] = f.ProductID
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success":               false,
			"message":               "Some products have already been rated",
			"alreadyRatedProducts":  alreadyRatedProducts,
		})
		return
	}

	// Use transaction to create all feedbacks
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			feedback := models.Feedback{
				UserID:         user.ID,
				OrderID:        req.OrderID,
				ProductID:      item.ProductID,
				RatingOverall:  item.RatingOverall,
				RatingTaste:    item.RatingTaste,
				RatingQuality:  item.RatingQuality,
				RatingQuantity: item.RatingQuantity,
			}
			if item.Comment != "" {
				feedback.Comment = &item.Comment
			}

			if err := tx.Create(&feedback).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to submit feedback. Please try again.",
		})
		return
	}

	// Update product stats asynchronously
	go func() {
		for _, item := range req.Items {
			UpdateProductStats(item.ProductID)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Feedback submitted successfully",
	})
}

// GetPendingFeedback retrieves orders with unrated items
func GetPendingFeedback(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "User not found."})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid user data."})
		return
	}

	// Find delivered orders in last 48 hours
	fortyEightHoursAgo := time.Now().Add(-48 * time.Hour)

	var orders []models.Order
	database.DB.
		Joins("JOIN customer_details ON customer_details.id = orders.customer_id").
		Where("customer_details.user_id = ? AND orders.status = ? AND orders.delivered_at >= ?",
			user.ID, models.OrderStatusDelivered, fortyEightHoursAgo).
		Preload("Items.Product").
		Preload("Feedbacks").
		Order("delivered_at DESC").
		Limit(5).
		Find(&orders)

	var pendingFeedback *gin.H

	// Find first order with unrated items
	for _, order := range orders {
		ratedProductIDs := make(map[int]bool)
		for _, f := range order.Feedbacks {
			ratedProductIDs[f.ProductID] = true
		}

		var unratedItems []gin.H
		for _, item := range order.Items {
			if !ratedProductIDs[item.ProductID] {
				unratedItems = append(unratedItems, gin.H{
					"productId": item.ProductID,
					"name":      item.Product.Name,
					"imageUrl":  item.Product.ImageURL,
				})
			}
		}

		if len(unratedItems) > 0 {
			temp := gin.H{
				"orderId": order.ID,
				"date":    order.DeliveredAt,
				"items":   unratedItems,
			}
			pendingFeedback = &temp
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pending": pendingFeedback,
	})
}

// GetFeedbackStatusForOrder retrieves feedback status for a specific order
func GetFeedbackStatusForOrder(c *gin.Context) {
	orderIDStr := c.Param("orderId")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid order ID"})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "User not found."})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid user data."})
		return
	}

	// Fetch order
	var order models.Order
	if err := database.DB.
		Preload("Items.Product").
		Preload("Feedbacks").
		Preload("Customer").
		First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Order not found"})
		return
	}

	if order.Customer.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Unauthorized"})
		return
	}

	// Identify unrated items
	ratedProductIDs := make(map[int]bool)
	for _, f := range order.Feedbacks {
		ratedProductIDs[f.ProductID] = true
	}

	var unratedItems []gin.H
	for _, item := range order.Items {
		if !ratedProductIDs[item.ProductID] {
			unratedItems = append(unratedItems, gin.H{
				"productId": item.ProductID,
				"name":      item.Product.Name,
			})
		}
	}

	deliveryDate := order.CreatedAt
	if order.DeliveredAt != nil {
		deliveryDate = *order.DeliveredAt
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"orderId":           order.ID,
			"date":              deliveryDate,
			"items":             unratedItems,
			"isFullyRated":      len(unratedItems) == 0,
			"existingFeedbacks": order.Feedbacks,
		},
	})
}

// GetProductReviews retrieves paginated reviews for a product
func GetProductReviews(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid product ID"})
		return
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	skip := (page - 1) * limit

	// Fetch reviews
	var reviews []models.Feedback
	database.DB.
		Where("product_id = ?", productID).
		Preload("User").
		Order("created_at DESC").
		Offset(skip).
		Limit(limit).
		Find(&reviews)

	// Get total count
	var total int64
	database.DB.Model(&models.Feedback{}).
		Where("product_id = ?", productID).
		Count(&total)

	// Format reviews
	formattedReviews := make([]gin.H, len(reviews))
	for i, r := range reviews {
		formattedReviews[i] = gin.H{
			"id":              r.ID,
			"ratingOverall":   r.RatingOverall,
			"ratingTaste":     r.RatingTaste,
			"ratingQuality":   r.RatingQuality,
			"ratingQuantity":  r.RatingQuantity,
			"comment":         r.Comment,
			"createdAt":       r.CreatedAt,
			"user": gin.H{
				"name":  r.User.Name,
				"image": r.User.ImageURL,
			},
		}
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"reviews": formattedReviews,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}
