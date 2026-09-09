package httpapi

import "testing"

func TestSimplifyBalancesForSharedExpense(t *testing.T) {
	balances := []balanceResponse{{
		Currency: "INR",
		Members: []memberBalance{
			{User: userResponse{ID: "adi", DisplayName: "Adi"}, BalanceCents: 30},
			{User: userResponse{ID: "aditya", DisplayName: "Aditya"}, BalanceCents: -30},
		},
	}}

	suggestions := simplifyBalances(balances)
	if len(suggestions) != 1 {
		t.Fatalf("suggestion count = %d, want 1", len(suggestions))
	}
	if suggestions[0].From.ID != "aditya" || suggestions[0].To.ID != "adi" || suggestions[0].AmountCents != 30 {
		t.Fatalf("suggestion = %#v, want Aditya -> Adi for 30 cents", suggestions[0])
	}
}
