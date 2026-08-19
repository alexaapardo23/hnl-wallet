package main

import (
	"fmt"
	"log"
	"math"
	"os"

	tb "github.com/tigerbeetle/tigerbeetle-go"

	"hnl-wallet/backend/models"
	"hnl-wallet/backend/seed"
	"hnl-wallet/backend/tigerbeetle"
)

const (
	ledger          = 700
	externalAccount = "EXTERNAL"
)

func centsFromDollars(dollars float64) uint64 {
	return uint64(math.Round(dollars * 100))
}

// resolveAccountID maps a data.json account_number (or the EXTERNAL
// sentinel) to its TigerBeetle account ID.
func resolveAccountID(accountNumber string, index map[string]int) (tb.Uint128, error) {
	if accountNumber == externalAccount {
		return tigerbeetle.SystemAccountID, nil
	}

	i, ok := index[accountNumber]
	if !ok {
		return tb.Uint128{}, fmt.Errorf("unknown account_number: %q", accountNumber)
	}

	return tigerbeetle.AccountID(i), nil
}

func transferCode(transactionType string) (uint16, error) {
	switch transactionType {
	case "deposit":
		return tigerbeetle.CodeDeposit, nil
	case "withdrawal":
		return tigerbeetle.CodeWithdrawal, nil
	case "transfer":
		return tigerbeetle.CodeTransfer, nil
	case "internal_transfer":
		return tigerbeetle.CodeInternalTransfer, nil
	default:
		return 0, fmt.Errorf("unknown transaction type: %q", transactionType)
	}
}

func buildTransfer(t models.Transaction, index map[string]int) (tb.Transfer, error) {
	debitID, err := resolveAccountID(t.FromAccount, index)
	if err != nil {
		return tb.Transfer{}, err
	}

	creditID, err := resolveAccountID(t.ToAccount, index)
	if err != nil {
		return tb.Transfer{}, err
	}

	code, err := transferCode(t.Type)
	if err != nil {
		return tb.Transfer{}, err
	}

	return tb.Transfer{
		ID:              tb.ID(),
		DebitAccountID:  debitID,
		CreditAccountID: creditID,
		Amount:          tb.ToUint128(centsFromDollars(t.Amount)),
		Ledger:          ledger,
		Code:            code,
	}, nil
}

func main() {
	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = "../data/data.json"
	}

	data, err := seed.LoadData(dataFile)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d transactions from %s\n", len(data.Transactions), dataFile)

	accountIndex := make(map[string]int, len(data.Accounts))
	for i, account := range data.Accounts {
		accountIndex[account.AccountNumber] = i
	}

	client, err := tigerbeetle.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	transfers := make([]tb.Transfer, 0, len(data.Transactions))

	for _, t := range data.Transactions {
		transfer, err := buildTransfer(t, accountIndex)
		if err != nil {
			log.Fatalf("transaction %s -> %s: %v", t.FromAccount, t.ToAccount, err)
		}

		transfers = append(transfers, transfer)
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

	fmt.Printf("Created %d/%d historical transfers\n", created, len(transfers))
}
