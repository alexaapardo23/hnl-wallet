package main

import (
	"fmt"
	"log"
	"net/http"

	"hnl-wallet/backend/database"
	tbClient "hnl-wallet/backend/tigerbeetle"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "HNL Wallet API is running")
}

func main() {
	db, err := database.ConnectPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL")

	tb, err := tbClient.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer tb.Close()

	log.Println("Connected to TigerBeetle")

	http.HandleFunc("/health", healthHandler)

	log.Println("HNL Wallet API running on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}