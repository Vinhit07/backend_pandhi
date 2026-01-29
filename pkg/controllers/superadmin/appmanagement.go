package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetOutletAppFeatures returns app features for outlet
func GetOutletAppFeatures(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Valid outletId is required"})
		return
	}

	var outlet models.Outlet
	if err := database.DB.First(&outlet, outletID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Outlet not found"})
		return
	}

	var features []models.OutletAppManagement
	database.DB.Where(`"outletId" = ?`, outletID).Find(&features)

	allFeatures := []string{"APP", "UPI", "LIVE_COUNTER", "COUPONS"}
	featureStatus := make(map[string]bool)

	for _, feature := range allFeatures {
		featureStatus[feature] = false
	}

	for _, f := range features {
		featureStatus[string(f.Feature)] = f.IsEnabled
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Outlet app features fetched successfully",
		"data":    featureStatus,
	})
}

// UpdateOutletAppFeatures updates app features for outlet
func UpdateOutletAppFeatures(c *gin.Context) {
	var req struct {
		OutletID int `json:"outletId" binding:"required"`
		Features []struct {
			Feature   string `json:"feature" binding:"required"`
			IsEnabled bool   `json:"isEnabled"`
		} `json:"features" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "outletId and features array are required"})
		return
	}

	validFeatures := map[string]bool{"APP": true, "UPI": true, "LIVE_COUNTER": true, "COUPONS": true}
	for _, f := range req.Features {
		if !validFeatures[f.Feature] {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid feature. Must be APP, UPI, LIVE_COUNTER, or COUPONS"})
			return
		}
	}

	var outlet models.Outlet
	if err := database.DB.First(&outlet, req.OutletID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Outlet not found"})
		return
	}

	database.DB.Transaction(func(tx *gorm.DB) error {
		for _, f := range req.Features {
			var existing models.OutletAppManagement
			err := tx.Where("outlet_id = ? AND feature = ?", req.OutletID, f.Feature).First(&existing).Error
			if err == nil {
				tx.Model(&existing).Update("is_enabled", f.IsEnabled)
			} else {
				newFeature := models.OutletAppManagement{
					OutletID:  req.OutletID,
					Feature:   models.OutletAppFeature(f.Feature),
					IsEnabled: f.IsEnabled,
				}
				tx.Create(&newFeature)
			}
		}
		return nil
	})

	c.JSON(http.StatusOK, gin.H{"message": "Outlet app features updated successfully"})
}

// GetOutletNonAvailabilityPreview returns non-availability preview
func GetOutletNonAvailabilityPreview(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Valid outletId is required"})
		return
	}

	var nonAvailable []models.OutletAvailability
	database.DB.Where(`"outletId" = ?`, outletID).Find(&nonAvailable)

	previewData := []gin.H{}
	for _, entry := range nonAvailable {
		previewData = append(previewData, gin.H{
			"date":              entry.Date.Format("2006-01-02"),
			"nonAvailableSlots": entry.NonAvailableSlots,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Outlet non-availability preview fetched",
		"data":    previewData,
	})
}

// SetOutletAvailability sets non-available slots for dates
func SetOutletAvailability(c *gin.Context) {
	var req struct {
		OutletID          int `json:"outletId" binding:"required"`
		NonAvailableDates []struct {
			Date              string   `json:"date" binding:"required"`
			NonAvailableSlots []string `json:"nonAvailableSlots" binding:"required"`
		} `json:"nonAvailableDates" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "outletId and nonAvailableDates array are required"})
		return
	}

	database.DB.Transaction(func(tx *gorm.DB) error {
		// Delete existing
		tx.Where(`"outletId" = ?`, req.OutletID).Delete(&models.OutletAvailability{})

		// Create new
		for _, entry := range req.NonAvailableDates {
			parsedDate, err := time.Parse("2006-01-02", entry.Date)
			if err != nil {
				continue
			}


			// Convert []string to models.JSONArray ([]interface{})
			jsonArray := make(models.JSONArray, len(entry.NonAvailableSlots))
			for i, v := range entry.NonAvailableSlots {
				jsonArray[i] = v
			}

			availability := models.OutletAvailability{
				OutletID:          req.OutletID,
				Date:              parsedDate,
				NonAvailableSlots: jsonArray,
			}
			tx.Create(&availability)
		}
		return nil
	})

	c.JSON(http.StatusOK, gin.H{"message": "Outlet availability updated successfully"})
}

// GetAvailableDatesAndSlots returns available dates and slots for next 30 days
func GetAvailableDatesAndSlots(c *gin.Context) {
	outletIDStr := c.Param("outletId")
	outletID, err := strconv.Atoi(outletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Valid outletId is required"})
		return
	}

	today := time.Now()
	next30Days := today.AddDate(0, 0, 30)

	var nonAvailable []models.OutletAvailability
	database.DB.Where("outlet_id = ? AND date >= ? AND date <= ?",
		outletID, today, next30Days).Find(&nonAvailable)

	allSlots := []string{"SLOT_11_12", "SLOT_12_13", "SLOT_13_14", "SLOT_14_15", "SLOT_15_16", "SLOT_16_17"}

	availableDates := []gin.H{}
	for d := today; d.Before(next30Days) || d.Equal(next30Days); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")

		var nonAvailEntry *models.OutletAvailability
		for i := range nonAvailable {
			if nonAvailable[i].Date.Format("2006-01-02") == dateStr {
				nonAvailEntry = &nonAvailable[i]
				break
			}
		}

		availableSlots := []string{}
		if nonAvailEntry != nil {
			for _, slot := range allSlots {
				found := false
				for _, nas := range nonAvailEntry.NonAvailableSlots {
					if nas == slot {
						found = true
						break
					}
				}
				if !found {
					availableSlots = append(availableSlots, slot)
				}
			}
		} else {
			availableSlots = allSlots
		}

		if len(availableSlots) > 0 {
			availableDates = append(availableDates, gin.H{
				"date":           dateStr,
				"availableSlots": availableSlots,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Available dates and slots fetched",
		"data":    availableDates,
	})
}
