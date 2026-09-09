package httpapi

import (
	"context"
	"net/http"

	"github.com/adityavankani/expense-app/internal/database/sqlc"
)

type balanceResponse struct {
	Currency string          `json:"currency"`
	Members  []memberBalance `json:"members"`
}

type memberBalance struct {
	User         userResponse `json:"user"`
	BalanceCents int64        `json:"balance_cents"`
}

func (a *api) listBalances(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	balances, err := a.groupBalances(r.Context(), groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not calculate balances")
		return
	}
	writeJSON(w, http.StatusOK, balances)
}

func (a *api) groupBalances(ctx context.Context, groupID string) ([]balanceResponse, error) {
	rows, err := a.queries.ListBalances(ctx, sqlc.ListBalancesParams{GroupID: groupID})
	if err != nil {
		return nil, err
	}
	balances := make([]balanceResponse, 0)
	byCurrency := make(map[string]int)
	for _, row := range rows {
		currency := row.Currency
		member := memberBalance{User: userResponse{ID: row.UserID, Email: row.Email, DisplayName: row.DisplayName, UPIID: row.UPIID}, BalanceCents: row.BalanceCents}
		index, found := byCurrency[currency]
		if !found {
			index = len(balances)
			byCurrency[currency] = index
			balances = append(balances, balanceResponse{Currency: currency, Members: make([]memberBalance, 0)})
		}
		balances[index].Members = append(balances[index].Members, member)
	}
	return balances, nil
}
