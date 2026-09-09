package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/adityavankani/expense-app/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const maxRequestBodyBytes = 1 << 20

type api struct {
	pool       *pgxpool.Pool
	queries    *sqlc.Queries
	sessionTTL time.Duration
}

type contextKey string

const userIDContextKey contextKey = "userID"
const sessionTokenHashContextKey contextKey = "sessionTokenHash"

func NewAPI(pool *pgxpool.Pool, sessionTTL time.Duration) *api {
	return &api{pool: pool, queries: sqlc.New(pool), sessionTTL: sessionTTL}
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	UPIID       string `json:"upi_id"`
}

type authResponse struct {
	User         userResponse `json:"user"`
	SessionToken string       `json:"session_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

type registerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateProfileRequest struct {
	UPIID string `json:"upi_id"`
}

func (a *api) updateProfile(w http.ResponseWriter, r *http.Request) {
	var input updateProfileRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.UPIID = strings.TrimSpace(input.UPIID)
	if len(input.UPIID) > 320 {
		writeError(w, http.StatusBadRequest, "upi_id must be at most 320 characters")
		return
	}
	row, err := a.queries.UpdateUserUPI(r.Context(), userIDFromContext(r), input.UPIID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update profile")
		return
	}
	writeJSON(w, http.StatusOK, userResponse{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, UPIID: row.UPIID})
}

func (a *api) register(w http.ResponseWriter, r *http.Request) {
	var input registerRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if err := validateRegistration(input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not secure password")
		return
	}

	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	defer tx.Rollback(r.Context())

	createdUser, err := a.queries.WithTx(tx).CreateUser(r.Context(), sqlc.CreateUserParams{
		Email: input.Email, DisplayName: input.DisplayName, PasswordHash: string(passwordHash),
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "email is already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	user := userResponse{ID: createdUser.ID, Email: createdUser.Email, DisplayName: createdUser.DisplayName}

	response, err := a.createSession(r.Context(), tx, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	defer tx.Rollback(r.Context())

	userRow, err := a.queries.WithTx(tx).GetUserByEmail(r.Context(), email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(userRow.PasswordHash), []byte(input.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	user := userResponse{ID: userRow.ID, Email: userRow.Email, DisplayName: userRow.DisplayName}

	response, err := a.createSession(r.Context(), tx, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not sign in")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *api) createSession(ctx context.Context, tx pgx.Tx, user userResponse) (authResponse, error) {
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return authResponse{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	tokenHash := sha256.Sum256([]byte(token))
	expiresAt := time.Now().UTC().Add(a.sessionTTL)
	if err := sqlc.New(tx).CreateSession(ctx, sqlc.CreateSessionParams{UserID: user.ID, TokenHash: tokenHash[:], ExpiresAt: expiresAt}); err != nil {
		return authResponse{}, err
	}
	return authResponse{User: user, SessionToken: token, ExpiresAt: expiresAt}, nil
}

func validateRegistration(input registerRequest) error {
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 320 {
		return errors.New("email must be a valid address")
	}
	if len(input.DisplayName) < 1 || len(input.DisplayName) > 100 {
		return errors.New("display_name must be between 1 and 100 characters")
	}
	if len(input.Password) < 12 || len(input.Password) > 72 {
		return errors.New("password must be between 12 and 72 characters")
	}
	return nil
}

func (a *api) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "bearer token is required")
			return
		}
		tokenHash := sha256.Sum256([]byte(token))
		userID, err := a.queries.GetUserIDBySessionTokenHash(r.Context(), tokenHash[:])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired session")
			return
		}
		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		ctx = context.WithValue(ctx, sessionTokenHashContextKey, tokenHash)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userIDFromContext(r *http.Request) string {
	return r.Context().Value(userIDContextKey).(string)
}

func sessionTokenHashFromContext(r *http.Request) [sha256.Size]byte {
	return r.Context().Value(sessionTokenHashContextKey).([sha256.Size]byte)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func writeDatabaseInputError(w http.ResponseWriter, err error, fallback string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "22P02", "23503", "23514":
			writeError(w, http.StatusBadRequest, pgErr.Message)
			return
		}
	}
	writeError(w, http.StatusInternalServerError, fallback)
}
