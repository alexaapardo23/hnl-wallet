package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"hnl-wallet/backend/api"
	"hnl-wallet/backend/database"
	tbClient "hnl-wallet/backend/tigerbeetle"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("warning: could not load .env: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	db, err := database.NewPostgresPool()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tb, err := tbClient.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer tb.Close()

	log.Println("Connected to TigerBeetle")

	server := &api.Server{
		DB:        db,
		TB:        tb,
		JWTSecret: []byte(jwtSecret),
	}

	log.Println("HNL Wallet API running on :8080")

	if err := http.ListenAndServe(":8080", api.NewRouter(server)); err != nil {
		log.Fatal(err)
	}
}
