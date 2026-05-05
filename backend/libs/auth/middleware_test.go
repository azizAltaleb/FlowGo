package auth

import "testing"

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		expected      string
	}{
		{name: "valid token", authorization: "Bearer abc123", expected: "abc123"},
		{name: "case-insensitive scheme", authorization: "bearer xyz", expected: "xyz"},
		{name: "missing token", authorization: "Bearer ", expected: ""},
		{name: "invalid scheme", authorization: "Basic x", expected: ""},
		{name: "empty", authorization: "", expected: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := extractBearerToken(test.authorization); got != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}
}
