package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	tb "github.com/tigerbeetle/tigerbeetle-go"

	"hnl-wallet/backend/tigerbeetle"
)

const ledger = 700

type createAccountRequest struct {
	AccountType string `json:"account_type"`
}

type accountResponse struct {
	AccountNumber  string  `json:"account_number"`
	InitialBalance float64 `json:"initial_balance"`
	Currency       string  `json:"currency"`
	AccountType    string  `json:"account_type"`
}

// createAccountHandler opens a new account for the authenticated user.
//
// account_number and the TigerBeetle account ID are both drawn from
// tigerbeetle_account_seq (starting at 1607, right after the 1605 seeded
// accounts occupying IDs 2..1606 — see README Account ID Mapping), keeping
// every account's ID small and human-readable instead of switching to
// TigerBeetle's random ID() generator only for accounts created here.
func (s *Server) createAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	accountCode, err := tigerbeetle.AccountTypeCode(req.AccountType)
	if err != nil {
		writeError(w, http.StatusBadRequest, "account_type must be one of: checking, savings, investment")
		return
	}

	userID := userIDFromContext(r)
	ctx := r.Context()

	var sequence uint64
	if err := s.DB.QueryRow(ctx, `SELECT nextval('tigerbeetle_account_seq')`).Scan(&sequence); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to allocate account ID")
		return
	}

	tbAccountID := tb.ToUint128(sequence)

	accountNumber, err := randomAccountNumber(sequence)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate account number")
		return
	}

	userData128, err := tigerbeetle.AccountNumberToUserData128(accountNumber)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode account number")
		return
	}

	results, err := s.TB.CreateAccounts([]tb.Account{{
		ID:          tbAccountID,
		Ledger:      ledger,
		Code:        accountCode,
		UserData128: userData128,
	}})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create TigerBeetle account")
		return
	}
	for _, result := range results {
		if result.Status != tb.AccountCreated {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("TigerBeetle rejected account: %s", result.Status))
			return
		}
	}

	_, err = s.DB.Exec(
		ctx,
		`
		INSERT INTO accounts (account_number, user_id, initial_balance, currency, account_type, tigerbeetle_account_id)
		VALUES ($1, $2, 0, 'USD', $3, $4)
		`,
		accountNumber, userID, req.AccountType, tigerbeetle.UUIDString(tbAccountID),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save account")
		return
	}

	writeJSON(w, http.StatusCreated, accountResponse{
		AccountNumber:  accountNumber,
		InitialBalance: 0,
		Currency:       "USD",
		AccountType:    req.AccountType,
	})
}

// randomAccountNumber builds a "4001-XXXX-XXXX-NNNN" account number
// matching the seed dataset's format, with the trailing group set to
// sequence so it stays a stable, collision-free identifier.
func randomAccountNumber(sequence uint64) (string, error) {
	var randomDigits [8]byte
	if _, err := rand.Read(randomDigits[:]); err != nil {
		return "", err
	}

	group := func(b [8]byte, offset int) uint16 {
		return (uint16(b[offset])<<8 | uint16(b[offset+1])) % 10000
	}

	return fmt.Sprintf(
		"4001-%04d-%04d-%04d",
		group(randomDigits, 0),
		group(randomDigits, 2),
		sequence-1,
	), nil
}

func (s *Server) listAccountsHandler(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	rows, err := s.DB.Query(
		r.Context(),
		`SELECT account_number, initial_balance, currency, account_type FROM accounts WHERE user_id = $1 ORDER BY account_number`,
		userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list accounts")
		return
	}
	defer rows.Close()

	accounts := []accountResponse{}
	for rows.Next() {
		var a accountResponse
		if err := rows.Scan(&a.AccountNumber, &a.InitialBalance, &a.Currency, &a.AccountType); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read accounts")
			return
		}
		accounts = append(accounts, a)
	}

	writeJSON(w, http.StatusOK, accounts)
}

type balanceResponse struct {
	AccountNumber string  `json:"account_number"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
}

// accountBalanceHandler returns an account's live balance computed from
// TigerBeetle (CreditsPosted - DebitsPosted), not a stored column — see
// README Financial Data Model: TigerBeetle is the source of truth.
func (s *Server) accountBalanceHandler(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	accountNumber := r.PathValue("account_number")

	var tigerbeetleAccountID string
	err := s.DB.QueryRow(
		r.Context(),
		`SELECT tigerbeetle_account_id FROM accounts WHERE account_number = $1 AND user_id = $2`,
		accountNumber, userID,
	).Scan(&tigerbeetleAccountID)

	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up account")
		return
	}

	tbID, err := tigerbeetle.ParseUUIDString(tigerbeetleAccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to decode TigerBeetle account ID")
		return
	}

	tbAccounts, err := s.TB.LookupAccounts([]tb.Uint128{tbID})
	if err != nil || len(tbAccounts) == 0 {
		writeError(w, http.StatusInternalServerError, "failed to fetch balance from TigerBeetle")
		return
	}

	credits := tbAccounts[0].CreditsPosted.BigInt()
	debits := tbAccounts[0].DebitsPosted.BigInt()
	cents := credits.Sub(credits, debits).Int64()

	writeJSON(w, http.StatusOK, balanceResponse{
		AccountNumber: accountNumber,
		Balance:       float64(cents) / 100,
		Currency:      "USD",
	})
}
