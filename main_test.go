package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JB4779/chirpy/internal/auth"
	"github.com/google/uuid"
)

func TestCleanBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean text",
			input:    "Hello from Chirpy",
			expected: "Hello from Chirpy",
		},
		{
			name:     "kerfuffle",
			input:    "This is a kerfuffle",
			expected: "This is a ****",
		},
		{
			name:     "uppercase",
			input:    "This is a KERFUFFLE",
			expected: "This is a ****",
		},
		{
			name:     "mixed case",
			input:    "This is a ShArBeRt",
			expected: "This is a ****",
		},
		{
			name:     "multiple bad words",
			input:    "kerfuffle sharbert fornax",
			expected: "**** **** ****",
		},
		{
			name:     "punctuation not replaced",
			input:    "kerfuffle!",
			expected: "kerfuffle!",
		},
		{
			name:     "bad word surrounded by clean words",
			input:    "I think fornax is silly",
			expected: "I think **** is silly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := cleanBody(tt.input)

			if actual != tt.expected {
				t.Errorf(
					"cleanBody(%q) = %q, expected %q",
					tt.input,
					actual,
					tt.expected,
				)
			}
		})
	}
}

func TestRespondWithJSON(t *testing.T) {
	recorder := httptest.NewRecorder()

	payload := struct {
		Message string `json:"message"`
	}{
		Message: "hello",
	}

	respondWithJSON(
		recorder,
		http.StatusCreated,
		payload,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusCreated {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusCreated,
			result.StatusCode,
		)
	}

	contentType := result.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf(
			"expected Content-Type application/json, got %q",
			contentType,
		)
	}

	var response struct {
		Message string `json:"message"`
	}

	err := json.NewDecoder(result.Body).Decode(&response)
	if err != nil {
		t.Fatalf(
			"couldn't decode response: %v",
			err,
		)
	}

	if response.Message != "hello" {
		t.Errorf(
			"expected message %q, got %q",
			"hello",
			response.Message,
		)
	}
}

func TestRespondWithError(t *testing.T) {
	recorder := httptest.NewRecorder()

	respondWithError(
		recorder,
		http.StatusBadRequest,
		"Something went wrong",
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			result.StatusCode,
		)
	}

	var response struct {
		Error string `json:"error"`
	}

	err := json.NewDecoder(result.Body).Decode(&response)
	if err != nil {
		t.Fatalf(
			"couldn't decode response: %v",
			err,
		)
	}

	if response.Error != "Something went wrong" {
		t.Errorf(
			"expected error %q, got %q",
			"Something went wrong",
			response.Error,
		)
	}
}

func TestHandlerReadiness(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/healthz",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlerReadiness(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusOK,
			result.StatusCode,
		)
	}

	contentType := result.Header.Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf(
			"unexpected Content-Type: %q",
			contentType,
		)
	}
}

func TestUpdateUserMissingToken(t *testing.T) {
	cfg := &apiConfig{
		jwtSecret: "test-secret",
	}

	body := `{
		"email":"updated@example.com",
		"password":"newpassword"
	}`

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/users",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	cfg.handlerUpdateUser(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			result.StatusCode,
		)
	}
}

func TestUpdateUserInvalidToken(t *testing.T) {
	cfg := &apiConfig{
		jwtSecret: "test-secret",
	}

	body := `{
		"email":"updated@example.com",
		"password":"newpassword"
	}`

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/users",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	request.Header.Set(
		"Authorization",
		"Bearer this-is-not-a-valid-jwt",
	)

	recorder := httptest.NewRecorder()

	cfg.handlerUpdateUser(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			result.StatusCode,
		)
	}
}

func TestDeleteChirpMissingToken(t *testing.T) {
	cfg := &apiConfig{
		jwtSecret: "test-secret",
	}

	chirpID := uuid.New()

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/chirps/"+chirpID.String(),
		nil,
	)

	recorder := httptest.NewRecorder()

	cfg.handlerDeleteChirp(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			result.StatusCode,
		)
	}
}

func TestDeleteChirpInvalidToken(t *testing.T) {
	cfg := &apiConfig{
		jwtSecret: "test-secret",
	}

	chirpID := uuid.New()

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/chirps/"+chirpID.String(),
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer this-is-not-a-valid-jwt",
	)

	recorder := httptest.NewRecorder()

	cfg.handlerDeleteChirp(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			result.StatusCode,
		)
	}
}

func TestDeleteChirpInvalidID(t *testing.T) {
	const secret = "test-secret"

	cfg := &apiConfig{
		jwtSecret: secret,
	}

	userID := uuid.New()

	token, err := auth.MakeJWT(
		userID,
		secret,
		time.Hour,
	)
	if err != nil {
		t.Fatalf(
			"couldn't create test JWT: %v",
			err,
		)
	}

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/chirps/not-a-valid-uuid",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	request.SetPathValue(
		"chirpID",
		"not-a-valid-uuid",
	)

	recorder := httptest.NewRecorder()

	cfg.handlerDeleteChirp(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusNotFound {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusNotFound,
			result.StatusCode,
		)
	}
}

func TestPolkaWebhookIgnoresUnknownEvent(t *testing.T) {
	cfg := &apiConfig{}

	body := `{
		"event": "something.else",
		"data": {
			"user_id": "3311741c-680c-4546-99f3-fc9efac2036c"
		}
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/polka/webhooks",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	cfg.handlerPolkaWebhook(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusNoContent {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusNoContent,
			result.StatusCode,
		)
	}
}

func TestPolkaWebhookInvalidJSON(t *testing.T) {
	cfg := &apiConfig{}

	body := `{
		"event": "user.upgraded",
		"data":
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/polka/webhooks",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	recorder := httptest.NewRecorder()

	cfg.handlerPolkaWebhook(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			result.StatusCode,
		)
	}
}

func TestPolkaWebhookInvalidJSONReturnsError(t *testing.T) {
	cfg := &apiConfig{}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/polka/webhooks",
		strings.NewReader(`{"event":`),
	)

	recorder := httptest.NewRecorder()

	cfg.handlerPolkaWebhook(
		recorder,
		request,
	)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			result.StatusCode,
		)
	}

	var response struct {
		Error string `json:"error"`
	}

	err := json.NewDecoder(result.Body).Decode(&response)
	if err != nil {
		t.Fatalf(
			"couldn't decode error response: %v",
			err,
		)
	}

	if response.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestUserIncludesChirpyRed(t *testing.T) {
	user := User{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Email:       "test@example.com",
		IsChirpyRed: true,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf(
			"couldn't marshal user: %v",
			err,
		)
	}

	var result map[string]interface{}

	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf(
			"couldn't unmarshal user JSON: %v",
			err,
		)
	}

	value, exists := result["is_chirpy_red"]
	if !exists {
		t.Fatal(
			"expected user JSON to contain is_chirpy_red",
		)
	}

	if value != true {
		t.Errorf(
			"expected is_chirpy_red true, got %v",
			value,
		)
	}
}

func TestUserChirpyRedFalse(t *testing.T) {
	user := User{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Email:       "test@example.com",
		IsChirpyRed: false,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf(
			"couldn't marshal user: %v",
			err,
		)
	}

	var result struct {
		IsChirpyRed bool `json:"is_chirpy_red"`
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf(
			"couldn't unmarshal user JSON: %v",
			err,
		)
	}

	if result.IsChirpyRed {
		t.Error(
			"expected is_chirpy_red to be false",
		)
	}
}
