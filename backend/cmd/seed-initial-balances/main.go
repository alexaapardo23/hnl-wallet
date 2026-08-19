package main

import (
	"fmt"
	"log"
	"os"

	tb "github.com/tigerbeetle/tigerbeetle-go"

	"hnl-wallet/backend/seed"
	"hnl-wallet/backend/tigerbeetle"
)

const ledger = 700

func main() {
	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = "../data/data.json"
	}

	data, err := seed.LoadData(dataFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d accounts from %s\n", len(data.Accounts), dataFile)

	client, err := tigerbeetle.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	transfers := make([]tb.Transfer, 0, len(data.Accounts))
	var totalCents uint64

	for i, account := range data.Accounts {
		amountCents := tigerbeetle.CentsFromDollars(account.InitialBalance)
		totalCents += amountCents

		transfers = append(transfers, tb.Transfer{
			ID:              tb.ID(),
			DebitAccountID:  tigerbeetle.SystemAccountID,
			CreditAccountID: tigerbeetle.AccountID(i),
			Amount:          tb.ToUint128(amountCents),
			Ledger:          ledger,
			Code:            tigerbeetle.CodeInitialBalance,
		})
	}

	const batchSize = 1000
	created := 0

	for start := 0; start < len(transfers); start += batchSize {
		end := min(start+batchSize, len(transfers))
		batch := transfers[start:end]

		results, err := client.CreateTransfers(batch)
		if err != nil {
			log.Fatalf("failed to create transfers batch [%d:%d]: %v", start, end, err)
		}

		failed := 0
		for _, result := range results {
			if result.Status != tb.TransferCreated {
				log.Printf("transfer creation error: %s", result.Status)
				failed++
			}
		}

		created += len(batch) - failed
	}

	fmt.Printf(
		"Created %d/%d INITIAL_BALANCE transfers (total %d cents = $%.2f)\n",
		created,
		len(transfers),
		totalCents,
		float64(totalCents)/100,
	)
}
