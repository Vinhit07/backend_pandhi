package database

import (
	"backend_pandhi/pkg/config"
	"backend_pandhi/pkg/models"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// QuotedNamingStrategy wraps the default naming strategy and quotes all identifiers
// This ensures PostgreSQL uses case-sensitive column names as defined in the schema
type QuotedNamingStrategy struct {
	schema.NamingStrategy
}

// ColumnName quotes column names for PostgreSQL case-sensitivity
func (q QuotedNamingStrategy) ColumnName(table, column string) string {
	return fmt.Sprintf("\"%s\"", q.NamingStrategy.ColumnName(table, column))
}

// TableName quotes table names
func (q QuotedNamingStrategy) TableName(table string) string {
	return fmt.Sprintf("\"%s\"", q.NamingStrategy.TableName(table))
}

// JoinTableName quotes join table names
func (q QuotedNamingStrategy) JoinTableName(joinTable string) string {
	return fmt.Sprintf("\"%s\"", q.NamingStrategy.JoinTableName(joinTable))
}

// InitDatabase initializes the database connection
func InitDatabase() error {
	var err error

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Info),
		PrepareStmt: false,
		NamingStrategy: QuotedNamingStrategy{
			schema.NamingStrategy{
				SingularTable: false,
			},
		},
	}

	// Development mode - verbose logging
	if config.IsDevelopment() {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		// Production mode - only errors
		gormConfig.Logger = logger.Default.LogMode(logger.Error)
	}

	// Connect to PostgreSQL with implicit prepared statements disabled
	DB, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  config.AppConfig.DatabaseURL,
		PreferSimpleProtocol: true, // Disable implicit prepared statements to avoid "prepared statement already exists" errors
	}), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings (matching Express.js pg-pool behavior)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	log.Println("✅ Database connection established")

	return nil
}

// AutoMigrate runs auto-migration for all models
func AutoMigrate() error {
	log.Println("🔄 Running database migrations...")

	err := DB.AutoMigrate(
		// Core models
		&models.Outlet{},
		&models.User{},
		&models.CustomerDetails{},
		&models.StaffDetails{},
		&models.StaffPermission{},
		&models.Badge{},

		// Product & Inventory
		&models.Product{},
		&models.Inventory{},
		&models.StockHistory{},

		// Cart
		&models.Cart{},
		&models.CartItem{},

		// Orders
		&models.Order{},
		&models.OrderItem{},

		// Wallet
		&models.Wallet{},
		&models.WalletTransaction{},

		// Admin
		&models.Admin{},
		&models.AdminOutlet{},
		&models.AdminPermission{},

		// Support
		&models.Ticket{},
		&models.Coupon{},
		&models.CouponUsage{},
		&models.Expense{},

		// Notifications
		&models.Notification{},
		&models.ScheduledNotification{},
		&models.NotificationDelivery{},
		&models.UserDeviceToken{},

		// Outlet Management
		&models.OutletAvailability{},
		&models.OutletAppManagement{},

		// Feedback & Quota
		&models.Feedback{},
		&models.UserFreeQuota{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✅ Database migrations completed")

	// Create indexes matching Prisma schema
	createIndexes()

	// Create Postgres functions
	createPostgresFunctions()

	return nil
}

// createPostgresFunctions creates any raw SQL functions needed
func createPostgresFunctions() {
	log.Println("🔄 Creating PostgreSQL functions...")

	functionSQL := `
	CREATE OR REPLACE FUNCTION verify_badge_access(p_badge_id text)
	RETURNS json AS $$
	DECLARE
		v_email text;
		v_is_claimed boolean;
		v_name text;
	BEGIN
		SELECT email, is_claimed, name INTO v_email, v_is_claimed, v_name
		FROM badges
		WHERE badge_id = p_badge_id;

		IF NOT FOUND THEN
			RETURN json_build_object('success', false, 'message', 'Badge not found');
		END IF;

		IF v_is_claimed THEN
			RETURN json_build_object('success', false, 'message', 'Badge previously claimed');
		END IF;

		RETURN json_build_object('success', true, 'email', v_email, 'name', v_name);
	END;
	$$ LANGUAGE plpgsql;
	`

	if err := DB.Exec(functionSQL).Error; err != nil {
		log.Printf("❌ Failed to create verify_badge_access function: %v", err)
	} else {
		log.Println("✅ verify_badge_access function created or updated")
	}
}

// createIndexes creates additional indexes to match Prisma schema
func createIndexes() {
	log.Println("🔄 Creating additional indexes...")

	// Product indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS "idx_product_isveg" ON "Product"("isVeg")`)

	// OutletAvailability indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS "OutletAvailability_outletId_date_idx" ON "OutletAvailability"("outletId", "date")`)

	// NotificationDelivery indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS "NotificationDelivery_scheduledNotificationId_status_idx" ON "NotificationDelivery"("scheduledNotificationId", "status")`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS "NotificationDelivery_userId_status_idx" ON "NotificationDelivery"("userId", "status")`)

	// Feedback indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS "Feedback_userId_idx" ON "Feedback"("userId")`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS "Feedback_productId_idx" ON "Feedback"("productId")`)

	// UserFreeQuota indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS "UserFreeQuota_userId_consumptionDate_idx" ON "UserFreeQuota"("userId", "consumptionDate")`)

	// OutletAppManagement index
	DB.Exec(`CREATE INDEX IF NOT EXISTS "OutletAppManagement_outletId_feature_idx" ON "OutletAppManagement"("outletId", "feature")`)

	// Unique constraints
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "AdminOutlet_adminId_outletId_key" ON "AdminOutlet"("adminId", "outletId")`)
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "AdminPermission_adminOutletId_type_key" ON "AdminPermission"("adminOutletId", "type")`)
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "CartItem_cartId_productId_key" ON "CartItem"("cartId", "productId")`)
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "Feedback_orderId_productId_key" ON "Feedback"("orderId", "productId")`)
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "NotificationDelivery_scheduledNotificationId_userId_devi_key" ON "NotificationDelivery"("scheduledNotificationId", "userId", "deviceToken")`)
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "OutletAppManagement_outletId_feature_key" ON "OutletAppManagement"("outletId", "feature")`)
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "OutletAvailability_outletId_date_key" ON "OutletAvailability"("outletId", "date")`)
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "UserFreeQuota_userId_consumptionDate_key" ON "UserFreeQuota"("userId", "consumptionDate")`)

	log.Println("✅ Additional indexes created")
}

// CloseDatabase closes the database connection
func CloseDatabase() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Error getting database instance: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	} else {
		log.Println("✅ Database connection closed")
	}
}
