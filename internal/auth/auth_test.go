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
			"expected token %q, got %q",
			"abc123",
			token,
		)
	}
}

func TestGetBearerTokenMissingHeader(t *testing.T) {
	headers := http.Header{}

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Error(
			"expected missing Authorization header to return an error",
		)
	}
}

func TestGetBearerTokenWrongScheme(t *testing.T) {
	headers := http.Header{}
	headers.Set(
		"Authorization",
		"Basic abc123",
	)

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Error(
			"expected invalid authorization scheme to return an error",
		)
	}
}

func TestGetBearerTokenEmptyToken(t *testing.T) {
	headers := http.Header{}
	headers.Set(
		"Authorization",
		"Bearer ",
	)

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Error(
			"expected empty bearer token to return an error",
		)
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	password := "my-secret-password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf(
			"HashPassword returned an error: %v",
			err,
		)
	}

	if hash == password {
		t.Error(
			"hashed password should not equal plaintext password",
		)
	}

	matches, err := CheckPasswordHash(
		password,
		hash,
	)
	if err != nil {
		t.Fatalf(
			"CheckPasswordHash returned an error: %v",
			err,
		)
	}

	if !matches {
		t.Error(
			"expected correct password to match hash",
		)
	}
}

func TestCheckPasswordHashWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf(
			"HashPassword returned an error: %v",
			err,
		)
	}

	matches, err := CheckPasswordHash(
		"wrong-password",
		hash,
	)
	if err != nil {
		t.Fatalf(
			"CheckPasswordHash returned an error: %v",
			err,
		)
	}

	if matches {
		t.Error(
			"expected wrong password not to match hash",
		)
	}
}

func TestMakeRefreshToken(t *testing.T) {
	token1 := MakeRefreshToken()
	token2 := MakeRefreshToken()

	if len(token1) != 64 {
		t.Errorf(
			"expected refresh token length 64, got %d",
			len(token1),
		)
	}

	if token1 == token2 {
		t.Error(
			"expected refresh tokens to be unique",
		)
	}

	decoded, err := hex.DecodeString(token1)
	if err != nil {
		t.Fatalf(
			"refresh token is not valid hex: %v",
			err,
		)
	}

	if len(decoded) != 32 {
		t.Errorf(
			"expected 32 random bytes, got %d",
			len(decoded),
		)
	}
}

func TestGetAPIKey(t *testing.T) {
	headers := http.Header{}
	headers.Set(
		"Authorization",
		"ApiKey abc123",
	)

	key, err := GetAPIKey(headers)
	if err != nil {
		t.Fatalf(
			"GetAPIKey returned an error: %v",
			err,
		)
	}

	if key != "abc123" {
		t.Errorf(
			"expected key %q, got %q",
			"abc123",
			key,
		)
	}
}

func TestGetAPIKeyMissingHeader(t *testing.T) {
	headers := http.Header{}

	_, err := GetAPIKey(headers)
	if err == nil {
		t.Error(
			"expected missing Authorization header to return an error",
		)
	}
}

func TestGetAPIKeyWrongScheme(t *testing.T) {
	headers := http.Header{}
	headers.Set(
		"Authorization",
		"Bearer abc123",
	)

	_, err := GetAPIKey(headers)
	if err == nil {
		t.Error(
			"expected invalid authorization scheme to return an error",
		)
	}
}

func TestGetAPIKeyEmptyKey(t *testing.T) {
	headers := http.Header{}
	headers.Set(
		"Authorization",
		"ApiKey ",
	)

	_, err := GetAPIKey(headers)
	if err == nil {
		t.Error(
			"expected empty API key to return an error",
		)
	}
}
