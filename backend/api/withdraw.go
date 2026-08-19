package api

import (
	"errors"
	"fmt"
	"net/http"

	tb "github.com/tigerbeetle/tigerbeetle-go"

	"hnl-wallet/backend/tigerbeetle"
)

type withdrawRequest struct {
	Amount float64 `json:"amount"`
}

type withdrawResponse struct {
	TransactionID string  `json:"transaction_id"`
	AccountNumber string  `json:"account_number"`
	Amount        float64 `json:"amount"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
}

// withdrawHandler debits an account to SystemAccountID ("the bank"),
// following:
//
//	request -> JWT (user_id) -> ownership check -> amount > 0 -> sufficient balance -> account -> SYSTEM -> TigerBeetle
//
// Unlike depositHandler, a withdrawal can push a balance negative — the
// overdrafts seen in the seed dataset are only tolerated there (see README
// Seed anomalies are not permitted going forward). None of this project's
// accounts (seeded or API-created) were created with the
// debits_must_not_exceed_credits flag — TigerBeetle accounts are immutable,
// so it can't be added retroactively — so this handler enforces "no new
// overdraft" itself: it reads the live balance and rejects any withdrawal
// that would take it below zero, rather than relying on TigerBeetle to
// reject the transfer.
func (s *Server) withdrawHandler(w http.ResponseWriter, r *http.Request) {
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

	var req withdrawRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate amount > 0 (also rejects NaN, since NaN > 0 is false).
	if !(req.Amount > 0) {
		writeError(w, http.StatusBadRequest, "amount must be greater than 0")
		return
	}

	amountCents := tigerbeetle.CentsFromDollars(req.Amount)

	currentBalanceCents, err := s.balanceCents(account.TigerBeetleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check current balance")
		return
	}
	if int64(amountCents) > currentBalanceCents {
		writeError(w, http.StatusUnprocessableEntity, "insufficient funds")
		return
	}

	// account -> SYSTEM, in TigerBeetle.
	transferID := tb.ID()
	results, err := s.TB.CreateTransfers([]tb.Transfer{{
		ID:              transferID,
		DebitAccountID:  account.TigerBeetleID,
		CreditAccountID: tigerbeetle.SystemAccountID,
		Amount:          tb.ToUint128(amountCents),
		Ledger:          ledger,
		Code:            tigerbeetle.CodeWithdrawal,
	}})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create withdrawal")
		return
	}
	for _, result := range results {
		if result.Status != tb.TransferCreated {
			writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("withdrawal rejected: %s", result.Status))
			return
		}
	}

	newBalanceCents, err := s.balanceCents(account.TigerBeetleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "withdrawal succeeded but failed to fetch new balance")
		return
	}

	writeJSON(w, http.StatusCreated, withdrawResponse{
		TransactionID: transferID.String(),
		AccountNumber: account.AccountNumber,
		Amount:        req.Amount,
		Balance:       float64(newBalanceCents) / 100,
		Currency:      "USD",
	})
}
