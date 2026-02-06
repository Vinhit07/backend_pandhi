package models

import (
	"time"
)

// Outlet model - mirrors Prisma Out let model
type Outlet struct {
	ID         int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name       string    `gorm:"unique;not null;column:name" json:"name"`
	Address    *string   `gorm:"column:address" json:"address"`
	Email      *string   `gorm:"unique;column:email" json:"email"`
	IsActive   bool      `gorm:"default:true;column:isActive" json:"isActive"`
	CreatedAt  time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`
	StaffCount int       `gorm:"default:0;column:staffCount" json:"staffCount"`
	Phone      *string   `gorm:"column:phone" json:"phone"`

	// Relationships
	Admins                 []AdminOutlet           `gorm:"foreignKey:OutletID" json:"admins,omitempty"`
	Coupons                []Coupon                `gorm:"foreignKey:OutletID" json:"coupons,omitempty"`
	Expenses               []Expense               `gorm:"foreignKey:OutletID" json:"expenses,omitempty"`
	Inventories            []Inventory             `gorm:"foreignKey:OutletID" json:"inventories,omitempty"`
	Notifications          []Notification          `gorm:"foreignKey:OutletID" json:"notifications,omitempty"`
	Orders                 []Order                 `gorm:"foreignKey:OutletID" json:"orders,omitempty"`
	AppManagement          []OutletAppManagement   `gorm:"foreignKey:OutletID" json:"appManagement,omitempty"`
	Availability           []OutletAvailability    `gorm:"foreignKey:OutletID" json:"availability,omitempty"`
	Products               []Product               `gorm:"foreignKey:OutletID" json:"products,omitempty"`
	ScheduledNotifications []ScheduledNotification `gorm:"foreignKey:OutletID" json:"scheduledNotifications,omitempty"`
	StockHistory           []StockHistory          `gorm:"foreignKey:OutletID" json:"stockHistory,omitempty"`
	Users                  []User                  `gorm:"foreignKey:OutletID" json:"users,omitempty"`
}

// TableName specifies the table name for Outlet model
func (Outlet) TableName() string {
	return "Outlet"
}

// User model - mirrors Prisma User model
type User struct {
	ID         int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Email      string    `gorm:"unique;not null;column:email" json:"email"`
	Name       string    `gorm:"not null;column:name" json:"name"`
	Password   *string   `gorm:"column:password" json:"-"` // Don't expose password in JSON
	Role       Role      `gorm:"type:text;default:'CUSTOMER';column:role" json:"role"`
	OutletID   *int      `gorm:"column:outletId" json:"outletId"`
	CreatedAt  time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	Phone      *string   `gorm:"column:phone" json:"phone"`
	GoogleID   *string   `gorm:"unique;column:googleId" json:"googleId"`
	IsVerified bool      `gorm:"default:false;column:isVerified" json:"isVerified"`
	ImageURL   *string   `gorm:"column:imageUrl" json:"imageUrl"`

	// Relationships
	CustomerInfo           *CustomerDetails       `gorm:"foreignKey:UserID" json:"customerInfo,omitempty"`
	StaffInfo              *StaffDetails          `gorm:"foreignKey:UserID" json:"staffInfo,omitempty"`
	Outlet                 *Outlet                `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
	NotificationDeliveries []NotificationDelivery `gorm:"foreignKey:UserID" json:"notificationDeliveries,omitempty"`
	Feedbacks              []Feedback             `gorm:"foreignKey:UserID" json:"feedbacks,omitempty"`
	DeviceTokens           []UserDeviceToken      `gorm:"foreignKey:UserID" json:"deviceTokens,omitempty"`
	FreeQuota              []UserFreeQuota        `gorm:"foreignKey:UserID" json:"freeQuota,omitempty"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "User"
}

// CustomerDetails model - mirrors Prisma CustomerDetails model
type CustomerDetails struct {
	ID          int           `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID      int           `gorm:"unique;not null;column:userId" json:"userId"`
	YearOfStudy *int          `gorm:"column:yearOfStudy" json:"yearOfStudy"`
	Bio         *string       `gorm:"column:bio" json:"bio"`
	Degree      *TypeOfDegree `gorm:"type:text;column:degree" json:"degree"`
	OrderCount  int           `gorm:"default:0;column:orderCount" json:"orderCount"`

	// Relationships
	User    User     `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Cart    *Cart    `gorm:"foreignKey:CustomerID" json:"cart,omitempty"`
	Orders  []Order  `gorm:"foreignKey:CustomerID" json:"orders,omitempty"`
	Tickets []Ticket `gorm:"foreignKey:CustomerID" json:"tickets,omitempty"`
	Wallet  *Wallet  `gorm:"foreignKey:CustomerID" json:"wallet,omitempty"`
}

// TableName specifies the table name for CustomerDetails model
func (CustomerDetails) TableName() string {
	return "CustomerDetails"
}

// StaffDetails model - mirrors Prisma StaffDetails model
type StaffDetails struct {
	ID                   int        `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID               int        `gorm:"unique;not null;column:userId" json:"userId"`
	StaffRole            string     `gorm:"default:'Staff';column:staffRole" json:"staffRole"`
	TwoFactorBackupCodes *string    `gorm:"type:jsonb;column:twoFactorBackupCodes" json:"twoFactorBackupCodes,omitempty"`
	TwoFactorEnabled     bool       `gorm:"default:false;column:twoFactorEnabled" json:"twoFactorEnabled"`
	TwoFactorEnabledAt   *time.Time `gorm:"column:twoFactorEnabledAt" json:"twoFactorEnabledAt"`
	TwoFactorSecret      *string    `gorm:"column:twoFactorSecret" json:"-"` // Don't expose secret
	AadharURL            *string    `gorm:"column:aadharUrl" json:"aadharUrl"`
	PanURL               *string    `gorm:"column:panUrl" json:"panUrl"`

	// Relationships
	User        User              `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	Permissions []StaffPermission `gorm:"foreignKey:StaffID" json:"permissions,omitempty"`
}

// TableName specifies the table name for StaffDetails model
func (StaffDetails) TableName() string {
	return "StaffDetails"
}

// StaffPermission model - mirrors Prisma StaffPermission model
type StaffPermission struct {
	ID        int            `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	StaffID   int            `gorm:"not null;column:staffId" json:"staffId"`
	Type      PermissionType `gorm:"type:text;not null;column:type" json:"type"`
	IsGranted bool           `gorm:"default:false;column:isGranted" json:"isGranted"`

	// Relationships
	Staff StaffDetails `gorm:"foreignKey:StaffID;references:ID" json:"staff,omitempty"`
}

// TableName specifies the table name for StaffPermission model
func (StaffPermission) TableName() string {
	return "StaffPermission"
}

// Cart model - mirrors Prisma Cart model
type Cart struct {
	ID         int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CreatedAt  time.Time `gorm:"autoCreateTime;column:createdAt" json:"createdAt"`
	CustomerID int       `gorm:"unique;not null;column:customerId" json:"customerId"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime;column:updatedAt" json:"updatedAt"`

	// Relationships
	Customer CustomerDetails `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`
	Items    []CartItem      `gorm:"foreignKey:CartID" json:"items,omitempty"`
}

// TableName specifies the table name for Cart model
func (Cart) TableName() string {
	return "Cart"
}

// CartItem model - mirrors Prisma CartItem model
type CartItem struct {
	ID        int `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CartID    int `gorm:"not null;column:cartId" json:"cartId"`
	ProductID int `gorm:"not null;column:productId" json:"productId"`
	Quantity  int `gorm:"not null;column:quantity" json:"quantity"`

	// Relationships
	Cart    Cart    `gorm:"foreignKey:CartID;references:ID" json:"cart,omitempty"`
	Product Product `gorm:"foreignKey:ProductID;references:ID" json:"product,omitempty"`
}

// TableName specifies the table name for CartItem model
func (CartItem) TableName() string {
	return "CartItem"
}

// Product model - mirrors Prisma Product model
type Product struct {
	ID                    int      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name                  string   `gorm:"unique;not null;column:name" json:"name"`
	Description           *string  `gorm:"column:description" json:"description"`
	Price                 float64  `gorm:"not null;column:price" json:"price"`
	ImageURL              *string  `gorm:"column:imageUrl" json:"imageUrl"`
	OutletID              int      `gorm:"not null;column:outletId" json:"outletId"`
	Category              Category `gorm:"type:text;not null;column:category" json:"category"`
	MinValue              *int     `gorm:"default:0;column:minValue" json:"minValue"`
	IsVeg                 bool     `gorm:"default:true;column:isVeg" json:"isVeg"`
	RatingSum30d          float64  `gorm:"default:0;column:ratingSum30d" json:"ratingSum30d"`
	RatingCount30d        int      `gorm:"default:0;column:ratingCount30d" json:"ratingCount30d"`
	TrendScore            float64  `gorm:"default:0;column:trendScore" json:"trendScore"`
	RatingSumLifetime     float64  `gorm:"default:0;column:ratingSumLifetime" json:"ratingSumLifetime"`
	RatingCountLifetime   int      `gorm:"default:0;column:ratingCountLifetime" json:"ratingCountLifetime"`
	AverageRatingLifetime float64  `gorm:"default:0;column:averageRatingLifetime" json:"averageRatingLifetime"`
	CompanyPaid           bool     `gorm:"default:false;column:companyPaid" json:"companyPaid"`

	// Relationships
	Outlet        Outlet         `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
	CartItems     []CartItem     `gorm:"foreignKey:ProductID" json:"cartItems,omitempty"`
	Inventory     *Inventory     `gorm:"foreignKey:ProductID" json:"inventory,omitempty"`
	Notifications []Notification `gorm:"foreignKey:ProductID" json:"notifications,omitempty"`
	OrderItems    []OrderItem    `gorm:"foreignKey:ProductID" json:"orderItems,omitempty"`
	StockHistory  []StockHistory `gorm:"foreignKey:ProductID" json:"stockHistory,omitempty"`
	Feedbacks     []Feedback     `gorm:"foreignKey:ProductID" json:"feedbacks,omitempty"`
}

// TableName specifies the table name for Product model
func (Product) TableName() string {
	return "Product"
}

// Inventory model - mirrors Prisma Inventory model
type Inventory struct {
	ID        int `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ProductID int `gorm:"unique;not null;column:productId" json:"productId"`
	OutletID  int `gorm:"not null;column:outletId" json:"outletId"`
	Quantity  int `gorm:"not null;column:quantity" json:"quantity"`
	Threshold int `gorm:"not null;column:threshold" json:"threshold"`

	// Relationships
	Outlet  Outlet  `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
	Product Product `gorm:"foreignKey:ProductID;references:ID" json:"product,omitempty"`
}

// TableName specifies the table name for Inventory model
func (Inventory) TableName() string {
	return "Inventory"
}

// StockHistory model - mirrors Prisma StockHistory model
type StockHistory struct {
	ID        int         `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	ProductID int         `gorm:"not null;column:productId" json:"productId"`
	OutletID  int         `gorm:"not null;column:outletId" json:"outletId"`
	Quantity  int         `gorm:"not null;column:quantity" json:"quantity"`
	Action    StockAction `gorm:"type:text;not null;column:action" json:"action"`
	Timestamp time.Time   `gorm:"autoCreateTime;column:timestamp" json:"timestamp"`

	// Relationships
	Outlet  Outlet  `gorm:"foreignKey:OutletID;references:ID" json:"outlet,omitempty"`
	Product Product `gorm:"foreignKey:ProductID;references:ID" json:"product,omitempty"`
}

// TableName specifies the table name for StockHistory model
func (StockHistory) TableName() string {
	return "StockHistory"
}
