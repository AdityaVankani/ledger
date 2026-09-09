package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/adityavankani/expense-app/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
)

type createSettlementRequest struct {
	ReceivedByUserID string `json:"received_by_user_id"`
	AmountCents      int64  `json:"amount_cents"`
	Currency         string `json:"currency"`
	SettledAt        string `json:"settled_at"`
	Note             string `json:"note"`
}

type settlementResponse struct {
	ID               string    `json:"id"`
	GroupID          string    `json:"group_id"`
	PaidByUserID     string    `json:"paid_by_user_id"`
	ReceivedByUserID string    `json:"received_by_user_id"`
	AmountCents      int64     `json:"amount_cents"`
	Currency         string    `json:"currency"`
	SettledAt        time.Time `json:"settled_at"`
	Note             string    `json:"note"`
	CreatedAt        time.Time `json:"created_at"`
}

func (a *api) createSettlement(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	var input createSettlementRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Currency = strings.TrimSpace(input.Currency)
	input.Note = strings.TrimSpace(input.Note)
	if err := validateSettlement(input, userIDFromContext(r)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.Currency == "" {
		input.Currency = "INR"
	} else {
		input.Currency = strings.ToUpper(input.Currency)
	}

	settledAt := time.Now().UTC()
	if input.SettledAt != "" {
		var err error
		settledAt, err = time.Parse(time.RFC3339, input.SettledAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "settled_at must use RFC 3339 format")
			return
		}
	}

	response := settlementResponse{
		GroupID:          groupID,
		PaidByUserID:     userIDFromContext(r),
		ReceivedByUserID: input.ReceivedByUserID,
		AmountCents:      input.AmountCents,
		Currency:         input.Currency,
		SettledAt:        settledAt,
		Note:             input.Note,
	}
	created, err := a.queries.CreateSettlement(r.Context(), sqlc.CreateSettlementParams{
		GroupID: response.GroupID, PaidByUserID: response.PaidByUserID,
		ReceivedByUserID: response.ReceivedByUserID, AmountCents: response.AmountCents,
		Currency: response.Currency, SettledAt: response.SettledAt, Note: response.Note,
	})
	response.ID, response.CreatedAt = created.ID, created.CreatedAt
	if err != nil {
		writeDatabaseInputError(w, err, "could not record settlement")
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func validateSettlement(input createSettlementRequest, payerUserID string) error {
	if input.ReceivedByUserID == "" {
		return errors.New("received_by_user_id is required")
	}
	if input.ReceivedByUserID == payerUserID {
		return errors.New("a settlement must be between two different members")
	}
	if input.AmountCents <= 0 {
		return errors.New("amount_cents must be greater than zero")
	}
	if !validCurrency(input.Currency) {
		return errors.New("currency must be a three-letter uppercase code")
	}
	if len(input.Note) > 500 {
		return errors.New("note must be at most 500 characters")
	}
	if input.SettledAt != "" {
		if _, err := time.Parse(time.RFC3339, input.SettledAt); err != nil {
			return errors.New("settled_at must use RFC 3339 format")
		}
	}
	return nil
}

func (a *api) deleteSettlement(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	_, err := a.queries.DeleteSettlement(r.Context(), sqlc.DeleteSettlementParams{
		DeletedByUserID: userIDFromContext(r), ID: r.PathValue("settlementID"), GroupID: groupID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "settlement not found or cannot be deleted")
		return
	}
	if err != nil {
		writeDatabaseInputError(w, err, "could not delete settlement")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
