package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONArray type for JSONB fields in PostgreSQL
type JSONArray []interface{}

// Scan implements the sql.Scanner interface
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Order model - mirrors Prisma Order model
type Order struct {
	ID                int            `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CustomerID        *int           `gorm:"column:customerId" json:"customerId"`
	OutletID          int            `gorm:"not null;column:outletId" json:"outletId"`
	TotalAmount       float64        `gorm:"not null;column:totalAmount" json:"totalAmount"`
	PaymentMethod     PaymentMethod  `gorm:"type:text;not null;column:paymentMethod" json:"paymentMethod"`
	Status            OrderStatus    `gorm:"type:text;not null;column:status" json:"status"`
	CreatedAt         time.Time      `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	Type              OrderType      `gorm:"type:text;not null;column:type" json:"type"`
	DeliveryDate      *time.Time     `gorm:"column:deliveryDate" json:"deliveryDate"`
	DeliverySlot      *DeliverySlot  `gorm:"type:text;column:deliverySlot" json:"deliverySlot"`
	IsPreOrder        bool           `gorm:"default:false;column:isPreOrder" json:"isPreOrder"`
	RazorpayPaymentID *string        `gorm:"column:razorpayPaymentId" json:"razorpayPaymentId"`
	DeliveredAt       *time.Time     `gorm:"column:deliveredAt" json:"deliveredAt"`

	// Relationships
	Customer  *CustomerDetails `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`
	Outlet    Outlet           `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
	Items     []OrderItem      `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Feedbacks []Feedback       `gorm:"foreignKey:OrderID" json:"feedbacks,omitempty"`
}

// TableName specifies the table name for Order model
func (Order) TableName() string {
	return "Order"
}

// OrderItem model - mirrors Prisma OrderItem model
type OrderItem struct {
	ID           int             `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OrderID      int             `gorm:"not null;column:orderId" json:"orderId"`
	ProductID    int             `gorm:"not null;column:productId" json:"productId"`
	Quantity     int             `gorm:"not null;column:quantity" json:"quantity"`
	UnitPrice    float64         `gorm:"not null;column:unitPrice" json:"unitPrice"`
	Status       OrderItemStatus `gorm:"type:text;default:'NOT_DELIVERED';column:status" json:"status"`
	FreeQuantity int             `gorm:"default:0;column:freeQuantity" json:"freeQuantity"`

	// Relationships
	Order   Order   `gorm:"foreignKey:OrderID;references:ID" json:"order,omitempty"`
	Product Product `gorm:"foreignKey:ProductID;references:ID" json:"product,omitempty"`
}

// TableName specifies the table name for OrderItem model
func (OrderItem) TableName() string {
	return "OrderItem"
}

// Wallet model - mirrors Prisma Wallet model
type Wallet struct {
	ID             int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CustomerID     int       `gorm:"unique;not null;column:customerId" json:"customerId"`
	Balance        float64   `gorm:"default:0;column:balance" json:"balance"`
	TotalRecharged float64   `gorm:"default:0;column:totalRecharged" json:"totalRecharged"`
	TotalUsed      float64   `gorm:"default:0;column:totalUsed" json:"totalUsed"`
	LastRecharged  *time.Time `gorm:"column:lastRecharged" json:"lastRecharged"`
	LastOrder      *time.Time `gorm:"column:lastOrder" json:"lastOrder"`

	// Relationships
	Customer     CustomerDetails     `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`
	Transactions []WalletTransaction `gorm:"foreignKey:WalletID" json:"transactions,omitempty"`
}

// TableName specifies the table name for Wallet model
func (Wallet) TableName() string {
	return "Wallet"
}

// WalletTransaction model - mirrors Prisma WalletTransaction model
type WalletTransaction struct {
	ID                int             `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	WalletID          int             `gorm:"not null;column:walletId" json:"walletId"`
	Amount            float64         `gorm:"not null;column:amount" json:"amount"`
	Method            PaymentMethod   `gorm:"type:text;not null;column:method" json:"method"`
	CreatedAt         time.Time       `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	Status            WalletTransType `gorm:"type:text;not null;column:status" json:"status"`
	GrossAmount       *float64        `gorm:"column:grossAmount" json:"grossAmount"`
	RazorpayOrderID   *string         `gorm:"column:razorpayOrderId" json:"razorpayOrderId"`
	RazorpayPaymentID *string         `gorm:"column:razorpayPaymentId" json:"razorpayPaymentId"`
	ServiceCharge     *float64        `gorm:"column:serviceCharge" json:"serviceCharge"`
	Description       string          `gorm:"column:description" json:"description"`

	// Relationships
	Wallet Wallet `gorm:"foreignKey:WalletID;references:ID" json:"wallet,omitempty"`
}

// TableName specifies the table name for WalletTransaction model
func (WalletTransaction) TableName() string {
	return "WalletTransaction"
}

// Expense model - mirrors Prisma Expense model
type Expense struct {
	ID          int           `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OutletID    int           `gorm:"not null;column:outletId" json:"outletId"`
	Description string        `gorm:"not null;column:description" json:"description"`
	Category    string        `gorm:"not null;column:category" json:"category"`
	Amount      float64       `gorm:"not null;column:amount" json:"amount"`
	Method      PaymentMethod `gorm:"type:text;not null;column:method" json:"method"`
	PaidTo      string        `gorm:"not null;column:paidTo" json:"paidTo"`
	ExpenseDate time.Time     `gorm:"not null;column:expenseDate" json:"expenseDate"`
	CreatedAt   time.Time     `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`

	// Relationships
	Outlet Outlet `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
}

// TableName specifies the table name for Expense model
func (Expense) TableName() string {
	return "Expense"
}

// Ticket model - mirrors Prisma Ticket model
type Ticket struct {
	ID             int          `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CustomerID     int          `gorm:"not null;column:customerId" json:"customerId"`
	Title          string       `gorm:"not null;column:title" json:"title"`
	Description    string       `gorm:"not null;column:description" json:"description"`
	Priority       Priority     `gorm:"type:text;not null;column:priority" json:"priority"`
	Status         TicketStatus `gorm:"type:text;default:'OPEN';column:status" json:"status"`
	CreatedAt      time.Time    `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	ResolvedAt     *time.Time   `gorm:"column:resolvedAt" json:"resolvedAt"`
	ResolutionNote *string      `gorm:"column:resolutionNote" json:"resolutionNote"`
	ImageURL       *string      `gorm:"column:imageUrl" json:"imageUrl"`

	// Relationships
	Customer CustomerDetails `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`
}

// TableName specifies the table name for Ticket model
func (Ticket) TableName() string {
	return "Ticket"
}

// Coupon model - mirrors Prisma Coupon model
type Coupon struct {
	ID            int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Code          string    `gorm:"unique;not null;column:code" json:"code"`
	Description   string    `gorm:"not null;column:description" json:"description"`
	RewardValue   float64   `gorm:"not null;column:rewardValue" json:"rewardValue"`
	MinOrderValue float64   `gorm:"not null;column:minOrderValue" json:"minOrderValue"`
	ValidFrom     time.Time `gorm:"not null;column:validFrom" json:"validFrom"`
	ValidUntil    time.Time `gorm:"not null;column:validUntil" json:"validUntil"`
	IsActive      bool      `gorm:"default:true;column:isActive" json:"isActive"`
	UsageLimit    int       `gorm:"not null;column:usageLimit" json:"usageLimit"`
	UsedCount     int       `gorm:"default:0;column:usedCount" json:"usedCount"`
	CreatedAt     time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	OutletID      *int      `gorm:"column:outletId" json:"outletId"`
	UsageType     *string   `gorm:"column:usageType" json:"usageType"`

	// Relationships
	Outlet *Outlet       `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
	Usages []CouponUsage `gorm:"foreignKey:CouponID" json:"usages,omitempty"`
}

// TableName specifies the table name for Coupon model
func (Coupon) TableName() string {
	return "Coupon"
}

// CouponUsage model - mirrors Prisma CouponUsage model
type CouponUsage struct {
	ID       int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CouponID int       `gorm:"not null;column:couponId" json:"couponId"`
	OrderID  int       `gorm:"not null;column:orderId" json:"orderId"`
	UserID   int       `gorm:"not null;column:userId" json:"userId"`
	Amount   float64   `gorm:"not null;column:amount" json:"amount"`
	UsedAt   time.Time `gorm:"autoCreateTime;column:usedAt" json:"usedAt"`

	// Relationships
	Coupon Coupon `gorm:"foreignKey:CouponID;references:ID" json:"coupon,omitempty"`
}

// TableName specifies the table name for CouponUsage model
func (CouponUsage) TableName() string {
	return "CouponUsage"
}

// Admin model - mirrors Prisma Admin model
type Admin struct {
	ID         int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Email      string    `gorm:"unique;not null;column:email" json:"email"`
	Name       string    `gorm:"not null;column:name" json:"name"`
	Password   string    `gorm:"not null;column:password" json:"-"` // Don't expose password
	IsVerified bool      `gorm:"default:false;column:isVerified" json:"isVerified"`
	CreatedAt  time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`
	Phone      *string   `gorm:"column:phone" json:"phone"`
	ImageURL   *string   `gorm:"column:imageUrl" json:"imageUrl"`
	AadharURL  *string   `gorm:"column:aadharUrl" json:"aadharUrl"`
	PanURL     *string   `gorm:"column:panUrl" json:"panUrl"`

	// Relationships
	Outlets     []AdminOutlet     `gorm:"foreignKey:AdminID" json:"outlets,omitempty"`
	Permissions []AdminPermission `gorm:"foreignKey:AdminID" json:"permissions,omitempty"`
}

// TableName specifies the table name for Admin model
func (Admin) TableName() string {
	return "Admin"
}

// AdminOutlet model - mirrors Prisma AdminOutlet model (Junction table)
type AdminOutlet struct {
	ID       int `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	AdminID  int `gorm:"not null;column:adminId" json:"adminId"`
	OutletID int `gorm:"not null;column:outletId" json:"outletId"`

	// Relationships
	Admin       Admin             `gorm:"foreignKey:AdminID;references:ID" json:"admin,omitempty"`
	Outlet      Outlet            `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
	Permissions []AdminPermission `gorm:"foreignKey:AdminOutletID" json:"permissions,omitempty"`
}

// TableName specifies the table name for AdminOutlet model
func (AdminOutlet) TableName() string {
	return "AdminOutlet"
}

// AdminPermission model - mirrors Prisma AdminPermission model
type AdminPermission struct {
	ID            int                 `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	AdminOutletID int                 `gorm:"not null;column:adminOutletId" json:"adminOutletId"`
	AdminID       *int                `gorm:"column:adminId" json:"adminId"`
	Type          AdminPermissionType `gorm:"type:text;not null;column:type" json:"type"`
	IsGranted     bool                `gorm:"default:false;column:isGranted" json:"isGranted"`

	// Relationships
	Admin       *Admin       `gorm:"foreignKey:AdminID;references:ID" json:"admin,omitempty"`
	AdminOutlet AdminOutlet  `gorm:"foreignKey:AdminOutletID;references:ID" json:"adminOutlet,omitempty"`
}

// TableName specifies the table name for AdminPermission model
func (AdminPermission) TableName() string {
	return "AdminPermission"
}

// OutletAvailability model - mirrors Prisma OutletAvailability model
type OutletAvailability struct {
	ID                int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OutletID          int       `gorm:"not null;column:outletId" json:"outletId"`
	Date              time.Time `gorm:"not null;column:date" json:"date"`
	NonAvailableSlots JSONArray `gorm:"type:jsonb;column:nonAvailableSlots" json:"nonAvailableSlots"`

	// Relationships
	Outlet Outlet `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
}

// TableName specifies the table name for OutletAvailability model
func (OutletAvailability) TableName() string {
	return "OutletAvailability"
}

// Notification model - mirrors Prisma Notification model
type Notification struct {
	ID        int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Message   string    `gorm:"not null;column:message" json:"message"`
	ProductID int       `gorm:"not null;column:productId" json:"productId"`
	OutletID  int       `gorm:"not null;column:outletId" json:"outletId"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	IsRead    bool      `gorm:"default:false;column:isRead" json:"isRead"`

	// Relationships
	Outlet  Outlet  `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
	Product Product `gorm:"foreignKey:ProductID;references:ID" json:"product,omitempty"`
}

// TableName specifies the table name for Notification model
func (Notification) TableName() string {
	return "Notification"
}

// ScheduledNotification model - mirrors Prisma ScheduledNotification model
type ScheduledNotification struct {
	ID          int        `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Title       string     `gorm:"not null;column:title" json:"title"`
	Message     string     `gorm:"not null;column:message" json:"message"`
	Priority    Priority   `gorm:"type:text;not null;column:priority" json:"priority"`
	ImageURL    *string    `gorm:"column:imageUrl" json:"imageUrl"`
	ScheduledAt time.Time  `gorm:"not null;column:scheduledAt" json:"scheduledAt"`
	SentAt      *time.Time `gorm:"column:sentAt" json:"sentAt"`
	IsSent      bool       `gorm:"default:false;column:isSent" json:"isSent"`
	OutletID    int        `gorm:"not null;column:outletId" json:"outletId"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`

	// Relationships
	Deliveries []NotificationDelivery `gorm:"foreignKey:ScheduledNotificationID" json:"deliveries,omitempty"`
	Outlet     Outlet                 `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
}

// TableName specifies the table name for ScheduledNotification model
func (ScheduledNotification) TableName() string {
	return "ScheduledNotification"
}

// NotificationDelivery model - mirrors Prisma NotificationDelivery model
type NotificationDelivery struct {
	ID                      int                `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ScheduledNotificationID int                `gorm:"not null;column:scheduledNotificationId" json:"scheduledNotificationId"`
	UserID                  int                `gorm:"not null;column:userId" json:"userId"`
	DeviceToken             string             `gorm:"not null;column:deviceToken" json:"deviceToken"`
	Status                  NotificationStatus `gorm:"type:text;default:'PENDING';column:status" json:"status"`
	SentAt                  *time.Time         `gorm:"column:sentAt" json:"sentAt"`
	FailureReason           *string            `gorm:"column:failureReason" json:"failureReason"`
	MessageID               *string            `gorm:"column:messageId" json:"messageId"`
	CreatedAt               time.Time          `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	UpdatedAt               time.Time          `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`

	// Relationships
	ScheduledNotification ScheduledNotification `gorm:"foreignKey:ScheduledNotificationID;references:ID" json:"scheduledNotification,omitempty"`
	User                  User                  `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TableName specifies the table name for NotificationDelivery model
func (NotificationDelivery) TableName() string {
	return "NotificationDelivery"
}

// UserDeviceToken model - mirrors Prisma UserDeviceToken model
type UserDeviceToken struct {
	ID          int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID      int       `gorm:"not null;column:userId" json:"userId"`
	DeviceToken string    `gorm:"unique;not null;column:deviceToken" json:"deviceToken"`
	Platform    string    `gorm:"not null;column:platform" json:"platform"`
	IsActive    bool      `gorm:"default:true;column:isActive" json:"isActive"`
	CreatedAt   time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`

	// Relationships
	User User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TableName specifies the table name for UserDeviceToken model
func (UserDeviceToken) TableName() string {
	return "UserDeviceToken"
}

// OutletAppManagement model - mirrors Prisma OutletAppManagement model
type OutletAppManagement struct {
	ID        int              `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	OutletID  int              `gorm:"not null;column:outletId" json:"outletId"`
	Feature   OutletAppFeature `gorm:"type:text;not null;column:feature" json:"feature"`
	IsEnabled bool             `gorm:"default:false;column:isEnabled" json:"isEnabled"`

	// Relationships
	Outlet Outlet `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
}

// TableName specifies the table name for OutletAppManagement model
func (OutletAppManagement) TableName() string {
	return "OutletAppManagement"
}

// Feedback model - mirrors Prisma Feedback model
type Feedback struct {
	ID             int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID         int       `gorm:"not null;column:userId" json:"userId"`
	ProductID      int       `gorm:"not null;column:productId" json:"productId"`
	OrderID        int       `gorm:"not null;column:orderId" json:"orderId"`
	RatingOverall  float64   `gorm:"not null;column:ratingOverall" json:"ratingOverall"`
	RatingTaste    float64   `gorm:"not null;column:ratingTaste" json:"ratingTaste"`
	RatingQuality  float64   `gorm:"not null;column:ratingQuality" json:"ratingQuality"`
	RatingQuantity float64   `gorm:"not null;column:ratingQuantity" json:"ratingQuantity"`
	Comment        *string   `gorm:"column:comment" json:"comment"`
	CreatedAt      time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`

	// Relationships
	User    User    `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Product Product `gorm:"foreignKey:ProductID;references:ID" json:"product,omitempty"`
	Order   Order   `gorm:"foreignKey:OrderID;references:ID" json:"order,omitempty"`
}

// TableName specifies the table name for Feedback model
func (Feedback) TableName() string {
	return "Feedback"
}

// UserFreeQuota model - mirrors Prisma UserFreeQuota model
type UserFreeQuota struct {
	ID              int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID          int       `gorm:"not null;column:userId" json:"userId"`
	ConsumptionDate time.Time `gorm:"type:date;not null;column:consumptionDate" json:"consumptionDate"`
	QuantityUsed    int       `gorm:"default:0;column:quantityUsed" json:"quantityUsed"`

	// Relationships
	User User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TableName specifies the table name for UserFreeQuota model
func (UserFreeQuota) TableName() string {
	return "UserFreeQuota"
}
