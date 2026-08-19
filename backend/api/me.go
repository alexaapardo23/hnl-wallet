package api

import (
	"net/http"
	"time"
)

type meResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
}

// meHandler returns the authenticated user's own profile.
func (s *Server) meHandler(w http.ResponseWriter, r *http.Request) {
	var resp meResponse

	err := s.DB.QueryRow(
		r.Context(),
		`SELECT id, email, full_name, created_at FROM users WHERE id = $1`,
		userIDFromContext(r),
	).Scan(&resp.ID, &resp.Email, &resp.FullName, &resp.CreatedAt)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
