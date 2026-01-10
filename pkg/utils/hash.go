package utils

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func HashOrRead(password string) ([]byte, error) {
	if strings.HasPrefix(password, "$2a$") || strings.HasPrefix(password, "$2b$") || strings.HasPrefix(password, "$2y$") {
		return []byte(password), nil // already bcrypt
	}
	return bcrypt.GenerateFromPassword([]byte(password), 10)
}
