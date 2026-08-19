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

	accounts := make([]tb.Account, 0, len(data.Accounts)+1)

	accounts = append(accounts, tb.Account{
		ID:     tigerbeetle.SystemAccountID,
		Ledger: ledger,
		Code:   tigerbeetle.CodeSystem,
	})

	for i, account := range data.Accounts {
		accountCode, err := tigerbeetle.AccountTypeCode(account.AccountType)
		if err != nil {
			log.Fatalf("account %s: %v", account.AccountNumber, err)
		}

		userData128, err := tigerbeetle.AccountNumberToUserData128(account.AccountNumber)
		if err != nil {
			log.Fatalf("account %s: %v", account.AccountNumber, err)
		}

		accounts = append(accounts, tb.Account{
			ID:          tigerbeetle.AccountID(i),
			Ledger:      ledger,
			Code:        accountCode,
			UserData128: userData128,
		})
	}

	const batchSize = 1000
	created := 0

	for start := 0; start < len(accounts); start += batchSize {
		end := min(start+batchSize, len(accounts))
		batch := accounts[start:end]

		results, err := client.CreateAccounts(batch)
		if err != nil {
			log.Fatalf("failed to create accounts batch [%d:%d]: %v", start, end, err)
		}

		failed := 0
		for _, result := range results {
			if result.Status != tb.AccountCreated {
				log.Printf("account creation error: %s", result.Status)
				failed++
			}
		}

		created += len(batch) - failed
	}

	fmt.Printf(
		"Created %d/%d accounts in TigerBeetle (1 SYSTEM + %d real accounts)\n",
		created,
		len(accounts),
		len(data.Accounts),
	)
}
