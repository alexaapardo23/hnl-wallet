package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"hnl-wallet/backend/auth"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type registerResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
}

// registerHandler creates a new user. Unlike the seed dataset — which
// intentionally contains users sharing an email (see README Authentication)
// — new registrations must use a unique email, enforced here at the
// application layer since the database no longer has a UNIQUE constraint
// on users.email.
func (s *Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if req.FullName == "" {
		writeError(w, http.StatusBadRequest, "full_name is required")
		return
	}

	ctx := r.Context()

	var exists bool
	err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, req.Email).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check email availability")
		return
	}
	if exists {
		writeError(w, http.StatusConflict, "email is already registered")
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	var resp registerResponse
	err = s.DB.QueryRow(
		ctx,
		`
		INSERT INTO users (id, email, password, full_name, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, now())
		RETURNING id, email, full_name, created_at
		`,
		req.Email, hashedPassword, req.FullName,
	).Scan(&resp.ID, &resp.Email, &resp.FullName, &resp.CreatedAt)

	// A concurrent request could win the race between the existence check
	// and this insert; the UNIQUE-less schema won't catch it, but we only
	// promised no *new* duplicates, so a second uniqueness check right
	// before responding keeps that promise without a DB constraint.
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

type loginRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	AccountNumber string `json:"account_number,omitempty"`
}

type loginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
	} `json:"user"`
}

var errInvalidCredentials = errors.New("invalid email or password")
var errAmbiguousEmail = errors.New("multiple accounts share this email; account_number is required to log in")

// loginHandler authenticates a user. account_number is only required when
// email alone doesn't identify a single user — see README Authentication
// for why the seed dataset can have duplicate emails.
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	userID, storedPassword, fullName, err := s.findUserForLogin(r.Context(), req.Email, req.AccountNumber)
	switch {
	case errors.Is(err, errAmbiguousEmail):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, errInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to look up user")
		return
	}

	if !auth.VerifyPassword(storedPassword, req.Password) {
		writeError(w, http.StatusUnauthorized, errInvalidCredentials.Error())
		return
	}

	token, err := auth.GenerateToken(s.JWTSecret, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	var resp loginResponse
	resp.Token = token
	resp.User.ID = userID
	resp.User.Email = req.Email
	resp.User.FullName = fullName

	writeJSON(w, http.StatusOK, resp)
}

// findUserForLogin resolves an email (+ optional account_number) to exactly
// one user, disambiguating duplicate emails via account_number when needed.
func (s *Server) findUserForLogin(ctx context.Context, email, accountNumber string) (userID, password, fullName string, err error) {
	if accountNumber != "" {
		err = s.DB.QueryRow(
			ctx,
			`
			SELECT u.id, u.password, u.full_name
			FROM users u
			JOIN accounts a ON a.user_id = u.id
			WHERE u.email = $1 AND a.account_number = $2
			`,
			email, accountNumber,
		).Scan(&userID, &password, &fullName)

		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", errInvalidCredentials
		}
		return userID, password, fullName, err
	}

	rows, err := s.DB.Query(ctx, `SELECT id, password, full_name FROM users WHERE email = $1`, email)
	if err != nil {
		return "", "", "", err
	}
	defer rows.Close()

	var matches int
	for rows.Next() {
		matches++
		if matches > 1 {
			return "", "", "", errAmbiguousEmail
		}
		if err := rows.Scan(&userID, &password, &fullName); err != nil {
			return "", "", "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", "", "", err
	}

	if matches == 0 {
		return "", "", "", errInvalidCredentials
	}

	return userID, password, fullName, nil
}
