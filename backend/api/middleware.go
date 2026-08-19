package api

import (
	"context"
	"net/http"
	"strings"

	"hnl-wallet/backend/auth"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

// requireAuth validates the "Authorization: Bearer <token>" header and
// injects the authenticated user's ID into the request context.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		userID, err := auth.ParseToken(s.JWTSecret, token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userIDFromContext(r *http.Request) string {
	userID, _ := r.Context().Value(userIDContextKey).(string)
	return userID
}
