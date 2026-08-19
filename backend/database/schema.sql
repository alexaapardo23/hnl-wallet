-- PostgreSQL schema for HNL Wallet.
--
-- Applied automatically on first boot by mounting this file into
-- /docker-entrypoint-initdb.d/ (see docker-compose.yml) — the official
-- postgres image runs every .sql/.sh file there, but only when the data
-- directory is still empty (a brand new postgres_data volume). It is a
-- no-op on every later `docker compose up` against an already-initialized
-- volume.
--
-- Extracted from the live schema this project has been running against
-- (pg_dump --schema-only) rather than kept as a hand-maintained file that
-- could drift from what the Go code actually expects.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY,
    -- No UNIQUE constraint here — intentional, see README "Data Quality
    -- Decision: Duplicate Emails". The seed dataset has 20 pairs of users
    -- sharing an email; user_id is internal identity, account_number is
    -- financial identity, email is just a contact/auth attribute.
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE accounts (
    account_number VARCHAR(20) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    initial_balance DECIMAL(19, 2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    account_type VARCHAR(20) NOT NULL,
    tigerbeetle_account_id UUID UNIQUE
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    from_account VARCHAR(20) NOT NULL,
    to_account VARCHAR(20) NOT NULL,
    amount DECIMAL(19, 2) NOT NULL,
    type VARCHAR(20) NOT NULL,
    description TEXT,
    timestamp TIMESTAMP NOT NULL,
    status VARCHAR(20) NOT NULL
);

-- Allocates TigerBeetle account IDs for accounts created after the seed
-- import (see backend/api/accounts.go). Starts at 1607: ID 1 is
-- SystemAccountID and 2..1606 are the 1605 seed accounts, both assigned
-- deterministically by cmd/seed-tigerbeetle, not from this sequence.
CREATE SEQUENCE tigerbeetle_account_seq START WITH 1607 INCREMENT BY 1;
