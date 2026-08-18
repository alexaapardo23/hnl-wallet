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

## API Endpoints

## Authentication

## Financial Operations

## AI / MCP Integration

## Environment Variables

## Testing

## Running the Application

## Seed Data