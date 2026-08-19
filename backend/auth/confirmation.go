package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// confirmationTTL is short on purpose: a confirmation token represents a
// specific pending money movement (exact tool + arguments), not a session —
// it shouldn't stay valid long enough to be replayed well after the user
// saw the proposal.
const confirmationTTL = 5 * time.Minute

// ConfirmationClaims binds a proposed financial action to the user who
// triggered it. Arguments is the exact JSON the MCP tool will be called
// with on confirmation — signing it means POST /chat/confirm executes
// precisely what was proposed, with no way for a client to alter the
// amount or accounts between proposal and execution.
type ConfirmationClaims struct {
	UserID    string `json:"user_id"`
	Tool      string `json:"tool"`
	Arguments string `json:"arguments"`
	jwt.RegisteredClaims
}

// GenerateConfirmationToken signs a pending financial action, valid for
// confirmationTTL.
func GenerateConfirmationToken(secret []byte, userID, tool, argumentsJSON string) (string, error) {
	claims := ConfirmationClaims{
		UserID:    userID,
		Tool:      tool,
		Arguments: argumentsJSON,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(confirmationTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(secret)
}

// ParseConfirmationToken validates a confirmation token and returns the
// action it was issued for.
func ParseConfirmationToken(secret []byte, tokenString string) (*ConfirmationClaims, error) {
	claims := &ConfirmationClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
