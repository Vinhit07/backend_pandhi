package main

import (
	"backend_pandhi/pkg/config"
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"backend_pandhi/pkg/utils"
	"log"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	seedSuperAdmin()
	seedAdmin()
}

func seedSuperAdmin() {
	email := "superadmin@gmail.com"
	password := "superadmin123"

	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err == nil {
		log.Printf("SuperAdmin %s already exists", email)
		return
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Create SuperAdmin
	// Note: Providing dummy phone as it might be required or unique
	phone := "0000000000"
	// Using pointer for phone as per model
	user = models.User{
		Name:     "Super Admin",
		Email:    email,
		Password: &hashedPassword,
		Role:     models.RoleSuperAdmin, // Ensure this constant exists or use "SUPERADMIN"
		Phone:    &phone,
		OutletID: nil, // SuperAdmin might not need outlet
	}

	if err := database.DB.Create(&user).Error; err != nil {
		log.Fatal("Failed to create SuperAdmin:", err)
	}

	log.Printf("✅ SuperAdmin %s created successfully", email)
}

func seedAdmin() {
	email := "staff1@gmail.com"
	password := "staff123"

	var admin models.Admin
	if err := database.DB.Where("email = ?", email).First(&admin).Error; err == nil {
		log.Printf("Admin %s already exists", email)

		// Ensure verified
		if !admin.IsVerified {
			admin.IsVerified = true
			database.DB.Save(&admin)
			log.Printf("✅ Admin %s verified", email)
		}
		return
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	phone := "1111111111"
	admin = models.Admin{
		Name:       "Test Admin",
		Email:      email,
		Password:   hashedPassword,
		IsVerified: true,
		Phone:      &phone,
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		log.Fatal("Failed to create Admin:", err)
	}

	log.Printf("✅ Admin %s created successfully", email)
}
