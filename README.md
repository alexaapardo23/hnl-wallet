# HNL Wallet

Technical assessment for a digital wallet application.

## Overview

HNL Wallet is a digital wallet application that allows users to:

- Register and authenticate
- Create and manage a wallet account
- Check their balance
- Deposit funds
- Withdraw funds
- Transfer funds to other accounts
- View transaction history
- Interact with the wallet through an AI-powered chat

The application uses **USD** as its currency.

## Architecture

The application uses a dual-database architecture:

- **PostgreSQL**: user information and authentication
- **TigerBeetle**: financial accounts, balances, and financial transfers

The application is composed of:

- **Frontend**: React + Vite
- **Backend**: Go
- **Database**: PostgreSQL
- **Financial ledger**: TigerBeetle
- **Infrastructure**: Docker / Docker Compose
- **AI integration**: MCP

The main architectural principle is that PostgreSQL manages application and user data, while TigerBeetle is the source of truth for financial data.

## Project Structure

```text
hnl-wallet/
├── backend/
├── frontend/
├── data/
├── docker-compose.yml
├── .env
├── .gitignore
└── README.md
```

## Local PostgreSQL

PostgreSQL runs as a Docker container during development.

The development database is configured with:

- **Database**: `hnl_wallet`
- **User**: `hnl_user`
- **Port**: `5432`

Start PostgreSQL with:

```bash
docker compose up -d
```

## TigerBeetle

TigerBeetle is used as the financial ledger for HNL Wallet.

It is responsible for:

- Financial accounts
- Account balances
- Deposits
- Withdrawals
- Transfers

For local development, TigerBeetle runs as a Docker container with a single replica.

The local TigerBeetle instance is configured with:

- **Port**: `3000`
- **Data volume**: `tigerbeetle_data`
- **Cluster**: `0`
- **Replica**: `0`
- **Replica count**: `1`

TigerBeetle requires `io_uring` for its storage engine. The Docker container uses the following security configuration:

```yaml
security_opt:
  - seccomp=unconfined
```

## Financial Data Model

HNL Wallet uses TigerBeetle as the source of truth for financial balances and transfers.

PostgreSQL stores the relationship between application users and their TigerBeetle accounts.

Conceptually:

```text
PostgreSQL                         TigerBeetle

┌──────────────┐                  ┌──────────────────┐
│    users     │                  │     accounts     │
├──────────────┤                  ├──────────────────┤
│ id           │───────┐          │ id               │
│ email        │       │          │ balance          │
│ password     │       └─────────►│ ledger           │
└──────────────┘                  │ code             │
                                  └──────────────────┘
```

The application does not store the user's financial balance in PostgreSQL. The balance is derived from the financial activity recorded in TigerBeetle.

### Currency Representation

HNL Wallet uses USD.

Financial amounts are represented as integer cents rather than floating-point numbers.

For example:

```text
$1.00    → 100 cents
$10.00   → 1000 cents
$25.50   → 2550 cents
$100.00  → 10000 cents
```

This avoids floating-point precision issues when handling monetary values.

### TigerBeetle Ledger Configuration

The current development configuration uses:

- **Ledger**: `700` for the USD ledger
- **Code**: `10` for wallet accounts

These values are application-level conventions and are not intrinsic representations of USD in TigerBeetle.

### Development Accounts

The current local TigerBeetle instance contains three test accounts:

| Account | Owner |
|---|---|
| Account 1 | Alexandra |
| Account 2 | Gerardo |
| Account 100 | System / Funding account |

All accounts belong to:

- **Ledger**: `700`
- **Code**: `10`

## Running the Infrastructure

Start PostgreSQL and TigerBeetle with:

```bash
docker compose up -d
```

Check the running services with:

```bash
docker compose ps
```

The expected services are:

- `hnl-postgres` on port `5432`
- `hnl-tigerbeetle` on port `3000`

## Backend

## Frontend

## Database Schema

### Identity Model

Users are identified by three distinct attributes, each with a different purpose:

| Attribute | Role | Unique? |
|---|---|---|
| `user_id` (UUID) | Internal identity — primary key, used for all internal relations (accounts, ownership) | Yes |
| `account_number` | Visible financial identity — what identifies a wallet/account for transfers | Yes |
| `email` | Contact / authentication attribute | **No** |

`users.email` does **not** have a uniqueness constraint at the database level. This was a deliberate decision — see [Authentication](#authentication) for the reasoning.

## API Endpoints

## Authentication

### Duplicate emails are allowed by design

The seed dataset (`data/data.json`) intentionally contains users that share the same email address but have different `user_id`s, `full_name`s, and their own real accounts/transactions in TigerBeetle. This was verified: all 40 users involved in the 20 duplicate-email groups have active accounts and transaction history.

Since `user_id` (not `email`) is the true internal identity, and `account_number` is the true financial identity, a `UNIQUE` constraint on `users.email` was removed from the schema — enforcing it would have made it impossible to seed this intentionally "dirty" dataset without discarding or rewriting real financial data tied to those users, which was considered out of scope for this exercise.

**Practical consequence for login (not yet implemented):** since `email` alone cannot uniquely identify a user, and some duplicate-email pairs even share the same password, login must not rely on `email` + `password` alone. The planned login credentials are:

- `email`
- `password`
- `account_number`

`account_number` disambiguates between users that share both email and password, while keeping `email` as a contact/login-facing attribute rather than a strict unique identifier.

## Financial Operations

### Balance Reconciliation (initial_balance vs. transactions)

For each account, a balance can be derived from the seed dataset alone:

```text
calculated_balance =
  initial_balance
  + deposits            (EXTERNAL → account)
  + incoming transfers  (transfer / internal_transfer into the account)
  - withdrawals         (account → EXTERNAL)
  - outgoing transfers  (transfer / internal_transfer out of the account)
```

This was checked against `data/data.json` (1605 accounts, 6429 transactions, all `status: completed`):

- **Referential integrity**: every `from_account` / `to_account` in `transactions` resolves to a real `account_number` or the `EXTERNAL` sentinel — 0 orphan references.
- **Aggregate consistency**: `transfer` / `internal_transfer` are zero-sum within the system, so the sum of all `calculated_balance` values must equal `Σ initial_balance + Σ deposits - Σ withdrawals`. Verified exactly: both sides equal **$40,364,992.41**, difference `0.00`.
- **Negative balances**: 70 of 1605 accounts (4.4%) end up with a negative `calculated_balance` (as low as **-$10,933.69**), spread evenly across account types (savings 4.2%, checking 4.1%, investment 6.4%).

The dataset does not prevent overdrafts — transaction amounts are not capped by available balance. This is assumed to be intentional test data, since TigerBeetle supports enforcing non-negative balances via account flags (e.g. `debits_must_not_exceed_credits`), which is not yet wired up in this project.

## AI / MCP Integration

## Environment Variables

## Testing

## Running the Application

## Seed Data

`data/data.json` intentionally includes 20 pairs of users (40 users total) sharing the same email address, each with their own accounts and transaction history. See [Authentication](#authentication) for how this is handled.