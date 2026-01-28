package customer

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateTicket creates a new support ticket
func CreateTicket(c *gin.Context) {
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
		Priority    string `json:"priority" binding:"required"`
		IssueType   string `json:"issueType"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Please provide title, description, and priority"})
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

	// Load customer info
	var userWithCustomer models.User
	if err := database.DB.Preload("CustomerInfo").First(&userWithCustomer, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if userWithCustomer.CustomerInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Create ticket
	ticket := models.Ticket{
		CustomerID:  userWithCustomer.CustomerInfo.ID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    models.Priority(strings.ToUpper(req.Priority)),
		Status:      models.TicketStatusOpen,
	}

	if err := database.DB.Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Reload with customer info
	database.DB.Preload("Customer.User").First(&ticket, ticket.ID)

	issueType := req.IssueType
	if issueType == "" {
		issueType = "General"
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Ticket created successfully",
		"ticket": gin.H{
			"id":           ticket.ID,
			"ticketNumber": fmt.Sprintf("TKT-%d-%03d", time.Now().Year(), ticket.ID),
			"title":        ticket.Title,
			"description":  ticket.Description,
			"priority":     ticket.Priority,
			"status":       ticket.Status,
			"createdAt":    ticket.CreatedAt,
			"issueType":    issueType,
		},
	})
}

// GetCustomerTickets retrieves all tickets for the authenticated customer
func GetCustomerTickets(c *gin.Context) {
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

	// Load customer info
	var userWithCustomer models.User
	if err := database.DB.Preload("CustomerInfo").First(&userWithCustomer, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if userWithCustomer.CustomerInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Fetch tickets
	var tickets []models.Ticket
	if err := database.DB.
		Where("customer_id = ?", userWithCustomer.CustomerInfo.ID).
		Order("created_at DESC").
		Preload("Customer.User").
		Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Format tickets
	var ongoing []gin.H
	var completed []gin.H

	for _, ticket := range tickets {
		progressPercentage := 30
		progress := "In Progress"
		if ticket.ResolutionNote != nil && *ticket.ResolutionNote != "" {
			progressPercentage = 80
			progress = "Waiting for Response"
		}
		if ticket.Status == models.TicketStatusClosed {
			progressPercentage = 100
			progress = "Resolved"
		}

		// Determine issue type based on title
		issueType := "Others"
		if strings.Contains(ticket.Title, "Payment") {
			issueType = "Payment Problems"
		} else if strings.Contains(ticket.Title, "Order") {
			issueType = "Order Issues"
		} else if strings.Contains(ticket.Title, "Account") {
			issueType = "Account Issues"
		} else if strings.Contains(ticket.Title, "Technical") {
			issueType = "Technical Support"
		}

		var resolvedDate *string
		if ticket.ResolvedAt != nil {
			dateStr := ticket.ResolvedAt.Format("2006-01-02")
			resolvedDate = &dateStr
		}

		formattedTicket := gin.H{
			"id":                 strconv.Itoa(ticket.ID),
			"ticketNumber":       fmt.Sprintf("TKT-%d-%03d", ticket.CreatedAt.Year(), ticket.ID),
			"title":              ticket.Title,
			"description":        ticket.Description,
			"priority":           ticket.Priority,
			"status":             strings.ToLower(string(ticket.Status)),
			"progress":           progress,
			"progressPercentage": progressPercentage,
			"dateIssued":         ticket.CreatedAt.Format("2006-01-02"),
			"resolvedDate":       resolvedDate,
			"resolutionNote":     ticket.ResolutionNote,
			"issueType":          issueType,
		}

		if ticket.Status == models.TicketStatusOpen {
			ongoing = append(ongoing, formattedTicket)
		} else {
			completed = append(completed, formattedTicket)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tickets": gin.H{
			"ongoing":   ongoing,
			"completed": completed,
		},
	})
}

// GetTicketDetails retrieves details of a specific ticket
func GetTicketDetails(c *gin.Context) {
	ticketIDStr := c.Param("ticketId")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide valid ticket ID"})
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

	// Load customer info
	var userWithCustomer models.User
	if err := database.DB.Preload("CustomerInfo").First(&userWithCustomer, user.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if userWithCustomer.CustomerInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Customer not found"})
		return
	}

	// Fetch ticket
	var ticket models.Ticket
	if err := database.DB.
		Where("id = ? AND customer_id = ?", ticketID, userWithCustomer.CustomerInfo.ID).
		Preload("Customer.User").
		First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Ticket not found"})
		return
	}

	// Format ticket
	progress := "In Progress"
	progressPercentage := 30
	if ticket.ResolutionNote != nil && *ticket.ResolutionNote != "" {
		progress = "Waiting for Response"
		progressPercentage = 80
	}
	if ticket.Status == models.TicketStatusClosed {
		progress = "Resolved"
		progressPercentage = 100
	}

	issueType := "Others"
	if strings.Contains(ticket.Title, "Payment") {
		issueType = "Payment Problems"
	} else if strings.Contains(ticket.Title, "Order") {
		issueType = "Order Issues"
	} else if strings.Contains(ticket.Title, "Account") {
		issueType = "Account Issues"
	} else if strings.Contains(ticket.Title, "Technical") {
		issueType = "Technical Support"
	}

	var resolvedDate *string
	if ticket.ResolvedAt != nil {
		dateStr := ticket.ResolvedAt.Format("2006-01-02")
		resolvedDate = &dateStr
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket": gin.H{
			"id":                 strconv.Itoa(ticket.ID),
			"ticketNumber":       fmt.Sprintf("TKT-%d-%03d", ticket.CreatedAt.Year(), ticket.ID),
			"title":              ticket.Title,
			"description":        ticket.Description,
			"priority":           ticket.Priority,
			"status":             strings.ToLower(string(ticket.Status)),
			"progress":           progress,
			"progressPercentage": progressPercentage,
			"dateIssued":         ticket.CreatedAt.Format("2006-01-02"),
			"resolvedDate":       resolvedDate,
			"resolutionNote":     ticket.ResolutionNote,
			"issueType":          issueType,
		},
	})
}
