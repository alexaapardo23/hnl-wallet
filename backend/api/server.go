package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	tb "github.com/tigerbeetle/tigerbeetle-go"
)

// Server holds the dependencies every handler needs.
type Server struct {
	DB        *pgxpool.Pool
	TB        tb.Client
	JWTSecret []byte
}

func NewRouter(s *Server) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)

	mux.HandleFunc("POST /auth/register", s.registerHandler)
	mux.HandleFunc("POST /auth/login", s.loginHandler)

	mux.Handle("GET /me", s.requireAuth(http.HandlerFunc(s.meHandler)))

	mux.Handle("POST /accounts", s.requireAuth(http.HandlerFunc(s.createAccountHandler)))
	mux.Handle("GET /accounts", s.requireAuth(http.HandlerFunc(s.listAccountsHandler)))
	mux.Handle("GET /accounts/{account_number}", s.requireAuth(http.HandlerFunc(s.accountDetailHandler)))
	mux.Handle("GET /accounts/{account_number}/balance", s.requireAuth(http.HandlerFunc(s.accountBalanceHandler)))
	mux.Handle("GET /accounts/{account_number}/transactions", s.requireAuth(http.HandlerFunc(s.accountTransactionsHandler)))

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
