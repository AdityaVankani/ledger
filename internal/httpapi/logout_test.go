package httpapi

import (
	"context"
	"crypto/sha256"
	"net/http/httptest"
	"testing"
)

func TestSessionTokenHashFromContext(t *testing.T) {
	expected := sha256.Sum256([]byte("session-token"))
	request := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(request.Context(), sessionTokenHashContextKey, expected)
	if actual := sessionTokenHashFromContext(request.WithContext(ctx)); actual != expected {
		t.Fatalf("sessionTokenHashFromContext() = %x, want %x", actual, expected)
	}
}
