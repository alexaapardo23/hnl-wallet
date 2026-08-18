package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"hnl-wallet/backend/database"
	"hnl-wallet/backend/seed"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("warning: could not load .env: %v", err)
	}

	dataFile := os.Getenv("DATA_FILE")

	if dataFile == "" {
		log.Fatal("DATA_FILE environment variable is not set")
	}

	data, err := seed.LoadData(dataFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Loaded %d users, %d accounts and %d transactions\n",
		len(data.Users),
		len(data.Accounts),
		len(data.Transactions),
	)

	pool, err := database.NewPostgresPool()
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	ctx := context.Background()

	if err := seed.SeedPostgres(ctx, pool, data); err != nil {
		log.Fatal(err)
	}

	log.Println("PostgreSQL seed completed successfully")
}