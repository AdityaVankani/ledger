package httpapi

import (
	"net/http"
	"sort"
)

type settlementSuggestion struct {
	Currency    string       `json:"currency"`
	From        userResponse `json:"from"`
	To          userResponse `json:"to"`
	AmountCents int64        `json:"amount_cents"`
}

type balanceParticipant struct {
	user      userResponse
	remaining int64
}

func (a *api) listSettlementSuggestions(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	balances, err := a.groupBalances(r.Context(), groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not calculate settlement suggestions")
		return
	}
	writeJSON(w, http.StatusOK, simplifyBalances(balances))
}

func simplifyBalances(balances []balanceResponse) []settlementSuggestion {
	suggestions := make([]settlementSuggestion, 0)
	for _, currencyBalance := range balances {
		creditors := make([]balanceParticipant, 0)
		debtors := make([]balanceParticipant, 0)
		for _, member := range currencyBalance.Members {
			switch {
			case member.BalanceCents > 0:
				creditors = append(creditors, balanceParticipant{user: member.User, remaining: member.BalanceCents})
			case member.BalanceCents < 0:
				debtors = append(debtors, balanceParticipant{user: member.User, remaining: -member.BalanceCents})
			}
		}

		sort.Slice(creditors, func(i, j int) bool {
			if creditors[i].remaining != creditors[j].remaining {
				return creditors[i].remaining > creditors[j].remaining
			}
			return creditors[i].user.ID < creditors[j].user.ID
		})
		sort.Slice(debtors, func(i, j int) bool {
			if debtors[i].remaining != debtors[j].remaining {
				return debtors[i].remaining > debtors[j].remaining
			}
			return debtors[i].user.ID < debtors[j].user.ID
		})

		creditorIndex, debtorIndex := 0, 0
		for creditorIndex < len(creditors) && debtorIndex < len(debtors) {
			amount := min(creditors[creditorIndex].remaining, debtors[debtorIndex].remaining)
			suggestions = append(suggestions, settlementSuggestion{
				Currency:    currencyBalance.Currency,
				From:        debtors[debtorIndex].user,
				To:          creditors[creditorIndex].user,
				AmountCents: amount,
			})
			creditors[creditorIndex].remaining -= amount
			debtors[debtorIndex].remaining -= amount
			if creditors[creditorIndex].remaining == 0 {
				creditorIndex++
			}
			if debtors[debtorIndex].remaining == 0 {
				debtorIndex++
			}
		}
	}
	return suggestions
}
