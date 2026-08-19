package auth

import "golang.org/x/crypto/bcrypt"

// bcryptPrefixes are the hash identifiers bcrypt writes at the start of
// every hash it produces. Anything else is treated as a legacy plaintext
// password — the seed dataset (data/data.json) stores passwords in plain
// text (e.g. "Isabel2024!"), and those rows are intentionally left
// untouched rather than migrated, so login has to support both forms.
var bcryptPrefixes = []string{"$2a$", "$2b$", "$2y$"}

// HashPassword hashes a plaintext password for storage. Always used for
// newly registered users.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// VerifyPassword checks a plaintext password against a stored value that
// may be either a bcrypt hash (new users) or legacy plaintext (seed users).
func VerifyPassword(stored, candidate string) bool {
	for _, prefix := range bcryptPrefixes {
		if len(stored) >= len(prefix) && stored[:len(prefix)] == prefix {
			return bcrypt.CompareHashAndPassword([]byte(stored), []byte(candidate)) == nil
		}
	}

	return stored == candidate
}
