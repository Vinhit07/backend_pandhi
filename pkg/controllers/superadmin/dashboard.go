package superadmin

import (
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/services"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetDashboardOverview returns overall statistics
func GetDashboardOverview(c *gin.Context) {
	var totalActiveOutlets int64
	database.DB.Model(&models.Outlet{}).Where(`"isActive" = ?`, true).Count(&totalActiveOutlets)

	var totalRevenue float64
	database.DB.Model(&models.Order{}).
		Where("status IN ?", []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Select("COALESCE(SUM(\"totalAmount\"), 0)").Scan(&totalRevenue)

	var totalCustomers int64
	database.DB.Model(&models.CustomerDetails{}).Count(&totalCustomers)

	var totalOrders int64
	database.DB.Model(&models.Order{}).Count(&totalOrders)

	// Top performing outlet
	type OutletRevenue struct {
		OutletID    int
		TotalAmount float64
	}
	var topOutlet OutletRevenue
	database.DB.Model(&models.Order{}).
		Select("\"outletId\", SUM(\"totalAmount\") as \"totalAmount\"").
		Group("\"outletId\"").
		Order("\"totalAmount\" DESC").
		Limit(1).Scan(&topOutlet)

	var topOutletDetails *models.Outlet
	if topOutlet.OutletID > 0 {
		topOutletDetails = &models.Outlet{}
		database.DB.Select("id, name").First(topOutletDetails, topOutlet.OutletID)
	}

	c.JSON(http.StatusOK, gin.H{
		"totalActiveOutlets":  totalActiveOutlets,
		"totalRevenue":        totalRevenue,
		"totalCustomers":      totalCustomers,
		"totalOrders":         totalOrders,
		"topPerformingOutlet": topOutletDetails,
	})
}

// GetRevenueTrend returns daily revenue trend
func GetRevenueTrend(c *gin.Context) {
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
	to = to.Add(23*time.Hour + 59*time.Minute)

	var orders []struct {
		TotalAmount float64
		CreatedAt   time.Time
	}
	database.DB.Model(&models.Order{}).
		Select("\"totalAmount\", \"createdAt\"").
		Where("\"createdAt\" >= ? AND \"createdAt\" <= ? AND status IN ?",
			from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Find(&orders)

	dailyRevenue := make(map[string]float64)
	for _, order := range orders {
		date := order.CreatedAt.Format("2006-01-02")
		dailyRevenue[date] += order.TotalAmount
	}

	result := []gin.H{}
	for date, revenue := range dailyRevenue {
		result = append(result, gin.H{"date": date, "revenue": revenue})
	}

	c.JSON(http.StatusOK, result)
}

// GetOrderStatusDistribution returns order counts by status
func GetOrderStatusDistribution(c *gin.Context) {
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
	to = to.Add(23*time.Hour + 59*time.Minute)

	type StatusCount struct {
		Status models.OrderStatus
		Count  int64
	}
	var statusCounts []StatusCount
	database.DB.Model(&models.Order{}).
		Select("status, COUNT(*) as count").
		Where("\"createdAt\" >= ? AND \"createdAt\" <= ?", from, to).
		Group("status").
		Scan(&statusCounts)

	result := gin.H{
		"delivered":           int64(0),
		"pending":             int64(0),
		"cancelled":           int64(0),
		"partiallyDelivered":  int64(0),
	}

	for _, sc := range statusCounts {
		switch sc.Status {
		case models.OrderStatusDelivered:
			result["delivered"] = sc.Count
		case models.OrderStatusPending:
			result["pending"] = sc.Count
		case models.OrderStatusCancelled:
			result["cancelled"] = sc.Count
		case models.OrderStatusPartiallyDelivered:
			result["partiallyDelivered"] = sc.Count
		}
	}

	c.JSON(http.StatusOK, result)
}

// GetOrderSourceDistribution returns APP vs MANUAL counts
func GetOrderSourceDistribution(c *gin.Context) {
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
	to = to.Add(23*time.Hour + 59*time.Minute)

	type TypeCount struct {
		Type  models.OrderType
		Count int64
	}
	var typeCounts []TypeCount
	database.DB.Model(&models.Order{}).
		Select("type, COUNT(*) as count").
		Where("\"createdAt\" >= ? AND \"createdAt\" <= ?", from, to).
		Group("type").
		Scan(&typeCounts)

	result := gin.H{
		"appOrders":    int64(0),
		"manualOrders": int64(0),
	}

	for _, tc := range typeCounts {
		if tc.Type == models.OrderTypeApp {
			result["appOrders"] = tc.Count
		} else if tc.Type == models.OrderTypeManual {
			result["manualOrders"] = tc.Count
		}
	}

	c.JSON(http.StatusOK, result)
}

// GetTopSellingItems returns top 3 products by quantity
func GetTopSellingItems(c *gin.Context) {
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
	to = to.Add(23*time.Hour + 59*time.Minute)

	type ProductStats struct {
		ProductID    int     `json:"productId"`
		ProductName  string  `json:"productName"`
		TotalOrders  int     `json:"totalOrders"`
		TotalRevenue float64 `json:"totalRevenue"`
	}
	var stats []ProductStats
	database.DB.Table("\"OrderItem\"").
		Select("\"OrderItem\".\"productId\", \"Product\".name as product_name, SUM(\"OrderItem\".quantity) as total_orders, SUM(\"OrderItem\".quantity * \"OrderItem\".\"unitPrice\") as total_revenue").
		Joins("JOIN \"Order\" ON \"Order\".id = \"OrderItem\".\"orderId\"").
		Joins("JOIN \"Product\" ON \"Product\".id = \"OrderItem\".\"productId\"").
		Where("\"Order\".\"createdAt\" >= ? AND \"Order\".\"createdAt\" <= ? AND \"Order\".status IN ?",
			from, to, []models.OrderStatus{models.OrderStatusDelivered, models.OrderStatusPartiallyDelivered}).
		Group("\"OrderItem\".\"productId\", \"Product\".name").
		Order("total_orders DESC").
		Limit(3).
		Scan(&stats)

	c.JSON(http.StatusOK, stats)
}

// GetPeakTimeSlots returns order counts by delivery slot
func GetPeakTimeSlots(c *gin.Context) {
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
	to = to.Add(23*time.Hour + 59*time.Minute)

	type SlotCount struct {
		DeliverySlot string
		Count        int64
	}
	var slots []SlotCount
	database.DB.Model(&models.Order{}).
		Select("\"deliverySlot\", COUNT(*) as count").
		Where("\"createdAt\" >= ? AND \"createdAt\" <= ? AND \"deliverySlot\" IS NOT NULL", from, to).
		Group("\"deliverySlot\"").
		Order("count DESC").
		Scan(&slots)

	result := []gin.H{}
	for _, slot := range slots {
		displayName := formatSlotForDisplay(slot.DeliverySlot)
		result = append(result, gin.H{
			"timeSlot":    slot.DeliverySlot,
			"displayName": displayName,
			"orderCount":  slot.Count,
		})
	}

	c.JSON(http.StatusOK, result)
}

func formatSlotForDisplay(slot string) string {
	if slot == "" {
		return "N/A"
	}
	re := regexp.MustCompile(`SLOT_(\d+)_(\d+)`)
	matches := re.FindStringSubmatch(slot)
	if len(matches) == 3 {
		start, _ := strconv.Atoi(matches[1])
		end, _ := strconv.Atoi(matches[2])
		return fmt.Sprintf("%s - %s", formatHour(start), formatHour(end))
	}
	return slot
}

func formatHour(hour int) string {
	if hour == 12 {
		return "12 PM"
	}
	if hour == 0 {
		return "12 AM"
	}
	if hour < 12 {
		return fmt.Sprintf("%d AM", hour)
	}
	h := hour % 12
	if h == 0 {
		h = 12
	}
	return fmt.Sprintf("%d PM", h)
}

// GetPendingAdminVerifications returns unverified admins
func GetPendingAdminVerifications(c *gin.Context) {
	var admins []models.Admin
	database.DB.Select(`id, email, name, phone, "aadharUrl", "panUrl", "createdAt"`).
		Where(`"isVerified" = ?`, false).
		Find(&admins)

	adminsWithSignedURLs := []gin.H{}
	for _, admin := range admins {
		aadharURL := ""
		panURL := ""
		if admin.AadharURL != nil {
			aadharURL, _ = services.GetSignedURL(*admin.AadharURL)
		}
		if admin.PanURL != nil {
			panURL, _ = services.GetSignedURL(*admin.PanURL)
		}

		adminsWithSignedURLs = append(adminsWithSignedURLs, gin.H{
			"id":        admin.ID,
			"email":     admin.Email,
			"name":      admin.Name,
			"phone":     admin.Phone,
			"aadharUrl": aadharURL,
			"panUrl":    panURL,
			"createdAt": admin.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, adminsWithSignedURLs)
}

// VerifyAdmin verifies admin and assigns outlets
func VerifyAdmin(c *gin.Context) {
	adminIDStr := c.Param("adminId")
	adminID, _ := strconv.Atoi(adminIDStr)

	var req struct {
		OutletIDs []int `json:"outletIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "At least one outletId is required for verification"})
		return
	}

	var admin models.Admin
	if err := database.DB.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Admin not found"})
		return
	}

	if admin.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Admin is already verified"})
		return
	}

	// Validate outlets
	var validOutlets []models.Outlet
	database.DB.Where(`id IN ? AND "isActive" = ?`, req.OutletIDs, true).Find(&validOutlets)
	if len(validOutlets) != len(req.OutletIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "One or more outlets are invalid or inactive"})
		return
	}

	// Begin verification
	database.DB.Transaction(func(tx *gorm.DB) error {
		tx.Model(&admin).Update("isVerified", true)

		// Create AdminOutlet relations
		for _, outletID := range req.OutletIDs {
			adminOutlet := models.AdminOutlet{AdminID: adminID, OutletID: outletID}
			tx.Create(&adminOutlet)

			// Create default permissions
			defaultPerms := []string{
				"ORDER_MANAGEMENT", "STAFF_MANAGEMENT", "INVENTORY_MANAGEMENT",
				"EXPENDITURE_MANAGEMENT", "WALLET_MANAGEMENT", "CUSTOMER_MANAGEMENT",
				"TICKET_MANAGEMENT", "NOTIFICATIONS_MANAGEMENT", "PRODUCT_MANAGEMENT",
				"APP_MANAGEMENT", "REPORTS_ANALYTICS", "SETTINGS", "ONBOARDING", "ADMIN_MANAGEMENT",
			}
			for _, permType := range defaultPerms {
				perm := models.AdminPermission{
					AdminOutletID: adminOutlet.ID,
					Type:          models.AdminPermissionType(permType),
					IsGranted:     false,
				}
				tx.Create(&perm)
			}
		}
		return nil
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin verified successfully",
		"admin": gin.H{
			"id":        admin.ID,
			"name":      admin.Name,
			"email":     admin.Email,
			"outletIds": req.OutletIDs,
		},
	})
}

// GetVerifiedAdmins returns verified admins
func GetVerifiedAdmins(c *gin.Context) {
	var admins []models.Admin
	database.DB.Where(`"isVerified" = ?`, true).
		Preload("Outlets").
		Find(&admins)

	result := []gin.H{}
	for _, admin := range admins {
		outletIDs := []int{}
		for _, ao := range admin.Outlets {
			outletIDs = append(outletIDs, ao.OutletID)
		}

		result = append(result, gin.H{
			"id":        admin.ID,
			"email":     admin.Email,
			"name":      admin.Name,
			"phone":     admin.Phone,
			"createdAt": admin.CreatedAt,
			"outlets":   outletIDs,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GetAdminDetails returns single admin details
func GetAdminDetails(c *gin.Context) {
	adminIDStr := c.Param("adminId")
	adminID, _ := strconv.Atoi(adminIDStr)

	var admin models.Admin
	if err := database.DB.Preload("Outlets.Outlet").Preload("Outlets.Permissions").First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Admin not found"})
		return
	}

	aadharURL := ""
	panURL := ""
	if admin.AadharURL != nil {
		aadharURL, _ = services.GetSignedURL(*admin.AadharURL)
	}
	if admin.PanURL != nil {
		panURL, _ = services.GetSignedURL(*admin.PanURL)
	}

	outlets := []gin.H{}
	for _, ao := range admin.Outlets {
		permissions := []gin.H{}
		for _, p := range ao.Permissions {
			permissions = append(permissions, gin.H{
				"type":      p.Type,
				"isGranted": p.IsGranted,
			})
		}

		outlets = append(outlets, gin.H{
			"outletId": ao.OutletID,
			"outlet": gin.H{
				"name":    ao.Outlet.Name,
				"address": ao.Outlet.Address,
			},
			"permissions": permissions,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         admin.ID,
		"email":      admin.Email,
		"phone":      admin.Phone,
		"aadharUrl":  aadharURL,
		"panUrl":     panURL,
		"createdAt":  admin.CreatedAt,
		"isVerified": admin.IsVerified,
		"outlets":    outlets,
	})
}

// DeleteAdmin deletes an admin
func DeleteAdmin(c *gin.Context) {
	adminIDStr := c.Param("adminId")
	adminID, _ := strconv.Atoi(adminIDStr)

	var admin models.Admin
	if err := database.DB.First(&admin, adminID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Admin not found"})
		return
	}

	// Prevent self-deletion (optional)
	user, _ := c.Get("user")
	if authUser, ok := user.(*models.User); ok && authUser.Role == models.RoleSuperAdmin && authUser.ID == adminID {
		c.JSON(http.StatusForbidden, gin.H{"message": "Cannot delete your own account"})
		return
	}

	database.DB.Delete(&admin)

	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted successfully"})
}

// MapOutletsToAdmin maps outlets to an admin
func MapOutletsToAdmin(c *gin.Context) {
	var req struct {
		AdminID   int   `json:"adminId" binding:"required"`
		OutletIDs []int `json:"outletIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "adminId and a non-empty array of outletIds are required"})
		return
	}

	var admin models.Admin
	if err := database.DB.Preload("Outlets").First(&admin, req.AdminID).Error; err != nil || !admin.IsVerified {
		c.JSON(http.StatusNotFound, gin.H{"message": "Admin not found or not verified"})
		return
	}

	// Validate outlets
	var validOutlets []models.Outlet
	database.DB.Where(`id IN ? AND "isActive" = ?`, req.OutletIDs, true).Find(&validOutlets)
	if len(validOutlets) != len(req.OutletIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "One or more outlets are invalid or inactive"})
		return
	}

	existingOutletIDs := []int{}
	for _, ao := range admin.Outlets {
		existingOutletIDs = append(existingOutletIDs, ao.OutletID)
	}

	newoutlets := []int{}
	for _, oid := range req.OutletIDs {
		found := false
		for _, eoid := range existingOutletIDs {
			if oid == eoid {
				found = true
				break
			}
		}
		if !found {
			newoutlets = append(newoutlets, oid)
		}
	}

	for _, oid := range newoutlets {
		adminOutlet := models.AdminOutlet{AdminID: req.AdminID, OutletID: oid}
		database.DB.Create(&adminOutlet)
	}

	// Reload
	database.DB.Preload("Outlets.Outlet").First(&admin, req.AdminID)

	outlets := []gin.H{}
	for _, ao := range admin.Outlets {
		outlets = append(outlets, gin.H{
			"outletId": ao.OutletID,
			"name":     ao.Outlet.Name,
			"address":  ao.Outlet.Address,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Outlets mapped to admin successfully",
		"admin": gin.H{
			"id":      admin.ID,
			"email":   admin.Email,
			"outlets": outlets,
		},
	})
}

// AssignAdminPermissions assigns permissions to admin for specific outlets
func AssignAdminPermissions(c *gin.Context) {
	var req struct {
		AdminID     int                    `json:"adminId" binding:"required"`
		Permissions map[int][]gin.H `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "adminId and a non-empty permissions object are required"})
		return
	}

	var admin models.Admin
	if err := database.DB.Preload("Outlets").First(&admin, req.AdminID).Error; err != nil || !admin.IsVerified {
		c.JSON(http.StatusNotFound, gin.H{"message": "Admin not found or not verified"})
		return
	}

	adminOutletIDs := map[int]bool{}
	for _, ao := range admin.Outlets {
		adminOutletIDs[ao.OutletID] = true
	}

	// Validate requested outlets
	for outletID := range req.Permissions {
		if !adminOutletIDs[outletID] {
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("Outlet %d is not mapped to this admin", outletID)})
			return
		}
	}

	// Update permissions
	for outletID, perms := range req.Permissions {
		var adminOutlet models.AdminOutlet
		database.DB.Where(`"adminId" = ? AND "outletId" = ?`, req.AdminID, outletID).First(&adminOutlet)

		for _, permObj := range perms {
			permType := models.AdminPermissionType(permObj["type"].(string))
			isGranted, _ := permObj["isGranted"].(bool)

			var existing models.AdminPermission
			err := database.DB.Where(`"adminOutletId" = ? AND type = ?`, adminOutlet.ID, permType).First(&existing).Error
			if err == nil {
				database.DB.Model(&existing).Update("isGranted", isGranted)
			} else {
				perm := models.AdminPermission{
					AdminOutletID: adminOutlet.ID,
					Type:          models.AdminPermissionType(permType),
					IsGranted:     isGranted,
				}
				database.DB.Create(&perm)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Permissions assigned successfully",
		"adminId": req.AdminID,
	})
}

// VerifyStaff verifies a staff member
func VerifyStaff(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, _ := strconv.Atoi(userIDStr)

	var req struct {
		OutletID  int    `json:"outletId" binding:"required"`
		StaffRole string `json:"staffRole"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "outletId is required for verification"})
		return
	}

	var user models.User
	if err := database.DB.Preload("StaffInfo").First(&user, userID).Error; err != nil || user.Role != models.RoleStaff {
		c.JSON(http.StatusNotFound, gin.H{"message": "Staff not found"})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Staff is already verified"})
		return
	}

	database.DB.Model(&user).Updates(map[string]interface{}{
		"isVerified": true,
		"outletId":   req.OutletID,
	})

	// Create or update staff details
	if user.StaffInfo == nil {
		staffInfo := models.StaffDetails{
			UserID:    userID,
			StaffRole: req.StaffRole,
		}
		database.DB.Create(&staffInfo)

		// Create default permissions
		defaultPerms := []string{"BILLING", "PRODUCT_INSIGHTS", "REPORTS", "INVENTORY"}
		for _, permType := range defaultPerms {
			perm := models.StaffPermission{
				StaffID:   staffInfo.ID,
				Type:      models.PermissionType(permType),
				IsGranted: false,
			}
			database.DB.Create(&perm)
		}
	} else {
		database.DB.Model(user.StaffInfo).Update("staffRole", req.StaffRole)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Staff verified successfully",
		"userId":  userID,
	})
}

// GetUnverifiedStaff returns unverified staff
func GetUnverifiedStaff(c *gin.Context) {
	var users []models.User
	database.DB.Where(`role = ? AND "isVerified" = ?`, models.RoleStaff, false).
		Select(`id, name, email, phone, "createdAt"`).
		Find(&users)

	c.JSON(http.StatusOK, users)
}

// GetVerifiedStaff returns verified staff
func GetVerifiedStaff(c *gin.Context) {
	var users []models.User
	database.DB.Where(`role = ? AND "isVerified" = ?`, models.RoleStaff, true).
		Select(`id, name, email, phone, "outletId", "createdAt"`).
		Find(&users)

	c.JSON(http.StatusOK, users)
}
