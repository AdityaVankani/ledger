package httpapi

import "testing"

func TestSimplifyBalances(t *testing.T) {
	balances := []balanceResponse{
		{
			Currency: "INR",
			Members: []memberBalance{
				{User: userResponse{ID: "a", DisplayName: "A"}, BalanceCents: 900},
				{User: userResponse{ID: "b", DisplayName: "B"}, BalanceCents: -600},
				{User: userResponse{ID: "c", DisplayName: "C"}, BalanceCents: -300},
			},
		},
	}

	suggestions := simplifyBalances(balances)
	if len(suggestions) != 2 {
		t.Fatalf("suggestion count = %d, want 2", len(suggestions))
	}
	if suggestions[0].From.ID != "b" || suggestions[0].To.ID != "a" || suggestions[0].AmountCents != 600 {
		t.Fatalf("first suggestion = %#v, want B -> A for 600", suggestions[0])
	}
	if suggestions[1].From.ID != "c" || suggestions[1].To.ID != "a" || suggestions[1].AmountCents != 300 {
		t.Fatalf("second suggestion = %#v, want C -> A for 300", suggestions[1])
	}
}

func TestSimplifyBalancesKeepsCurrenciesSeparate(t *testing.T) {
	balances := []balanceResponse{
		{Currency: "INR", Members: []memberBalance{{User: userResponse{ID: "a"}, BalanceCents: 100}, {User: userResponse{ID: "b"}, BalanceCents: -100}}},
		{Currency: "USD", Members: []memberBalance{{User: userResponse{ID: "a"}, BalanceCents: -20}, {User: userResponse{ID: "b"}, BalanceCents: 20}}},
	}

	suggestions := simplifyBalances(balances)
	if len(suggestions) != 2 {
		t.Fatalf("suggestion count = %d, want 2", len(suggestions))
	}
	if suggestions[0].Currency != "INR" || suggestions[1].Currency != "USD" {
		t.Fatalf("currencies = %q, %q; must not be combined", suggestions[0].Currency, suggestions[1].Currency)
	}
}
