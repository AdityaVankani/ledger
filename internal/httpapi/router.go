package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/adityavankani/expense-app/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool, sessionTTL time.Duration, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()
	api := NewAPI(pool, sessionTTL)
	queries := sqlc.New(pool)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := queries.DatabaseNow(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /v1/auth/register", api.register)
	mux.HandleFunc("POST /v1/auth/login", api.login)
	mux.Handle("GET /v1/me", api.requireUser(http.HandlerFunc(api.me)))
	mux.Handle("PATCH /v1/me", api.requireUser(http.HandlerFunc(api.updateProfile)))
	mux.Handle("DELETE /v1/auth/session", api.requireUser(http.HandlerFunc(api.logout)))
	mux.Handle("POST /v1/groups", api.requireUser(http.HandlerFunc(api.createGroup)))
	mux.Handle("DELETE /v1/groups/{groupID}", api.requireUser(http.HandlerFunc(api.deleteGroup)))
	mux.Handle("GET /v1/groups", api.requireUser(http.HandlerFunc(api.listGroups)))
	mux.Handle("GET /v1/groups/{groupID}/members", api.requireUser(http.HandlerFunc(api.listGroupMembers)))
	mux.Handle("POST /v1/groups/{groupID}/members", api.requireUser(http.HandlerFunc(api.addGroupMember)))
	mux.Handle("POST /v1/groups/{groupID}/expenses", api.requireUser(http.HandlerFunc(api.createExpense)))
	mux.Handle("GET /v1/groups/{groupID}/expenses", api.requireUser(http.HandlerFunc(api.listExpenses)))
	mux.Handle("DELETE /v1/groups/{groupID}/expenses/{expenseID}", api.requireUser(http.HandlerFunc(api.deleteExpense)))
	mux.Handle("PATCH /v1/groups/{groupID}/expenses/{expenseID}", api.requireUser(http.HandlerFunc(api.updateExpense)))
	mux.Handle("GET /v1/groups/{groupID}/balances", api.requireUser(http.HandlerFunc(api.listBalances)))
	mux.Handle("POST /v1/groups/{groupID}/settlements", api.requireUser(http.HandlerFunc(api.createSettlement)))
	mux.Handle("DELETE /v1/groups/{groupID}/settlements/{settlementID}", api.requireUser(http.HandlerFunc(api.deleteSettlement)))
	mux.Handle("GET /v1/groups/{groupID}/settlement-suggestions", api.requireUser(http.HandlerFunc(api.listSettlementSuggestions)))
	return withCORS(mux, allowedOrigin)
}

func withCORS(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigin != "" && origin != "" && strings.TrimSpace(origin) == strings.TrimSpace(allowedOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
