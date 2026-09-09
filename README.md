# Expense App backend

Minimal backend foundation for a group-expense application using Go, PostgreSQL, pgx, and sqlc.

## Local setup

1. Copy `.env.example` to `.env` and export its values (or set them in your shell).
2. Start PostgreSQL: `docker compose up -d postgres`.
3. Apply the migration: `DATABASE_URL="$DATABASE_URL" make migrate-up`.
4. Start the API: `go run ./cmd/api`.
5. Check it with `curl -i http://localhost:8080/healthz`.

## Frontend

The React/Vite client lives in `web/` and expects the API at `http://localhost:8080` by default.

```sh
cd web
npm install
npm run dev
```

Set `VITE_API_BASE` before starting Vite when the API runs elsewhere. The client includes registration, login, group creation, member invites, expense entry, balances, and settlement suggestions.

## Free deployment

The app can be deployed without a paid hosting plan using free tiers. Free tiers may sleep, have usage limits, or change over time; this does not require a payment method in every region, but provider signup policies vary.

Recommended split:

- **Supabase Free**: PostgreSQL database.
- **Render Free**: Go API using the repository `Dockerfile`.
- **Vercel Hobby**: Vite frontend from the `web/` directory.

### 1. Create the database

1. Create a Supabase project on the Free plan.
2. Copy its direct PostgreSQL connection string and keep the SSL option enabled (`sslmode=require`).
3. Apply `migrations/000001_initial_schema.up.sql`, `000002_auth_sessions.up.sql`, `000003_soft_delete.up.sql`, and `000004_user_upi_id.up.sql` in order using the Supabase SQL editor. Run each file once.

### 2. Deploy the API

1. Push this repository to GitHub.
2. Create a Render **Web Service** from the repository and choose the free instance.
3. Use Docker as the runtime; Render will use the root `Dockerfile`.
4. Add environment variables:

```text
DATABASE_URL=your Supabase direct PostgreSQL URL
APP_PORT=10000
SESSION_TTL=168h
ALLOWED_ORIGIN=https://your-frontend.vercel.app
```

5. Deploy and verify `https://your-api.onrender.com/healthz` returns `204`.

### 3. Deploy the frontend

1. Import the same repository into Vercel.
2. Set the project root to `web`.
3. Build command: `npm run build`.
4. Output directory: `dist`.
5. Add `VITE_API_BASE=https://your-api.onrender.com`.
6. Deploy, then replace Render's `ALLOWED_ORIGIN` with the final Vercel URL and redeploy the API.

The API uses bearer tokens, so no cookie domain configuration is needed. Never commit `.env`, database URLs, or provider secrets.

## Structure

- `cmd/api`: service entry point and lifecycle management.
- `internal/config`: environment-based configuration.
- `internal/database`: PostgreSQL pgx pool setup.
- `internal/httpapi`: HTTP router; only a database-backed health endpoint exists today.
- `migrations`: ordered, reversible PostgreSQL schema migrations.
- `db/queries`: SQL inputs for sqlc. Run `make sqlc` after adding queries.

Money is represented as integer cents (`amount_cents`), never floating point. Expense splits are validated by a deferred database constraint, so a transaction can create an expense and all of its splits together, but it cannot commit an incomplete or mismatched split. Payers, creators, split recipients, and settlement participants must all be members of the relevant group.

## Available endpoints

- `POST /v1/auth/register` — create a user and a session.
- `POST /v1/auth/login` — create a session for an existing user.
- `DELETE /v1/auth/session` — revoke the active bearer-token session.
- `GET /v1/me` — retrieve the signed-in user.
- `POST /v1/groups` — create a group and add its creator as the first member.
- `GET /v1/groups` — list groups to which the signed-in user belongs.
- `GET /v1/groups/{groupID}/members` — list members of one of the signed-in user's groups.
- `POST /v1/groups/{groupID}/members` — add a registered user by email.
- `POST /v1/groups/{groupID}/expenses` — record a group expense and every split in one transaction.
- `GET /v1/groups/{groupID}/expenses?limit=50` — show recent expenses, including their splits.
- `DELETE /v1/groups/{groupID}/expenses/{expenseID}` — soft-delete an expense created by the signed-in user.
- `GET /v1/groups/{groupID}/balances` — show each member's net position per currency.
- `POST /v1/groups/{groupID}/settlements` — record the signed-in user's payment to another member.
- `DELETE /v1/groups/{groupID}/settlements/{settlementID}` — soft-delete a settlement paid by the signed-in user.
- `GET /v1/groups/{groupID}/settlement-suggestions` — return a compact set of suggested payments.

Authenticated endpoints require `Authorization: Bearer <session_token>`. Session tokens are random, only their SHA-256 hashes are stored, and sessions expire according to `SESSION_TTL`.

All current group members may manage membership. Roles are intentionally deferred until the MVP proves it needs them.

### Create expense request

```json
{
  "description": "Dinner",
  "amount_cents": 1200,
  "currency": "INR",
  "paid_by_user_id": "member UUID",
  "expense_date": "2026-09-09",
  "splits": [
    {"user_id": "first member UUID", "amount_cents": 600},
    {"user_id": "second member UUID", "amount_cents": 600}
  ]
}
```

`currency` defaults to `INR` and `expense_date` defaults to today. Amounts must be integer cents. Every split must be for a group member and all splits must total exactly to `amount_cents`.

A positive balance means the member is owed money; a negative balance means the member owes money. Balances include settlements when they are recorded, and are returned separately for each currency.

### Create settlement request

```json
{
  "received_by_user_id": "member UUID",
  "amount_cents": 600,
  "currency": "INR",
  "settled_at": "2026-09-09T12:30:00Z",
  "note": "UPI payment"
}
```

The authenticated user is always recorded as the payer. `currency` defaults to `INR`; `settled_at` defaults to the current time.

Settlement suggestions are derived from current net balances and are returned per currency. The algorithm deterministically matches the largest outstanding debts and credits, producing no more than one fewer transfers than the number of involved members. They are recommendations only; recording a payment still requires the settlement endpoint.

Deleting an expense or settlement is reversible at the database level: records retain who deleted them and when. Deleted records are excluded from histories, balances, and settlement suggestions. Editing financial records is intentionally deferred until revision history is designed.

## Integration test

After applying migrations to an isolated local database, run the end-to-end test with:

```sh
DATABASE_URL="postgres://expense_app:expense_app@localhost:5433/expense_app?sslmode=disable" go test -tags=integration ./internal/httpapi
```
