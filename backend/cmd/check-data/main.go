package main

import (
	"fmt"
	"log"

	"hnl-wallet/backend/seed"
)

func main() {
	data, err := seed.LoadData("../data/data.json")
	if err != nil {
		log.Fatal(err)
	}

	// Map user ID -> user information
	users := make(map[string]struct {
		FullName string
		Password string
	})

	for _, user := range data.Users {
		users[user.ID] = struct {
			FullName string
			Password string
		}{
			FullName: user.FullName,
			Password: user.Password,
		}
	}

	// Map user ID -> accounts
	accountsByUser := make(map[string][]string)

	for _, account := range data.Accounts {
		accountsByUser[account.UserID] = append(
			accountsByUser[account.UserID],
			fmt.Sprintf(
				"%s ($%.2f)",
				account.AccountType,
				account.InitialBalance,
			),
		)
	}

	// Map email -> user IDs
	usersByEmail := make(map[string][]string)

	for _, user := range data.Users {
		usersByEmail[user.Email] = append(
			usersByEmail[user.Email],
			user.ID,
		)
	}

	for email, userIDs := range usersByEmail {
		if len(userIDs) <= 1 {
			continue
		}

		fmt.Println("========================================")
		fmt.Printf("Duplicate email: %s\n", email)

		for _, userID := range userIDs {
			user := users[userID]

			fmt.Printf("\nUser ID: %s\n", userID)
			fmt.Printf("Name: %s\n", user.FullName)
			fmt.Printf("Password: %s\n", user.Password)

			fmt.Println("Accounts:")

			if accounts := accountsByUser[userID]; len(accounts) > 0 {
				for _, account := range accounts {
					fmt.Printf("  - %s\n", account)
				}
			} else {
				fmt.Println("  - No accounts")
			}
		}
	}
}