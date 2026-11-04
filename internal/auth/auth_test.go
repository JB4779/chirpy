package auth

import (
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := MakeJWT(
		userID,
		secret,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("MakeJWT returned an error: %v", err)
	}

	returnedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT returned an error: %v", err)
	}

	if returnedID != userID {
		t.Errorf(
			"expected user ID %s, got %s",
			userID,
			returnedID,
		)
	}
}

func TestExpiredJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := MakeJWT(
		userID,
		secret,
		-time.Hour,
	)
	if err != nil {
		t.Fatalf("MakeJWT returned an error: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Error("expected expired JWT to return an error")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(
		userID,
		"correct-secret",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("MakeJWT returned an error: %v", err)
	}

	_, err = ValidateJWT(
		token,
		"wrong-secret",
	)
	if err == nil {
		t.Error("expected wrong secret to return an error")
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}

	headers.Set(
		"Authorization",
		"Bearer abc123",
	)

	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf(
			"GetBearerToken returned an error: %v",
			err,
		)
	}

	if token != "abc123" {
		t.Errorf(
			"expected token %q
