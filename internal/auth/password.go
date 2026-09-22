package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordLength = 8
	// bcrypt only uses the first 72 bytes of the password, so longer ones are rejected.
	MaxPasswordLength = 72
)

var ErrWeakPassword = errors.New("password must have between 8 and 72 characters")

// HashPassword returns the bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return "", ErrWeakPassword
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword compares the password with the hash in constant time.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash is used when the email doesn't exist, so login takes the same
// time in both cases and doesn't reveal which emails are registered.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("pharmacode-dummy-password"), bcrypt.DefaultCost)
