package httpapi

import "net/http"

func (a *api) logout(w http.ResponseWriter, r *http.Request) {
	tokenHash := sessionTokenHashFromContext(r)
	rowsAffected, err := a.queries.DeleteSession(r.Context(), userIDFromContext(r), tokenHash[:])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not end session")
		return
	}
	if rowsAffected != 1 {
		writeError(w, http.StatusUnauthorized, "invalid or expired session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
