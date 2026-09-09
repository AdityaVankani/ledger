//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestExpenseFlowIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(pool, time.Hour, "")

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	owner := registerIntegrationUser(t, router, "owner-"+suffix+"@example.test", "Owner")
	member := registerIntegrationUser(t, router, "member-"+suffix+"@example.test", "Member")

	groupRecorder := integrationRequest(t, router, http.MethodPost, "/v1/groups", owner.SessionToken, createGroupRequest{Name: "Integration group"})
	requireIntegrationStatus(t, groupRecorder, http.StatusCreated)
	var group groupResponse
	decodeIntegrationJSON(t, groupRecorder, &group)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM settlements WHERE group_id = $1`, group.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM expenses WHERE group_id = $1`, group.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM groups WHERE id = $1`, group.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1 OR id = $2`, owner.User.ID, member.User.ID)
	})

	memberRecorder := integrationRequest(t, router, http.MethodPost, "/v1/groups/"+group.ID+"/members", owner.SessionToken, addGroupMemberRequest{Email: member.User.Email})
	requireIntegrationStatus(t, memberRecorder, http.StatusCreated)

	expenseRecorder := integrationRequest(t, router, http.MethodPost, "/v1/groups/"+group.ID+"/expenses", owner.SessionToken, createExpenseRequest{
		Description:  "Dinner",
		AmountCents:  1200,
		Currency:     "INR",
		PaidByUserID: owner.User.ID,
		Splits: []expenseSplit{
			{UserID: owner.User.ID, AmountCents: 600},
			{UserID: member.User.ID, AmountCents: 600},
		},
	})
	requireIntegrationStatus(t, expenseRecorder, http.StatusCreated)

	balancesRecorder := integrationRequest(t, router, http.MethodGet, "/v1/groups/"+group.ID+"/balances", owner.SessionToken, nil)
	requireIntegrationStatus(t, balancesRecorder, http.StatusOK)

	settlementRecorder := integrationRequest(t, router, http.MethodPost, "/v1/groups/"+group.ID+"/settlements", member.SessionToken, createSettlementRequest{
		ReceivedByUserID: owner.User.ID,
		AmountCents:      600,
		Currency:         "INR",
	})
	requireIntegrationStatus(t, settlementRecorder, http.StatusCreated)
	var settlement settlementResponse
	decodeIntegrationJSON(t, settlementRecorder, &settlement)

	suggestionsRecorder := integrationRequest(t, router, http.MethodGet, "/v1/groups/"+group.ID+"/settlement-suggestions", owner.SessionToken, nil)
	requireIntegrationStatus(t, suggestionsRecorder, http.StatusOK)
	var suggestions []settlementSuggestion
	decodeIntegrationJSON(t, suggestionsRecorder, &suggestions)
	if len(suggestions) != 0 {
		t.Fatalf("suggestions = %#v, want no remaining debts", suggestions)
	}

	deleteSettlementRecorder := integrationRequest(t, router, http.MethodDelete, "/v1/groups/"+group.ID+"/settlements/"+settlement.ID, member.SessionToken, nil)
	requireIntegrationStatus(t, deleteSettlementRecorder, http.StatusNoContent)
	suggestionsRecorder = integrationRequest(t, router, http.MethodGet, "/v1/groups/"+group.ID+"/settlement-suggestions", owner.SessionToken, nil)
	requireIntegrationStatus(t, suggestionsRecorder, http.StatusOK)
	decodeIntegrationJSON(t, suggestionsRecorder, &suggestions)
	if len(suggestions) != 1 || suggestions[0].AmountCents != 600 {
		t.Fatalf("suggestions after deleting settlement = %#v, want one payment for 600", suggestions)
	}

	var expense expenseResponse
	decodeIntegrationJSON(t, expenseRecorder, &expense)
	deleteExpenseRecorder := integrationRequest(t, router, http.MethodDelete, "/v1/groups/"+group.ID+"/expenses/"+expense.ID, owner.SessionToken, nil)
	requireIntegrationStatus(t, deleteExpenseRecorder, http.StatusNoContent)
	suggestionsRecorder = integrationRequest(t, router, http.MethodGet, "/v1/groups/"+group.ID+"/settlement-suggestions", owner.SessionToken, nil)
	requireIntegrationStatus(t, suggestionsRecorder, http.StatusOK)
	decodeIntegrationJSON(t, suggestionsRecorder, &suggestions)
	if len(suggestions) != 0 {
		t.Fatalf("suggestions after deleting expense = %#v, want none", suggestions)
	}

	logoutRecorder := integrationRequest(t, router, http.MethodDelete, "/v1/auth/session", member.SessionToken, nil)
	requireIntegrationStatus(t, logoutRecorder, http.StatusNoContent)
	meRecorder := integrationRequest(t, router, http.MethodGet, "/v1/me", member.SessionToken, nil)
	requireIntegrationStatus(t, meRecorder, http.StatusUnauthorized)
}

func registerIntegrationUser(t *testing.T, router http.Handler, email, displayName string) authResponse {
	t.Helper()
	recorder := integrationRequest(t, router, http.MethodPost, "/v1/auth/register", "", registerRequest{
		Email: email, DisplayName: displayName, Password: "correct-horse-battery",
	})
	requireIntegrationStatus(t, recorder, http.StatusCreated)
	var response authResponse
	decodeIntegrationJSON(t, recorder, &response)
	return response
}

func integrationRequest(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var encodedBody []byte
	if body != nil {
		var err error
		encodedBody, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(encodedBody))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func requireIntegrationStatus(t *testing.T, recorder *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if recorder.Code != expected {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, expected, recorder.Body.String())
	}
}

func decodeIntegrationJSON(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(target); err != nil {
		t.Fatal(fmt.Errorf("decode response: %w", err))
	}
}
