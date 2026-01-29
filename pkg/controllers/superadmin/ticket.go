package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetTickets returns all tickets for an outlet with customer details
func GetTickets(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide valid OutletId"})
		return
	}

	// Get customers with tickets
	var users []models.User
	database.DB.Where(`"outletId" = ? AND role = ?`, outletID, models.RoleCustomer).
		Preload("CustomerInfo.Tickets").
		Find(&users)

	// Flatten tickets
	allTickets := []gin.H{}
	for _, user := range users {
		if user.CustomerInfo != nil {
			for _, ticket := range user.CustomerInfo.Tickets {
				allTickets = append(allTickets, gin.H{
					"ticketId":       ticket.ID,
					"description":    ticket.Description,
					"priority":       ticket.Priority,
					"status":         ticket.Status,
					"createdAt":      ticket.CreatedAt,
					"customerName":   user.Name,
					"customerEmail":  user.Email,
					"resolutionNote": ticket.ResolutionNote,
					"resolvedAt":     ticket.ResolvedAt,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"tickets": allTickets})
}

// TicketClose closes a ticket with resolution note
func TicketClose(c *gin.Context) {
	var req struct {
		TicketID       int    `json:"ticketId" binding:"required"`
		ResolutionNote string `json:"resolutionNote" binding:"required"`
		ResolvedAt     string `json:"resolvedAt" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Provide ticketId, resolutionNote, and resolvedAt"})
		return
	}

	resolvedTime, _ := time.Parse(time.RFC3339, req.ResolvedAt)

	// Update ticket
	if err := database.DB.Model(&models.Ticket{}).Where("id = ?", req.TicketID).Updates(map[string]interface{}{
		"status":         models.TicketStatusClosed,
		"resolutionNote": req.ResolutionNote,
		"resolvedAt":     resolvedTime,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	var ticket models.Ticket
	database.DB.First(&ticket, req.TicketID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Ticket closed successfully",
		"ticket":  ticket,
	})
}
