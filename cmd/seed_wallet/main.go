package main

import (
	"backend_pandhi/pkg/config"
	"backend_pandhi/pkg/database"
	"backend_pandhi/pkg/models"
	"fmt"
	"log"
	"time"
)

func main() {
	// Load config
	config.LoadConfig()

	// Connect to database
	if err := database.InitDatabase(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Find all customers and add 10 lakh to their wallets
	var customers []models.CustomerDetails
	database.DB.Find(&customers)

	if len(customers) == 0 {
		log.Fatal("No customers found in database")
	}

	for _, customer := range customers {
		var wallet models.Wallet
		err := database.DB.Where("\"customerId\" = ?", customer.ID).First(&wallet).Error

		now := time.Now()
		if err != nil {
			// Create wallet with 10 lakh
			wallet = models.Wallet{
				CustomerID:     customer.ID,
				Balance:        1000000, // 10 lakh INR
				TotalRecharged: 1000000,
				TotalUsed:      0,
				LastRecharged:  &now,
			}
			database.DB.Create(&wallet)
			fmt.Printf("✅ Created wallet for customer %d with ₹10,00,000\n", customer.ID)
		} else {
			// Update existing wallet
			database.DB.Model(&wallet).Updates(map[string]interface{}{
				"balance":            1000000,
				"\"totalRecharged\"": 1000000,
				"\"lastRecharged\"":  &now,
			})
			fmt.Printf("✅ Updated wallet for customer %d to ₹10,00,000\n", customer.ID)
		}
	}

	fmt.Println("\n🎉 All customer wallets updated to ₹10,00,000!")
}
