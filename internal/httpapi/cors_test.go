package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithCORS_AllowsCommaSeparatedOrigins(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := withCORS(next, "http://localhost:5173, https://ledgersplit.vercel.app")
	req := httptest.NewRequest(http.MethodOptions, "/v1/auth/register", nil)
	req.Header.Set("Origin", "https://ledgersplit.vercel.app")
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://ledgersplit.vercel.app" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "https://ledgersplit.vercel.app")
	}
}
