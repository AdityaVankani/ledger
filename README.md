# Expense App

A simple app to manage shared expenses with friends, roommates, travel groups, or teams.

## What it does

This app helps people:

- create groups for shared spending
- add members to each group
- record expenses and split the cost fairly
- see who owes whom
- track balances in real time
- generate settlement suggestions to settle payments

The main purpose is to remove the hassle of manually calculating who paid for what and how much each person should contribute.

## Why this app exists

When a group shares costs, it is easy to lose track of expenses, duplicates, and owed amounts. This app gives a single place to log expenses, store group activity, and calculate balances automatically.

## Main features

- User sign up and login
- Group creation and member management
- Expense entry with split amounts
- Balance tracking by group
- Settlement recording
- Suggested payment transfers between members
- Simple web interface for day-to-day usage

## Tech stack

- Go for the backend API
- PostgreSQL for data storage
- React + Vite for the frontend
- Docker for local database setup

## Basic setup

### 1. Clone the project

```bash
git clone <repo-url>
cd <project-folder>
```

### 2. Start PostgreSQL

```bash
docker compose up -d postgres
```

### 3. Set environment variables

Create a `.env` file or export the required values:

```bash
DATABASE_URL=postgres://expense_app:expense_app@localhost:5433/expense_app?sslmode=disable
APP_PORT=8080
SESSION_TTL=168h
ALLOWED_ORIGIN=http://localhost:5173
```

### 4. Run database migrations

```bash
make migrate-up
```

### 5. Start the backend

```bash
go run ./cmd/api
```

### 6. Start the frontend

```bash
cd web
npm install
npm run dev
```

The app should be available locally through the frontend, with the backend running on `http://localhost:8080`.

## Typical use flow

1. Create an account
2. Create a group
3. Add your friends or roommates
4. Add expenses for shared items
5. Check balances
6. Mark settlements as paid

## Project goal

This project is a practical group expense tracker designed to be easy to use, reliable for shared spending, and simple to run locally for development.

## Already deployed

The app is already live and ready to use. You can open it in the browser, start managing shared expenses right away, and recommend it to your friends and circles if it helps simplify group spending.

Use it for trips, house expenses, team outings, or any shared budget where people want a quick and fair way to track who owes what.

