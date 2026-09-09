package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adityavankani/expense-app/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
)

const maxExpenseSplits = 100

type createExpenseRequest struct {
	Description  string         `json:"description"`
	AmountCents  int64          `json:"amount_cents"`
	Currency     string         `json:"currency"`
	PaidByUserID string         `json:"paid_by_user_id"`
	ExpenseDate  string         `json:"expense_date"`
	Splits       []expenseSplit `json:"splits"`
}

type expenseHistoryResponse struct {
	ID          string                `json:"id"`
	Description string                `json:"description"`
	AmountCents int64                 `json:"amount_cents"`
	Currency    string                `json:"currency"`
	ExpenseDate string                `json:"expense_date"`
	PaidBy      userResponse          `json:"paid_by"`
	CreatedAt   time.Time             `json:"created_at"`
	Splits      []expenseSplitHistory `json:"splits"`
}

type expenseSplitHistory struct {
	User        userResponse `json:"user"`
	AmountCents int64        `json:"amount_cents"`
}

func (a *api) listExpenses(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}
	limit, ok := parseLimit(w, r)
	if !ok {
		return
	}

	rows, err := a.queries.ListExpenses(r.Context(), sqlc.ListExpensesParams{GroupID: groupID, Limit: limit})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read expenses")
		return
	}
	expenses := make([]expenseHistoryResponse, 0)
	byID := make(map[string]int)
	for _, row := range rows {
		expenseID, description, currency, expenseDate := row.ID, row.Description, row.Currency, row.ExpenseDate
		amountCents, createdAt := row.AmountCents, row.CreatedAt
		payer := userResponse{ID: row.PayerID, Email: row.PayerEmail, DisplayName: row.PayerDisplayName}
		split := expenseSplitHistory{User: userResponse{ID: row.SplitUserID, Email: row.SplitUserEmail, DisplayName: row.SplitUserDisplayName}, AmountCents: row.SplitAmountCents}

		index, found := byID[expenseID]
		if !found {
			index = len(expenses)
			byID[expenseID] = index
			expenses = append(expenses, expenseHistoryResponse{
				ID: expenseID, Description: description, AmountCents: amountCents,
				Currency: currency, ExpenseDate: expenseDate, PaidBy: payer,
				CreatedAt: createdAt, Splits: make([]expenseSplitHistory, 0),
			})
		}
		expenses[index].Splits = append(expenses[index].Splits, split)
	}
	writeJSON(w, http.StatusOK, expenses)
}

func (a *api) deleteExpense(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	_, err := a.queries.DeleteExpense(r.Context(), userIDFromContext(r), r.PathValue("expenseID"), groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "expense not found or cannot be deleted")
		return
	}
	if err != nil {
		writeDatabaseInputError(w, err, "could not delete expense")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) updateExpense(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	var input createExpenseRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Description = strings.TrimSpace(input.Description)
	if input.Currency == "" {
		input.Currency = "INR"
	}
	if input.ExpenseDate == "" {
		input.ExpenseDate = time.Now().UTC().Format(time.DateOnly)
	}
	if err := validateExpense(input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	defer tx.Rollback(r.Context())
	queries := a.queries.WithTx(tx)
	expenseID := r.PathValue("expenseID")
	updated, err := queries.UpdateExpense(r.Context(), sqlc.UpdateExpenseParams{
		ID: expenseID, PaidByUserID: input.PaidByUserID, Description: input.Description,
		AmountCents: input.AmountCents, Currency: input.Currency, ExpenseDate: input.ExpenseDate,
		GroupID: groupID, CreatedByUserID: userIDFromContext(r),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "expense not found or cannot be edited")
		return
	}
	if err != nil {
		writeDatabaseInputError(w, err, "could not update expense")
		return
	}
	if err := queries.DeleteExpenseSplits(r.Context(), expenseID); err != nil {
		writeDatabaseInputError(w, err, "could not update expense splits")
		return
	}
	for _, split := range input.Splits {
		if err := queries.CreateExpenseSplit(r.Context(), sqlc.CreateExpenseSplitParams{
			ExpenseID: expenseID, GroupID: groupID, UserID: split.UserID, AmountCents: split.AmountCents,
		}); err != nil {
			writeDatabaseInputError(w, err, "could not update expense splits")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeDatabaseInputError(w, err, "expense splits must exactly match the expense amount")
		return
	}
	writeJSON(w, http.StatusOK, expenseResponse{
		ID: expenseID, GroupID: groupID, PaidByUserID: input.PaidByUserID,
		CreatedByUserID: userIDFromContext(r), Description: input.Description,
		AmountCents: input.AmountCents, Currency: input.Currency, ExpenseDate: input.ExpenseDate,
		CreatedAt: updated.CreatedAt, Splits: input.Splits,
	})
}

func parseLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	const defaultLimit = 50
	const maxLimit = 100
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit == "" {
		return defaultLimit, true
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit < 1 || limit > maxLimit {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
		return 0, false
	}
	return limit, true
}

type expenseSplit struct {
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
}

type expenseResponse struct {
	ID              string         `json:"id"`
	GroupID         string         `json:"group_id"`
	PaidByUserID    string         `json:"paid_by_user_id"`
	CreatedByUserID string         `json:"created_by_user_id"`
	Description     string         `json:"description"`
	AmountCents     int64          `json:"amount_cents"`
	Currency        string         `json:"currency"`
	ExpenseDate     string         `json:"expense_date"`
	CreatedAt       time.Time      `json:"created_at"`
	Splits          []expenseSplit `json:"splits"`
}

func (a *api) createExpense(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	var input createExpenseRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateExpense(input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if input.Currency == "" {
		input.Currency = "INR"
	}
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Description = strings.TrimSpace(input.Description)
	if input.ExpenseDate == "" {
		input.ExpenseDate = time.Now().UTC().Format(time.DateOnly)
	}

	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	defer tx.Rollback(r.Context())

	response := expenseResponse{
		GroupID:         groupID,
		PaidByUserID:    input.PaidByUserID,
		CreatedByUserID: userIDFromContext(r),
		Description:     input.Description,
		AmountCents:     input.AmountCents,
		Currency:        input.Currency,
		ExpenseDate:     input.ExpenseDate,
		Splits:          input.Splits,
	}
	createdExpense, err := a.queries.WithTx(tx).CreateExpense(r.Context(), sqlc.CreateExpenseParams{
		GroupID: response.GroupID, PaidByUserID: response.PaidByUserID, CreatedByUserID: response.CreatedByUserID,
		Description: response.Description, AmountCents: response.AmountCents, Currency: response.Currency, ExpenseDate: response.ExpenseDate,
	})
	response.ID, response.CreatedAt = createdExpense.ID, createdExpense.CreatedAt
	if err != nil {
		writeDatabaseInputError(w, err, "could not create expense")
		return
	}

	for _, split := range input.Splits {
		if err := a.queries.WithTx(tx).CreateExpenseSplit(r.Context(), sqlc.CreateExpenseSplitParams{ExpenseID: response.ID, GroupID: response.GroupID, UserID: split.UserID, AmountCents: split.AmountCents}); err != nil {
			writeDatabaseInputError(w, err, "could not add expense splits")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeDatabaseInputError(w, err, "expense splits must exactly match the expense amount")
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func validateExpense(input createExpenseRequest) error {
	input.Description = strings.TrimSpace(input.Description)
	if len(input.Description) < 1 || len(input.Description) > 500 {
		return errors.New("description must be between 1 and 500 characters")
	}
	if input.AmountCents <= 0 {
		return errors.New("amount_cents must be greater than zero")
	}
	if !validCurrency(input.Currency) {
		return errors.New("currency must be a three-letter uppercase code")
	}
	if input.PaidByUserID == "" {
		return errors.New("paid_by_user_id is required")
	}
	if input.ExpenseDate != "" {
		if _, err := time.Parse(time.DateOnly, input.ExpenseDate); err != nil {
			return errors.New("expense_date must use YYYY-MM-DD")
		}
	}
	if len(input.Splits) == 0 || len(input.Splits) > maxExpenseSplits {
		return errors.New("splits must contain between 1 and 100 entries")
	}

	var splitTotal int64
	seenUsers := make(map[string]struct{}, len(input.Splits))
	for _, split := range input.Splits {
		if split.UserID == "" {
			return errors.New("each split must include user_id")
		}
		if split.AmountCents <= 0 {
			return errors.New("each split amount_cents must be greater than zero")
		}
		if _, exists := seenUsers[split.UserID]; exists {
			return errors.New("each user can appear only once in splits")
		}
		seenUsers[split.UserID] = struct{}{}
		if split.AmountCents > input.AmountCents-splitTotal {
			return errors.New("split amounts must equal amount_cents")
		}
		splitTotal += split.AmountCents
	}
	if splitTotal != input.AmountCents {
		return errors.New("split amounts must equal amount_cents")
	}
	return nil
}

func validCurrency(currency string) bool {
	if currency == "" {
		return true
	}
	currency = strings.TrimSpace(currency)
	if len(currency) != 3 {
		return false
	}
	for _, character := range currency {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}
