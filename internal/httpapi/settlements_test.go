package httpapi

import "testing"

func TestValidateSettlement(t *testing.T) {
	valid := createSettlementRequest{
		ReceivedByUserID: "recipient",
		AmountCents:      500,
		Currency:         "INR",
		SettledAt:        "2026-09-09T12:30:00Z",
		Note:             "UPI payment",
	}

	tests := []struct {
		name  string
		input createSettlementRequest
		valid bool
	}{
		{name: "valid", input: valid, valid: true},
		{name: "self payment", input: replaceRecipient(valid, "payer")},
		{name: "zero amount", input: replaceSettlementAmount(valid, 0)},
		{name: "invalid time", input: replaceSettlementTime(valid, "tomorrow")},
		{name: "lowercase currency", input: replaceSettlementCurrency(valid, "inr")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSettlement(test.input, "payer")
			if (err == nil) != test.valid {
				t.Fatalf("validateSettlement() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}

func replaceRecipient(input createSettlementRequest, recipient string) createSettlementRequest {
	input.ReceivedByUserID = recipient
	return input
}

func replaceSettlementAmount(input createSettlementRequest, amount int64) createSettlementRequest {
	input.AmountCents = amount
	return input
}

func replaceSettlementTime(input createSettlementRequest, settledAt string) createSettlementRequest {
	input.SettledAt = settledAt
	return input
}

func replaceSettlementCurrency(input createSettlementRequest, currency string) createSettlementRequest {
	input.Currency = currency
	return input
}
