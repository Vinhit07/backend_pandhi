package models

// typeOfDegree enum
type TypeOfDegree string

const (
	TypeOfDegreeUG TypeOfDegree = "UG"
	TypeOfDegreePG TypeOfDegree = "PG"
)

// OrderType enum
type OrderType string

const (
	OrderTypeManual OrderType = "MANUAL"
	OrderTypeApp    OrderType = "APP"
)

// OrderItemStatus enum
type OrderItemStatus string

const (
	OrderItemStatusNotDelivered OrderItemStatus = "NOT_DELIVERED"
	OrderItemStatusDelivered    OrderItemStatus = "DELIVERED"
)

// Role enum
type Role string

const (
	RoleCustomer   Role = "CUSTOMER"
	RoleStaff      Role = "STAFF"
	RoleSuperAdmin Role = "SUPERADMIN"
	RoleAdmin      Role = "ADMIN"
)

// AdminPermissionType enum
type AdminPermissionType string

const (
	AdminPermissionOrderManagement        AdminPermissionType = "ORDER_MANAGEMENT"
	AdminPermissionStaffManagement        AdminPermissionType = "STAFF_MANAGEMENT"
	AdminPermissionInventoryManagement    AdminPermissionType = "INVENTORY_MANAGEMENT"
	AdminPermissionExpenditureManagement  AdminPermissionType = "EXPENDITURE_MANAGEMENT"
	AdminPermissionWalletManagement       AdminPermissionType = "WALLET_MANAGEMENT"
	AdminPermissionCustomerManagement     AdminPermissionType = "CUSTOMER_MANAGEMENT"
	AdminPermissionTicketManagement       AdminPermissionType = "TICKET_MANAGEMENT"
	AdminPermissionNotificationsManagement AdminPermissionType = "NOTIFICATIONS_MANAGEMENT"
	AdminPermissionProductManagement      AdminPermissionType = "PRODUCT_MANAGEMENT"
	AdminPermissionAppManagement          AdminPermissionType = "APP_MANAGEMENT"
	AdminPermissionReportsAnalytics       AdminPermissionType = "REPORTS_ANALYTICS"
	AdminPermissionSettings               AdminPermissionType = "SETTINGS"
	AdminPermissionOnboarding             AdminPermissionType = "ONBOARDING"
	AdminPermissionAdminManagement        AdminPermissionType = "ADMIN_MANAGEMENT"
)

// PermissionType enum
type PermissionType string

const (
	PermissionTypeBilling         PermissionType = "BILLING"
	PermissionTypeProductInsights PermissionType = "PRODUCT_INSIGHTS"
	PermissionTypeReports         PermissionType = "REPORTS"
	PermissionTypeInventory       PermissionType = "INVENTORY"
)

// PaymentMethod enum
type PaymentMethod string

const (
	PaymentMethodUPI    PaymentMethod = "UPI"
	PaymentMethodCard   PaymentMethod = "CARD"
	PaymentMethodCash   PaymentMethod = "CASH"
	PaymentMethodWallet PaymentMethod = "WALLET"
)

// OrderStatus enum
type OrderStatus string

const (
	OrderStatusPending            OrderStatus = "PENDING"
	OrderStatusDelivered          OrderStatus = "DELIVERED"
	OrderStatusPartiallyDelivered OrderStatus = "PARTIALLY_DELIVERED"
	OrderStatusCancelled          OrderStatus = "CANCELLED"
	OrderStatusPartialCancel      OrderStatus = "PARTIAL_CANCEL"
)

// Priority enum
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
)

// Category enum
type Category string

const (
	CategoryMeals        Category = "Meals"
	CategoryStarters     Category = "Starters"
	CategoryDesserts     Category = "Desserts"
	CategoryBeverages    Category = "Beverages"
	CategorySpecialFoods Category = "SpecialFoods"
)

// TicketStatus enum
type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "OPEN"
	TicketStatusInProgress TicketStatus = "IN_PROGRESS"
	TicketStatusClosed     TicketStatus = "CLOSED"
)

// StockAction enum
type StockAction string

const (
	StockActionAdd    StockAction = "ADD"
	StockActionRemove StockAction = "REMOVE"
	StockActionUpdate StockAction = "UPDATE"
)

// WalletTransType enum
type WalletTransType string

const (
	WalletTransTypeRecharge WalletTransType = "RECHARGE"
	WalletTransTypeDeduct   WalletTransType = "DEDUCT"
	TransactionTypeCredit   WalletTransType = "CREDIT" // Added for refunds
	TransactionTypeDebit    WalletTransType = "DEBIT"
)

// DeliverySlot enum
type DeliverySlot string

const (
	DeliverySlot1112 DeliverySlot = "SLOT_11_12"
	DeliverySlot1213 DeliverySlot = "SLOT_12_13"
	DeliverySlot1314 DeliverySlot = "SLOT_13_14"
	DeliverySlot1415 DeliverySlot = "SLOT_14_15"
	DeliverySlot1516 DeliverySlot = "SLOT_15_16"
	DeliverySlot1617 DeliverySlot = "SLOT_16_17"
)

// OutletAppFeature enum
type OutletAppFeature string

const (
	OutletAppFeatureApp         OutletAppFeature = "APP"
	OutletAppFeatureUPI         OutletAppFeature = "UPI"
	OutletAppFeatureLiveCounter OutletAppFeature = "LIVE_COUNTER"
	OutletAppFeatureCoupons     OutletAppFeature = "COUPONS"
)

// NotificationStatus enum
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "PENDING"
	NotificationStatusSent      NotificationStatus = "SENT"
	NotificationStatusFailed    NotificationStatus = "FAILED"
	NotificationStatusDelivered NotificationStatus = "DELIVERED"
)
