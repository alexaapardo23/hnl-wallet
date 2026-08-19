package api

import (
	"errors"
	"fmt"
	"net/http"

	tb "github.com/tigerbeetle/tigerbeetle-go"

	"hnl-wallet/backend/tigerbeetle"
)

type depositRequest struct {
	Amount float64 `json:"amount"`
}

type depositResponse struct {
	TransactionID string  `json:"transaction_id"`
	AccountNumber string  `json:"account_number"`
	Amount        float64 `json:"amount"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
}

// depositHandler credits an account from SystemAccountID ("the bank"),
// following:
//
//	request -> JWT (user_id) -> ownership check -> amount > 0 -> SYSTEM -> account -> TigerBeetle
//
// It reuses the same debit-leaves/credit-arrives convention and CodeDeposit
// as the historical deposits imported by cmd/seed-transactions — see
// README Historical Transaction Import.
func (s *Server) depositHandler(w http.ResponseWriter, r *http.Request) {
	// JWT -> user_id (already resolved by requireAuth middleware).
	userID := userIDFromContext(r)
	accountNumber := r.PathValue("account_number")

	// Verify the account belongs to the authenticated user.
	account, err := s.lookupOwnedAccount(r.Context(), accountNumber, userID)
	if errors.Is(err, errAccountNotFound) {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up account")
		return
	}

	var req depositRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate amount > 0 (also rejects NaN, since NaN > 0 is false).
	if !(req.Amount > 0) {
		writeError(w, http.StatusBadRequest, "amount must be greater than 0")
		return
	}

	// SYSTEM -> account, in TigerBeetle.
	transferID := tb.ID()
	results, err := s.TB.CreateTransfers([]tb.Transfer{{
		ID:              transferID,
		DebitAccountID:  tigerbeetle.SystemAccountID,
		CreditAccountID: account.TigerBeetleID,
		Amount:          tb.ToUint128(tigerbeetle.CentsFromDollars(req.Amount)),
		Ledger:          ledger,
		Code:            tigerbeetle.CodeDeposit,
	}})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create deposit")
		return
	}
	for _, result := range results {
		if result.Status != tb.TransferCreated {
			writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("deposit rejected: %s", result.Status))
			return
		}
	}

	balanceCents, err := s.balanceCents(account.TigerBeetleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "deposit succeeded but failed to fetch new balance")
		return
	}

	writeJSON(w, http.StatusCreated, depositResponse{
		TransactionID: transferID.String(),
		AccountNumber: account.AccountNumber,
		Amount:        req.Amount,
		Balance:       float64(balanceCents) / 100,
		Currency:      "USD",
	})
}
