package main

import "testing"

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
