package routes

import (
	"backend_pandhi/pkg/controllers/superadmin"
	"backend_pandhi/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterSuperAdminRoutes(router *gin.Engine) {
	superadminGroup := router.Group("/api/superadmin")

	// Outlet Management (3 endpoints)
	superadminGroup.POST("/add-outlet/", middleware.RestrictToSuperAdmin(), superadmin.AddOutlets)
	superadminGroup.GET("/get-outlets/", middleware.RestrictToSuperAdminOrAdminOrCustomer(), superadmin.GetOutlets)
	superadminGroup.DELETE("/remove-outlet/:outletId", middleware.RestrictToSuperAdmin(), superadmin.RemoveOutlets)

	// Staff Management (6 endpoints)
	superadminGroup.POST("/outlets/add-staff/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.OutletAddStaff)
	superadminGroup.POST("/outlets/permissions/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.OutletStaffPermission)
	superadminGroup.GET("/outlets/get-staffs/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetOutletStaff)
	superadminGroup.PUT("/outlets/update-staff/:staffId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.OutletUpdateStaff)
	superadminGroup.DELETE("/outlets/delete-staff/:staffId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.OutletDeleteStaff)
	superadminGroup.GET("/outlets/staff/:staffId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetStaffById)

	// Product Management (4 endpoints)
	superadminGroup.GET("/outlets/get-products/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetProducts)
	superadminGroup.POST("/outlets/add-product/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.AddProduct)
	superadminGroup.DELETE("/outlets/delete-product/:id", middleware.RestrictToSuperAdminOrAdmin(), superadmin.DeleteProduct)
	superadminGroup.PUT("/outlets/update-product/:id", middleware.RestrictToSuperAdminOrAdmin(), superadmin.UpdateProduct)

	// Order Management (1 endpoint)
	superadminGroup.GET("/outlets/:outletId/orders/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.OutletTotalOrders)

	// Inventory Management (4 endpoints)
	superadminGroup.GET("/outlets/get-stocks/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetStocks)
	superadminGroup.POST("/outlets/add-stocks/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.AddStock)
	superadminGroup.POST("/outlets/deduct-stocks/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.DeductStock)
	superadminGroup.POST("/outlets/get-stock-history", middleware.RestrictToSuperAdminOrAdmin(), superadmin.StockHistory)

	// Expense Management (3 endpoints)
	superadminGroup.POST("/outlets/add-expenses/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.AddExpense)
	superadminGroup.GET("/outlets/get-expenses/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetExpenses)
	superadminGroup.POST("/outlets/get-expenses-bydate/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetExpenseByDate)

	// Wallet Management (3 endpoints)
	superadminGroup.GET("/outlets/wallet-history/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetCustomersWithWallet)
	superadminGroup.GET("/outlets/recharge-history/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetRechargeHistoryByOutlet)
	superadminGroup.GET("/outlets/paid-wallet/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetOrdersPaidViaWallet)

	// Customer Management (1 endpoint)
	superadminGroup.GET("/outlets/customers/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetOutletCustomers)

	// Ticket Management (2 endpoints)
	superadminGroup.GET("/outlets/tickets/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetTickets)
	superadminGroup.POST("/outlets/ticket-close/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.TicketClose)

	// Coupon Management (3 endpoints)
	superadminGroup.POST("/create-coupon/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.CreateCoupon)
	superadminGroup.GET("/get-coupons/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetCoupons)
	superadminGroup.DELETE("/delete-coupon/:couponId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.DeleteCoupon)

	// Notification Management (8 endpoints)
	superadminGroup.GET("/dashboard/low-stock-notifications", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetLowStockNotifications)
	superadminGroup.POST("/notifications/schedule", middleware.RestrictToSuperAdminOrAdmin(), superadmin.CreateScheduledNotification)
	superadminGroup.GET("/notifications/scheduled/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetScheduledNotifications)
	superadminGroup.DELETE("/notifications/scheduled/:notificationId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.CancelScheduledNotification)
	superadminGroup.POST("/notifications/send-immediate", middleware.RestrictToSuperAdminOrAdmin(), superadmin.SendImmediateNotification)
	superadminGroup.GET("/notifications/stats/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetNotificationStats)
	superadminGroup.GET("/notifications/fcm-status", middleware.RestrictToSuperAdminOrAdmin(), superadmin.TestFCMService)
	superadminGroup.POST("/notifications/test-single", middleware.RestrictToSuperAdminOrAdmin(), superadmin.TestSingleDeviceNotification)

	// App Management (5 endpoints)
	superadminGroup.GET("/outlets/get-non-availability-preview/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetOutletNonAvailabilityPreview)
	superadminGroup.POST("/outlets/set-availability/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.SetOutletAvailability)
	superadminGroup.GET("/outlets/get-available-dates/:outletId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetAvailableDatesAndSlots)
	superadminGroup.GET("/outlets/app-features/:outletId", middleware.RestrictToSuperAdminOrAdminOrCustomer(), superadmin.GetOutletAppFeatures)
	superadminGroup.POST("/outlets/app-features/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.UpdateOutletAppFeatures)

	// Reports Management (7 endpoints)
	superadminGroup.POST("/outlets/sales-report/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetOutletSalesReport)
	superadminGroup.POST("/outlets/revenue-report/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetOutletRevenueByItems)
	superadminGroup.POST("/outlets/revenue-split/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetRevenueSplit)
	superadminGroup.POST("/outlets/wallet-recharge-by-day/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetWalletRechargeByDay)
	superadminGroup.POST("/outlets/profit-loss-trends/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetProfitLossTrends)
	superadminGroup.POST("/outlets/customer-overview/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetCustomerOverview)
	superadminGroup.POST("/outlets/customer-per-order/:outletId/", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetCustomerPerOrder)

	// Dashboard Management (6 analytics endpoints)
	superadminGroup.POST("/dashboard/overview", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetDashboardOverview)
	superadminGroup.POST("/dashboard/revenue-trend", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetRevenueTrend)
	superadminGroup.POST("/dashboard/order-status-distribution", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetOrderStatusDistribution)
	superadminGroup.POST("/dashboard/order-source-distribution", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetOrderSourceDistribution)
	superadminGroup.POST("/dashboard/top-selling-items", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetTopSellingItems)
	superadminGroup.POST("/dashboard/peak-time-slots", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetPeakTimeSlots)

	// Admin Management (7 endpoints)
	superadminGroup.GET("/pending-admins", middleware.RestrictToSuperAdmin(), superadmin.GetPendingAdminVerifications)
	superadminGroup.POST("/verify-admin/:adminId", middleware.RestrictToSuperAdmin(), superadmin.VerifyAdmin)
	superadminGroup.GET("/verified-admins", middleware.RestrictToSuperAdmin(), superadmin.GetVerifiedAdmins)
	superadminGroup.GET("/admin/:adminId", middleware.RestrictToSuperAdminOrAdmin(), superadmin.GetAdminDetails)
	superadminGroup.DELETE("/admin/:adminId", middleware.RestrictToSuperAdmin(), superadmin.DeleteAdmin)
	superadminGroup.POST("/map-outlets-to-admin", middleware.RestrictToSuperAdmin(), superadmin.MapOutletsToAdmin)
	superadminGroup.POST("/assign-admin-permissions", middleware.RestrictToSuperAdmin(), superadmin.AssignAdminPermissions)

	// Staff Verification (3 endpoints)
	superadminGroup.POST("/verify-staff/:userId", middleware.RestrictToSuperAdmin(), superadmin.VerifyStaff)
	superadminGroup.GET("/unverified-staff", middleware.RestrictToSuperAdmin(), superadmin.GetUnverifiedStaff)
	superadminGroup.GET("/verified-staff", middleware.RestrictToSuperAdmin(), superadmin.GetVerifiedStaff)
}
