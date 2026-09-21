package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordLength = 8
	// bcrypt só considera os primeiros 72 bytes da senha; acima disso recusamos.
	MaxPasswordLength = 72
)

var ErrWeakPassword = errors.New("password must have between 8 and 72 characters")

// HashPassword gera o hash bcrypt da senha.
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

// CheckPassword compara a senha com o hash em tempo constante.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash é usado quando o email não existe, para o login levar o mesmo
// tempo nos dois casos e não revelar quais emails estão cadastrados.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("pharmacode-dummy-password"), bcrypt.DefaultCost)
