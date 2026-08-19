// Command mcp-server exposes read-only HNL Wallet tools over the Model
// Context Protocol (Streamable HTTP transport), so an LLM (via OpenRouter)
// can answer questions like "¿Cuánto dinero tengo?" by calling them.
//
// This server holds no business logic and never touches PostgreSQL or
// TigerBeetle directly — every tool is a thin wrapper that calls the
// existing HNL Wallet API (see README Financial Data Model: it is
// already the single authority for ownership checks and balances), using
// the caller's own bearer token. That keeps this process authorization-free
// by construction: it can only ever see what the forwarded token allows.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type authKey struct{}

// authFromRequest carries the incoming Authorization header into the tool
// handler's context, whatever it is — this server never inspects or
// validates it; the HNL Wallet API does that.
func authFromRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, authKey{}, r.Header.Get("Authorization"))
}

func bearerTokenFromContext(ctx context.Context) (string, error) {
	token, _ := ctx.Value(authKey{}).(string)
	if token == "" {
		return "", fmt.Errorf("missing Authorization header")
	}
	return token, nil
}

var apiBaseURL string

// callAPI makes an authenticated GET against the HNL Wallet API, forwarding
// the caller's token unchanged.
func callAPI(ctx context.Context, path, bearerToken string) ([]byte, int, error) {
	return doAPI(ctx, http.MethodGet, path, nil, bearerToken)
}

// postAPI makes an authenticated POST with a JSON body against the HNL
// Wallet API, forwarding the caller's token unchanged.
//
// This is the entire enforcement boundary for deposit/withdraw/transfer:
// this server never checks whose account_number/from_account it was given —
// it forwards it, and the HNL Wallet API decides, from the token alone,
// whether the account actually belongs to the caller (see
// api.lookupOwnedAccount). A model that hallucinates or is tricked into
// requesting someone else's account_number gets exactly the same 404 a
// direct REST call would.
func postAPI(ctx context.Context, path string, body []byte, bearerToken string) ([]byte, int, error) {
	return doAPI(ctx, http.MethodPost, path, bytes.NewReader(body), bearerToken)
}

func doAPI(ctx context.Context, method, path string, body io.Reader, bearerToken string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, apiBaseURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", bearerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, err
}

type accountSummary struct {
	AccountNumber string  `json:"account_number"`
	AccountType   string  `json:"account_type"`
	Currency      string  `json:"currency"`
	Balance       float64 `json:"balance"`
}

// handleGetAccounts lists the caller's accounts together with each one's
// live balance, so a single call can answer "how much money do I have".
func handleGetAccounts(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token, err := bearerTokenFromContext(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	body, status, err := callAPI(ctx, "/accounts", token)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to reach HNL Wallet API", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultErrorf("HNL Wallet API returned %d: %s", status, string(body)), nil
	}

	var basics []struct {
		AccountNumber string `json:"account_number"`
		AccountType   string `json:"account_type"`
		Currency      string `json:"currency"`
	}
	if err := json.Unmarshal(body, &basics); err != nil {
		return mcp.NewToolResultErrorFromErr("failed to parse accounts response", err), nil
	}

	// One extra call per account for its live balance — GET /accounts
	// intentionally doesn't include it (see README API Endpoints), and a
	// user has at most a handful of accounts, so this stays cheap.
	summaries := make([]accountSummary, 0, len(basics))
	for _, a := range basics {
		detailBody, detailStatus, err := callAPI(ctx, "/accounts/"+a.AccountNumber, token)
		if err != nil || detailStatus != http.StatusOK {
			continue
		}

		var detail struct {
			Balance float64 `json:"balance"`
		}
		if err := json.Unmarshal(detailBody, &detail); err != nil {
			continue
		}

		summaries = append(summaries, accountSummary{
			AccountNumber: a.AccountNumber,
			AccountType:   a.AccountType,
			Currency:      a.Currency,
			Balance:       detail.Balance,
		})
	}

	out, err := json.Marshal(summaries)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to encode result", err), nil
	}

	return mcp.NewToolResultText(string(out)), nil
}

// handleGetBalance returns a single account's live balance.
func handleGetBalance(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token, err := bearerTokenFromContext(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountNumber, err := request.RequireString("account_number")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	body, status, err := callAPI(ctx, "/accounts/"+accountNumber+"/balance", token)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to reach HNL Wallet API", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultErrorf("HNL Wallet API returned %d: %s", status, string(body)), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}

// handleDeposit, handleWithdraw, and handleTransfer are pure wrappers around
// their REST equivalents (see README Financial Operations) — same as every
// read tool, this server does no validation of its own. Ownership,
// amount > 0, and sufficient-funds checks all happen exactly once, in the
// HNL Wallet API, whether the caller is a REST client or this MCP server.
//
// None of these three execute anything by themselves: POST /chat never
// calls them directly on the model's first tool call — it always routes
// through a signed confirmation token first (see backend/api/chat.go and
// README Financial Actions Require Confirmation). These handlers exist so
// that *whatever already-confirmed* action gets executed still goes
// through the same MCP -> Go API path as every read tool, rather than a
// separate one.
func handleDeposit(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token, err := bearerTokenFromContext(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountNumber, err := request.RequireString("account_number")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	amount, err := request.RequireFloat("amount")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reqBody, err := json.Marshal(map[string]float64{"amount": amount})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to encode request", err), nil
	}

	body, status, err := postAPI(ctx, "/accounts/"+accountNumber+"/deposit", reqBody, token)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to reach HNL Wallet API", err), nil
	}
	if status != http.StatusCreated {
		return mcp.NewToolResultErrorf("HNL Wallet API returned %d: %s", status, string(body)), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}

func handleWithdraw(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token, err := bearerTokenFromContext(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	accountNumber, err := request.RequireString("account_number")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	amount, err := request.RequireFloat("amount")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reqBody, err := json.Marshal(map[string]float64{"amount": amount})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to encode request", err), nil
	}

	body, status, err := postAPI(ctx, "/accounts/"+accountNumber+"/withdraw", reqBody, token)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to reach HNL Wallet API", err), nil
	}
	if status != http.StatusCreated {
		return mcp.NewToolResultErrorf("HNL Wallet API returned %d: %s", status, string(body)), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}

func handleTransfer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token, err := bearerTokenFromContext(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	fromAccount, err := request.RequireString("from_account")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	toAccount, err := request.RequireString("to_account")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	amount, err := request.RequireFloat("amount")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reqBody, err := json.Marshal(map[string]any{
		"from_account": fromAccount,
		"to_account":   toAccount,
		"amount":       amount,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to encode request", err), nil
	}

	body, status, err := postAPI(ctx, "/transfers", reqBody, token)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to reach HNL Wallet API", err), nil
	}
	if status != http.StatusCreated {
		return mcp.NewToolResultErrorf("HNL Wallet API returned %d: %s", status, string(body)), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}

const defaultHistoryLimit = 10

type transactionEntry struct {
	AccountNumber             string  `json:"account_number"`
	ID                        string  `json:"id"`
	Type                      string  `json:"type"`
	Direction                 string  `json:"direction"`
	Amount                    float64 `json:"amount"`
	CounterpartyAccountNumber string  `json:"counterparty_account_number"`
	Timestamp                 string  `json:"timestamp"`
}

// handleGetTransactionHistory returns the caller's most recent transactions.
// With account_number, it's scoped to that one account (wrapping
// GET /accounts/{account_number}/transactions directly); without it, it
// fetches every account the caller owns and merges their histories — so
// "¿Cuáles fueron mis últimas transacciones?" doesn't require the model to
// already know an account_number.
func handleGetTransactionHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	token, err := bearerTokenFromContext(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	limit := request.GetInt("limit", defaultHistoryLimit)
	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	var accountNumbers []string
	if accountNumber := request.GetString("account_number", ""); accountNumber != "" {
		accountNumbers = []string{accountNumber}
	} else {
		body, status, err := callAPI(ctx, "/accounts", token)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to reach HNL Wallet API", err), nil
		}
		if status != http.StatusOK {
			return mcp.NewToolResultErrorf("HNL Wallet API returned %d: %s", status, string(body)), nil
		}

		var basics []struct {
			AccountNumber string `json:"account_number"`
		}
		if err := json.Unmarshal(body, &basics); err != nil {
			return mcp.NewToolResultErrorFromErr("failed to parse accounts response", err), nil
		}
		for _, a := range basics {
			accountNumbers = append(accountNumbers, a.AccountNumber)
		}
	}

	entries := make([]transactionEntry, 0, limit)
	for _, accountNumber := range accountNumbers {
		path := fmt.Sprintf("/accounts/%s/transactions?limit=%d", accountNumber, limit)

		body, status, err := callAPI(ctx, path, token)
		if err != nil || status != http.StatusOK {
			// Best-effort across accounts, same as get_accounts: one
			// account's history failing shouldn't fail the whole tool call
			// when there's more than one account to look at. If the caller
			// asked for a specific account_number, this is the only
			// iteration, so its error is the only thing that shows up —
			// still as an empty result rather than an explicit error,
			// matching the plural-account case.
			continue
		}

		var txs []struct {
			ID                        string  `json:"id"`
			Type                      string  `json:"type"`
			Direction                 string  `json:"direction"`
			Amount                    float64 `json:"amount"`
			CounterpartyAccountNumber string  `json:"counterparty_account_number"`
			Timestamp                 string  `json:"timestamp"`
		}
		if err := json.Unmarshal(body, &txs); err != nil {
			continue
		}

		for _, t := range txs {
			entries = append(entries, transactionEntry{
				AccountNumber:             accountNumber,
				ID:                        t.ID,
				Type:                      t.Type,
				Direction:                 t.Direction,
				Amount:                    t.Amount,
				CounterpartyAccountNumber: t.CounterpartyAccountNumber,
				Timestamp:                 t.Timestamp,
			})
		}
	}

	// Each account's transactions arrive newest-first already; merging
	// several accounts requires re-sorting the combined set the same way.
	// RFC3339 timestamps (as returned by GET .../transactions) sort
	// correctly as plain strings.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp > entries[j].Timestamp })
	if len(entries) > limit {
		entries = entries[:limit]
	}

	out, err := json.Marshal(entries)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to encode result", err), nil
	}

	return mcp.NewToolResultText(string(out)), nil
}

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("warning: could not load .env: %v", err)
	}

	apiBaseURL = os.Getenv("HNL_API_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080"
	}

	port := os.Getenv("MCP_SERVER_PORT")
	if port == "" {
		port = "8081"
	}

	mcpServer := server.NewMCPServer(
		"hnl-wallet-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Read-only tools only — no way to move money through MCP.
	mcpServer.AddTool(mcp.NewTool("get_accounts",
		mcp.WithDescription("List the authenticated user's own wallet accounts, each with its live balance (source of truth: TigerBeetle). Use this to answer questions about how much money the user has, in total or per account."),
	), handleGetAccounts)

	mcpServer.AddTool(mcp.NewTool("get_balance",
		mcp.WithDescription("Get the live balance of one specific account owned by the authenticated user."),
		mcp.WithString("account_number",
			mcp.Description("The account_number to check, e.g. 4001-1234-5678-0001"),
			mcp.Required(),
		),
	), handleGetBalance)

	mcpServer.AddTool(mcp.NewTool("get_transaction_history",
		mcp.WithDescription("List the authenticated user's most recent transactions (deposits, withdrawals, and transfers), newest first. Omit account_number to see recent activity across every account the user owns."),
		mcp.WithString("account_number",
			mcp.Description("Optional: limit to one specific account, e.g. 4001-1234-5678-0001. If omitted, merges recent transactions from all of the user's accounts."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of transactions to return (default 10)."),
		),
	), handleGetTransactionHistory)

	// Financial actions. These move real money, so POST /chat never invokes
	// them on a model's first request — it always requires a signed
	// confirmation token first (see README Financial Actions Require
	// Confirmation). That gate lives in the chat orchestrator, not here:
	// these tools stay pure wrappers, like every read tool above, so the
	// only thing standing between "the model asked for this" and "this
	// actually happened to real money" is the confirmation step plus the
	// HNL Wallet API's own ownership/funds checks — not this server.
	mcpServer.AddTool(mcp.NewTool("deposit",
		mcp.WithDescription("Credit money into one of the authenticated user's own accounts, from the external funding source. Only ever called after the user has explicitly confirmed the exact amount and account in chat."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("account_number",
			mcp.Description("The account to credit, e.g. 4001-1234-5678-0001. Must belong to the authenticated user."),
			mcp.Required(),
		),
		mcp.WithNumber("amount",
			mcp.Description("Amount in USD, must be greater than 0."),
			mcp.Required(),
		),
	), handleDeposit)

	mcpServer.AddTool(mcp.NewTool("withdraw",
		mcp.WithDescription("Debit money out of one of the authenticated user's own accounts, to the external funding source. Only ever called after the user has explicitly confirmed the exact amount and account in chat."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("account_number",
			mcp.Description("The account to debit, e.g. 4001-1234-5678-0001. Must belong to the authenticated user."),
			mcp.Required(),
		),
		mcp.WithNumber("amount",
			mcp.Description("Amount in USD, must be greater than 0 and not exceed the account's current balance."),
			mcp.Required(),
		),
	), handleWithdraw)

	mcpServer.AddTool(mcp.NewTool("transfer",
		mcp.WithDescription("Move money from one of the authenticated user's own accounts to any account (their own or someone else's). Only ever called after the user has explicitly confirmed the exact amount and accounts in chat."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("from_account",
			mcp.Description("The source account, e.g. 4001-1234-5678-0001. Must belong to the authenticated user — the HNL Wallet API rejects it otherwise, regardless of what is passed here."),
			mcp.Required(),
		),
		mcp.WithString("to_account",
			mcp.Description("The destination account. Does not have to belong to the authenticated user."),
			mcp.Required(),
		),
		mcp.WithNumber("amount",
			mcp.Description("Amount in USD, must be greater than 0 and not exceed from_account's current balance."),
			mcp.Required(),
		),
	), handleTransfer)

	httpServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithHTTPContextFunc(authFromRequest),
	)

	log.Printf("MCP server listening on :%s (HNL Wallet API at %s)", port, apiBaseURL)
	if err := httpServer.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}
