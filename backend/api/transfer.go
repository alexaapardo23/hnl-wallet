package api

import (
	"errors"
	"fmt"
	"net/http"

	tb "github.com/tigerbeetle/tigerbeetle-go"

	"hnl-wallet/backend/tigerbeetle"
)

type transferRequest struct {
	FromAccount string  `json:"from_account"`
	ToAccount   string  `json:"to_account"`
	Amount      float64 `json:"amount"`
}

type transferResponse struct {
	TransactionID string  `json:"transaction_id"`
	FromAccount   string  `json:"from_account"`
	ToAccount     string  `json:"to_account"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
}

// transferHandler moves money between two accounts:
//
//	JWT
//	  ↓
//	identify user
//	  ↓
//	verify from_account belongs to the user
//	  ↓
//	verify destination account
//	  ↓
//	validate amount
//	  ↓
//	verify funds
//	  ↓
//	TigerBeetle
//	A ──────────→ B
//
// Whether the transfer is reported as "transfer" or "internal_transfer"
// (CodeTransfer vs. CodeInternalTransfer) depends on whether from_account
// and to_account share an owner — the same distinction cmd/seed-transactions
// uses when replaying data.json.
func (s *Server) transferHandler(w http.ResponseWriter, r *http.Request) {
	// JWT -> identify user (already resolved by requireAuth middleware).
	userID := userIDFromContext(r)

	var req transferRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FromAccount == "" || req.ToAccount == "" {
		writeError(w, http.StatusBadRequest, "from_account and to_account are required")
		return
	}
	if req.FromAccount == req.ToAccount {
		writeError(w, http.StatusBadRequest, "from_account and to_account must be different")
		return
	}

	// Verify from_account belongs to the authenticated user.
	from, err := s.lookupOwnedAccount(r.Context(), req.FromAccount, userID)
	if errors.Is(err, errAccountNotFound) {
		writeError(w, http.StatusNotFound, "from_account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up from_account")
		return
	}

	// Verify the destination account exists (it may belong to another user).
	to, err := s.lookupAccountByNumber(r.Context(), req.ToAccount)
	if errors.Is(err, errAccountNotFound) {
		writeError(w, http.StatusNotFound, "to_account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up to_account")
		return
	}

	// Validate amount > 0 (also rejects NaN, since NaN > 0 is false).
	if !(req.Amount > 0) {
		writeError(w, http.StatusBadRequest, "amount must be greater than 0")
		return
	}

	amountCents := tigerbeetle.CentsFromDollars(req.Amount)

	// Verify funds: see README Withdrawals for why this is an
	// application-level check rather than a TigerBeetle account flag.
	currentBalanceCents, err := s.balanceCents(from.TigerBeetleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check current balance")
		return
	}
	if int64(amountCents) > currentBalanceCents {
		writeError(w, http.StatusUnprocessableEntity, "insufficient funds")
		return
	}

	transferType := tigerbeetle.CodeTransfer
	typeName := "transfer"
	if to.UserID == userID {
		transferType = tigerbeetle.CodeInternalTransfer
		typeName = "internal_transfer"
	}

	// A ──────────→ B, in TigerBeetle.
	transferID := tb.ID()
	results, err := s.TB.CreateTransfers([]tb.Transfer{{
		ID:              transferID,
		DebitAccountID:  from.TigerBeetleID,
		CreditAccountID: to.TigerBeetleID,
		Amount:          tb.ToUint128(amountCents),
		Ledger:          ledger,
		Code:            transferType,
	}})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create transfer")
		return
	}
	for _, result := range results {
		if result.Status != tb.TransferCreated {
			writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("transfer rejected: %s", result.Status))
			return
		}
	}

	newBalanceCents, err := s.balanceCents(from.TigerBeetleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "transfer succeeded but failed to fetch new balance")
		return
	}

	writeJSON(w, http.StatusCreated, transferResponse{
		TransactionID: transferID.String(),
		FromAccount:   from.AccountNumber,
		ToAccount:     to.AccountNumber,
		Amount:        req.Amount,
		Type:          typeName,
		Balance:       float64(newBalanceCents) / 100,
		Currency:      "USD",
	})
}
