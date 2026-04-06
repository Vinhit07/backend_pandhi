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

// AddExpense adds a new expense
func AddExpense(c *gin.Context) {
	var req struct {
		OutletID    int     `json:"outletId" binding:"required"`
		Description string  `json:"description" binding:"required"`
		Category    string  `json:"category" binding:"required"`
		Amount      float64 `json:"amount" binding:"required"`
		Method      string  `json:"method" binding:"required"`
		PaidTo      string  `json:"paidTo" binding:"required"`
		ExpenseDate string  `json:"expenseDate" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Please provide all required fields"})
		return
	}

	validMethods := []string{"UPI", "CARD", "CASH", "WALLET"}
	valid := false
	for _, m := range validMethods {
		if req.Method == m {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid payment method. Must be one of: UPI, CARD, CASH, WALLET"})
		return
	}

	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Amount must be a positive number"})
		return
	}

	parsedDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid expenseDate: Must be a valid date"})
		return
	}

	expense := models.Expense{
		OutletID:    req.OutletID,
		Description: req.Description,
		Category:    req.Category,
		Amount:      req.Amount,
		Method:      models.PaymentMethod(req.Method),
		PaidTo:      req.PaidTo,
		ExpenseDate: parsedDate,
	}

	database.DB.Create(&expense)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Expense created successfully",
		"expense": expense,
	})
}

// GetExpenses returns expenses for last 2 weeks
func GetExpenses(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	var outletID int
	var err error

	query := database.DB.Order(`"expenseDate" DESC`)

	if outletIDStr != "ALL" {
		outletID, err = strconv.Atoi(outletIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid outlet ID"})
			return
		}
		query = query.Where(`"outletId" = ?`, outletID)
		fmt.Printf("[DEBUG] GetExpenses - OutletID: %d\n", outletID)
	} else {
		fmt.Printf("[DEBUG] GetExpenses - All Outlets\n")
	}

	var expenses []models.Expense
	if err := query.Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch expenses", "error": err.Error()})
		return
	}
	fmt.Printf("[DEBUG] GetExpenses - Found %d expenses\n", len(expenses))

	message := "Expenses retrieved successfully"
	if len(expenses) == 0 {
		message = "No expenses found"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"data":    expenses,
	})
}

// GetExpenseByDate returns expenses within date range
func GetExpenseByDate(c *gin.Context) {
	var req struct {
		OutletID int    `json:"outletId" binding:"required"`
		From     string `json:"from" binding:"required"`
		To       string `json:"to" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide all the details"})
		return
	}

	from, _ := time.Parse("2006-01-02", req.From)
	to, _ := time.Parse("2006-01-02", req.To)

	var expenses []models.Expense
	query := database.DB.Where(`"expenseDate" >= ? AND "expenseDate" <= ?`, from, to).
		Order(`"expenseDate" DESC`)

	if req.OutletID > 0 {
		query = query.Where(`"outletId" = ?`, req.OutletID)
	}

	query.Find(&expenses)

	c.JSON(http.StatusOK, gin.H{
		"message": "Expenses fetched successfully",
		"count":   len(expenses),
		"data":    expenses,
	})
}
