package httpapi

import "testing"

func TestValidateExpense(t *testing.T) {
	valid := createExpenseRequest{
		Description:  "Dinner",
		AmountCents:  1200,
		Currency:     "INR",
		PaidByUserID: "payer",
		ExpenseDate:  "2026-09-09",
		Splits: []expenseSplit{
			{UserID: "payer", AmountCents: 600},
			{UserID: "member", AmountCents: 600},
		},
	}

	tests := []struct {
		name  string
		input createExpenseRequest
		valid bool
	}{
		{name: "valid", input: valid, valid: true},
		{name: "mismatched split total", input: withAmount(valid, 1201)},
		{name: "duplicate recipient", input: withSplits(valid, []expenseSplit{{UserID: "payer", AmountCents: 600}, {UserID: "payer", AmountCents: 600}})},
		{name: "zero amount", input: withAmount(valid, 0)},
		{name: "invalid date", input: withDate(valid, "2026-02-29")},
		{name: "lowercase currency", input: withCurrency(valid, "inr")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateExpense(test.input)
			if (err == nil) != test.valid {
				t.Fatalf("validateExpense() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func withAmount(input createExpenseRequest, amount int64) createExpenseRequest {
	input.AmountCents = amount
	return input
}

func withSplits(input createExpenseRequest, splits []expenseSplit) createExpenseRequest {
	input.Splits = splits
	return input
}

func withDate(input createExpenseRequest, date string) createExpenseRequest {
	input.ExpenseDate = date
	return input
}

func withCurrency(input createExpenseRequest, currency string) createExpenseRequest {
	input.Currency = currency
	return input
}
