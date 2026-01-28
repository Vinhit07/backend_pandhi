package routes

import (
	"backend_pandhi/pkg/controllers/staff"
	"backend_pandhi/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterStaffRoutes registers all staff-facing API routes
func RegisterStaffRoutes(router *gin.RouterGroup) {
	staffGroup := router.Group("/staff")
	staffGroup.Use(middleware.AuthenticateToken(), middleware.AuthorizeRoles("STAFF"))
	{
		// Home management
		staffGroup.GET("/outlets/get-home-data/", staff.GetHomeDetails)
		staffGroup.GET("/outlets/get-recent-orders/:outletId/", staff.RecentOrders)
		staffGroup.GET("/outlets/get-order/:outletId/:orderId/", staff.GetOrder)
		staffGroup.PUT("/outlets/update-order/", staff.UpdateOrder)
		staffGroup.GET("/outlets/tickets/count", staff.GetTicketsCount)

		// Manual Order
		staffGroup.POST("/outlets/add-manual-order/", staff.AddManualOrder)
		staffGroup.GET("/outlets/get-products-in-stock/:outletId", staff.GetProducts)

		// Inventory Management
		staffGroup.GET("/outlets/get-stocks/:outletId/", staff.GetStocks)
		staffGroup.POST("/outlets/add-stock/", staff.AddStock)
		staffGroup.POST("/outlets/deduct-stock/", staff.DeductStock)
		staffGroup.POST("/outlets/get-stock-history", staff.StockHistory)

		// Notification Management
		staffGroup.GET("/outlets/get-current-order/:outletId", staff.OutletCurrentOrder)

		// Recharge Management
		staffGroup.GET("/outlets/get-recharge-history/:outletId/", staff.GetRechargeHistory)
		staffGroup.POST("/outlets/recharge-wallet/", staff.AddRecharge)

		// Order management
		staffGroup.GET("/outlets/get-order-history/", staff.GetOrderHistory)
		staffGroup.GET("/outlets/get-orderdates/:outletId/", staff.GetAvailableDatesAndSlotsForStaff)

		// Reports Management
		staffGroup.POST("/outlets/sales-trend/:outletId/", staff.GetSalesTrend)
		staffGroup.POST("/outlets/order-type-breakdown/:outletId/", staff.GetOrderTypeBreakdown)
		staffGroup.POST("/outlets/new-customers-trend/:outletId/", staff.GetNewCustomersTrend)
		staffGroup.POST("/outlets/category-breakdown/:outletId/", staff.GetCategoryBreakdown)
		staffGroup.POST("/outlets/delivery-time-orders/:outletId/", staff.GetDeliveryTimeOrders)
		staffGroup.POST("/outlets/cancellation-refunds/:outletId/", staff.GetCancellationRefunds)
		staffGroup.POST("/outlets/quantity-sold/:outletId/", staff.GetQuantitySold)

		// Profile Management
		staffGroup.GET("/profile/", staff.GetStaffProfile)
		staffGroup.PUT("/profile/", staff.UpdateStaffProfile) // TODO: Add upload middleware
		staffGroup.POST("/profile/upload-image/", staff.UploadStaffImage)
		staffGroup.DELETE("/profile/delete-image/", staff.DeleteStaffImage)

		// Security Management
		staffGroup.POST("/security/change-password/", staff.ChangePassword)
		staffGroup.GET("/security/2fa-status/", staff.Get2FAStatus)
		staffGroup.POST("/security/generate-2fa/", staff.Generate2FASetup)
		staffGroup.POST("/security/enable-2fa/", staff.Enable2FA)
		staffGroup.POST("/security/disable-2fa/", staff.Disable2FA)
		staffGroup.GET("/security/backup-codes-count/", staff.GetBackupCodesCount)
	}
}
