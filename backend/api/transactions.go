package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	tb "github.com/tigerbeetle/tigerbeetle-go"

	"hnl-wallet/backend/tigerbeetle"
)

const (
	defaultTransactionsLimit = 50
	maxTransactionsLimit     = 200
)

type transactionResponse struct {
	ID                        string    `json:"id"`
	Type                      string    `json:"type"`
	Direction                 string    `json:"direction"`
	Amount                    float64   `json:"amount"`
	CounterpartyAccountNumber string    `json:"counterparty_account_number"`
	Timestamp                 time.Time `json:"timestamp"`
}

// accountTransactionsHandler returns an account's transfer history straight
// from TigerBeetle (via GetAccountTransfers), not the PostgreSQL
// `transactions` table — that table is only ever populated by cmd/seed from
// data.json and would silently miss any transfer created through this API
// (e.g. a future POST /transfers). TigerBeetle is the single source of
// truth for financial activity regardless of where a transfer originated.
func (s *Server) accountTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	account, err := s.lookupOwnedAccount(r.Context(), r.PathValue("account_number"), userIDFromContext(r))
	if errors.Is(err, errAccountNotFound) {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up account")
		return
	}

	limit := parseLimit(r.URL.Query().Get("limit"))

	filter := tb.AccountFilter{
		AccountID: account.TigerBeetleID,
		Limit:     uint32(limit),
		Flags: tb.AccountFilterFlags{
			Debits:   true,
			Credits:  true,
			Reversed: true, // newest first
		}.ToUint32(),
	}

	transfers, err := s.TB.GetAccountTransfers(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch transactions from TigerBeetle")
		return
	}

	counterpartyNumbers, err := s.resolveCounterpartyAccountNumbers(r.Context(), account.TigerBeetleID, transfers)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve counterparty accounts")
		return
	}

	response := make([]transactionResponse, 0, len(transfers))
	for _, t := range transfers {
		direction := "incoming"
		counterpartyID := t.DebitAccountID
		if t.DebitAccountID == account.TigerBeetleID {
			direction = "outgoing"
			counterpartyID = t.CreditAccountID
		}

		counterparty := "EXTERNAL"
		if counterpartyID != tigerbeetle.SystemAccountID {
			counterparty = counterpartyNumbers[tigerbeetle.UUIDString(counterpartyID)]
		}

		amountCents := t.Amount.BigInt().Int64()

		response = append(response, transactionResponse{
			ID:                        t.ID.String(),
			Type:                      tigerbeetle.TransferTypeName(t.Code),
			Direction:                 direction,
			Amount:                    float64(amountCents) / 100,
			CounterpartyAccountNumber: counterparty,
			Timestamp:                 time.Unix(0, int64(t.Timestamp)),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// resolveCounterpartyAccountNumbers batches a reverse lookup (TigerBeetle ID
// -> account_number) through PostgreSQL for every non-SYSTEM counterparty
// found in transfers, since a Transfer only carries account IDs.
func (s *Server) resolveCounterpartyAccountNumbers(ctx context.Context, ownAccountID tb.Uint128, transfers []tb.Transfer) (map[string]string, error) {
	seen := make(map[string]bool)
	ids := make([]string, 0, len(transfers))

	for _, t := range transfers {
		other := t.DebitAccountID
		if t.DebitAccountID == ownAccountID {
			other = t.CreditAccountID
		}
		if other == tigerbeetle.SystemAccountID {
			continue
		}

		uuidStr := tigerbeetle.UUIDString(other)
		if !seen[uuidStr] {
			seen[uuidStr] = true
			ids = append(ids, uuidStr)
		}
	}

	result := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	rows, err := s.DB.Query(
		ctx,
		`SELECT tigerbeetle_account_id, account_number FROM accounts WHERE tigerbeetle_account_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uuidStr, accountNumber string
		if err := rows.Scan(&uuidStr, &accountNumber); err != nil {
			return nil, err
		}
		result[uuidStr] = accountNumber
	}

	return result, rows.Err()
}

func parseLimit(raw string) int {
	if raw == "" {
		return defaultTransactionsLimit
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 || parsed > maxTransactionsLimit {
		return defaultTransactionsLimit
	}

	return parsed
}
