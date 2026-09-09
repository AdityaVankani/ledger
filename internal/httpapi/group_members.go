package httpapi

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/adityavankani/expense-app/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
)

type addGroupMemberRequest struct {
	Email string `json:"email"`
}

type groupMemberResponse struct {
	userResponse
	JoinedAt time.Time `json:"joined_at"`
}

func (a *api) listGroupMembers(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	rows, err := a.queries.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read group members")
		return
	}
	members := make([]groupMemberResponse, 0)
	for _, row := range rows {
		members = append(members, groupMemberResponse{userResponse: userResponse{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, UPIID: row.UPIID}, JoinedAt: row.JoinedAt})
	}
	writeJSON(w, http.StatusOK, members)
}

func (a *api) addGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("groupID")
	if !a.isGroupMember(r, groupID) {
		writeError(w, http.StatusForbidden, "you are not a member of this group")
		return
	}

	var input addGroupMemberRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 320 {
		writeError(w, http.StatusBadRequest, "email must be a valid address")
		return
	}

	row, err := a.queries.GetUserByEmail(r.Context(), input.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user is not registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not look up user")
		return
	}

	joinedAt, err := a.queries.AddGroupMemberReturning(r.Context(), sqlc.AddGroupMemberParams{GroupID: groupID, UserID: row.ID})
	if isUniqueViolation(err) {
		writeError(w, http.StatusConflict, "user is already a group member")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not add group member")
		return
	}
	writeJSON(w, http.StatusCreated, groupMemberResponse{userResponse: userResponse{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName}, JoinedAt: joinedAt})
}

func (a *api) isGroupMember(r *http.Request, groupID string) bool {
	isMember, err := a.queries.IsGroupMember(r.Context(), sqlc.IsGroupMemberParams{GroupID: groupID, UserID: userIDFromContext(r)})
	return err == nil && isMember
}
