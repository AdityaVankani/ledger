package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/adityavankani/expense-app/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
)

type createGroupRequest struct {
	Name string `json:"name"`
}

func (a *api) deleteGroup(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}
	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	defer tx.Rollback(r.Context())
	queries := a.queries.WithTx(tx)
	if err := queries.PurgeDeletedGroupSettlements(r.Context(), groupID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete group")
		return
	}
	if err := queries.PurgeDeletedGroupExpenses(r.Context(), groupID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete group")
		return
	}
	_, err = queries.DeleteGroup(r.Context(), groupID, userIDFromContext(r))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}
	if err != nil {
		if isForeignKeyViolation(err) {
			writeError(w, http.StatusConflict, "delete or settle all financial records before deleting this group")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not delete group")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete group")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type groupResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (a *api) createGroup(w http.ResponseWriter, r *http.Request) {
	var input createGroupRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) < 1 || len(input.Name) > 100 {
		writeError(w, http.StatusBadRequest, "name must be between 1 and 100 characters")
		return
	}

	userID := userIDFromContext(r)
	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	defer tx.Rollback(r.Context())

	createdGroup, err := a.queries.WithTx(tx).CreateGroup(r.Context(), sqlc.CreateGroupParams{
		Name: input.Name, CreatedByUserID: userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create group")
		return
	}
	if err := a.queries.WithTx(tx).AddGroupMember(r.Context(), sqlc.AddGroupMemberParams{GroupID: createdGroup.ID, UserID: userID}); err != nil {
		writeError(w, http.StatusInternalServerError, "could not add group owner")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create group")
		return
	}
	writeJSON(w, http.StatusCreated, groupResponse{ID: createdGroup.ID, Name: createdGroup.Name, CreatedAt: createdGroup.CreatedAt})
}

func (a *api) listGroups(w http.ResponseWriter, r *http.Request) {
	rows, err := a.queries.ListGroupsForUser(r.Context(), userIDFromContext(r))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	groups := make([]groupResponse, 0)
	for _, row := range rows {
		groups = append(groups, groupResponse{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt})
	}
	writeJSON(w, http.StatusOK, groups)
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	row, err := a.queries.GetUserByID(r.Context(), userIDFromContext(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	user := userResponse{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, UPIID: row.UPIID}
	writeJSON(w, http.StatusOK, user)
}
