package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"hnl-wallet/backend/api"
	"hnl-wallet/backend/database"
	"hnl-wallet/backend/openrouter"
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

	mcpServerURL := os.Getenv("MCP_SERVER_URL")
	if mcpServerURL == "" {
		mcpServerURL = "http://localhost:8081/mcp"
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	var openRouterClient *openrouter.Client
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		model := os.Getenv("OPENROUTER_MODEL")
		if model == "" {
			log.Fatal("OPENROUTER_MODEL environment variable is not set")
		}
		openRouterClient = openrouter.NewClient(apiKey, model)
		log.Printf("Chat enabled via OpenRouter (model: %s)", model)
	} else {
		log.Println("OPENROUTER_API_KEY not set — POST /chat is disabled")
	}

	server := &api.Server{
		DB:            db,
		TB:            tb,
		JWTSecret:     []byte(jwtSecret),
		OpenRouter:    openRouterClient,
		MCPServerURL:  mcpServerURL,
		AllowedOrigin: frontendURL,
	}

	log.Println("HNL Wallet API running on :8080")

	if err := http.ListenAndServe(":8080", api.NewRouter(server)); err != nil {
		log.Fatal(err)
	}
}
