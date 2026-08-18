package models

import "time"

type Data struct {
	Users        []User        `json:"users"`
	Accounts     []Account     `json:"accounts"`
	Transactions []Transaction `json:"transactions"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
}

type Account struct {
	AccountNumber   string  `json:"account_number"`
	UserID          string  `json:"user_id"`
	InitialBalance  float64 `json:"initial_balance"`
	Currency        string  `json:"currency"`
	AccountType     string  `json:"account_type"`
}

type Transaction struct {
	FromAccount string    `json:"from_account"`
	ToAccount   string    `json:"to_account"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}