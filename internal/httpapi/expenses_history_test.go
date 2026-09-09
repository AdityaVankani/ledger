package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int
		wantOK     bool
		wantStatus int
	}{
		{name: "default", wantLimit: 50, wantOK: true},
		{name: "explicit", query: "limit=25", wantLimit: 25, wantOK: true},
		{name: "zero", query: "limit=0", wantOK: false, wantStatus: http.StatusBadRequest},
		{name: "over max", query: "limit=101", wantOK: false, wantStatus: http.StatusBadRequest},
		{name: "not numeric", query: "limit=many", wantOK: false, wantStatus: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/groups/example/expenses?"+test.query, nil)
			response := httptest.NewRecorder()
			limit, ok := parseLimit(response, request)
			if limit != test.wantLimit || ok != test.wantOK {
				t.Fatalf("parseLimit() = (%d, %t), want (%d, %t)", limit, ok, test.wantLimit, test.wantOK)
			}
			if !test.wantOK && response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}
