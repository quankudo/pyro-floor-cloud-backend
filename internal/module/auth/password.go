package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const MaxPasswordBytes = 72

var (
	ErrEmptyPassword       = errors.New("password must not be empty")
	ErrPasswordTooLong     = errors.New("password exceeds bcrypt's 72-byte limit")
	ErrInvalidPasswordHash = errors.New("invalid password hash")
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", fmt.Errorf("generate bcrypt hash: %w", err)
	}

	return string(hash), nil
}

func VerifyPassword(password, passwordHash string) (bool, error) {
	if password == "" || len(password) > MaxPasswordBytes {
		return false, nil
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		return false, fmt.Errorf("%w: %v", ErrInvalidPasswordHash, err)
	}
}
