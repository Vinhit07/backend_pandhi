package routes

import (
	"backend_pandhi/pkg/controllers/customer"
	"backend_pandhi/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCustomerRoutes registers all customer-facing API routes
func RegisterCustomerRoutes(router *gin.RouterGroup) {
	customerGroup := router.Group("/customer")
	customerGroup.Use(middleware.AuthenticateToken(), middleware.AuthorizeRoles("CUSTOMER"))
	{
		// Products & Home
		customerGroup.GET("/outlets/get-product/", customer.GetProductsAndStocks)
		customerGroup.GET("/outlets/get-current-quota", customer.GetCurrentQuota)
		customerGroup.GET("/outlets/get-appdates/:outletId", customer.GetAvailableDatesAndSlotsForCustomer)

		// Cart management
		customerGroup.PUT("/outlets/update-cart-item", customer.UpdateCartItem)
		customerGroup.GET("/outlets/get-cart", customer.GetCart)

		// Profile management
		customerGroup.PUT("/outlets/edit-profile", customer.EditProfile) // TODO: Add upload middleware
		customerGroup.GET("/outlets/get-profile", customer.GetProfile)

		// Ticket management
		customerGroup.POST("/outlets/tickets/create", customer.CreateTicket)
		customerGroup.GET("/outlets/tickets", customer.GetCustomerTickets)
		customerGroup.GET("/outlets/tickets/:ticketId", customer.GetTicketDetails)

		// Coupon management
		customerGroup.GET("/outlets/coupons/:outletId", customer.GetCoupons)
		customerGroup.POST("/outlets/apply-coupon", customer.ApplyCoupon)

		// Feedback management
		customerGroup.POST("/feedback/submit", customer.SubmitFeedback)
		customerGroup.GET("/feedback/pending", customer.GetPendingFeedback)
		customerGroup.GET("/feedback/order/:orderId", customer.GetFeedbackStatusForOrder)
		customerGroup.GET("/feedback/product/:productId/reviews", customer.GetProductReviews)

		// Order management
		customerGroup.POST("/outlets/customer-order/", customer.CustomerAppOrder) // STUB - requires quota/inventory/payment integration
		customerGroup.GET("/outlets/customer-ongoing-order/", customer.CustomerAppOngoingOrderList)
		customerGroup.GET("/outlets/customer-order-history/", customer.CustomerAppOrderHistory)
		customerGroup.PUT("/outlets/customer-cancel-order/:orderId", customer.CustomerAppCancelOrder)
		customerGroup.POST("/outlets/create-razorpay-order", customer.CreateRazorpayOrder)     // STUB
		customerGroup.POST("/outlets/verify-razorpay-payment", customer.VerifyRazorpayPayment) // STUB

		// Wallet management
		customerGroup.POST("/outlets/create-wallet-recharge-order", customer.CreateWalletRechargeOrder)
		customerGroup.POST("/outlets/verify-wallet-recharge", customer.VerifyWalletRecharge)
		customerGroup.GET("/outlets/get-wallet-details", customer.GetWalletDetails)
		customerGroup.POST("/outlets/recharge-wallet", customer.RechargeWallet) // Legacy cash recharge
		customerGroup.GET("/outlets/get-recent-recharge", customer.RecentTrans)
		customerGroup.GET("/outlets/get-recharge-history", customer.GetRechargeHistory)
		customerGroup.GET("/outlets/service-charge-breakdown", customer.GetServiceChargeBreakdown)

		// Payment config
		customerGroup.GET("/outlets/razorpay-key", customer.GetRazorpayKey)
	}

	// Public customer routes (no auth required)
	router.GET("/customer/get-outlets/", customer.GetOutlets)
}
