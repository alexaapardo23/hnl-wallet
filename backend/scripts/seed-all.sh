#!/bin/sh
# Runs the full seed pipeline once, in the order each step depends on the
# last: Postgres rows first, then the matching TigerBeetle accounts (same
# deterministic ID-by-position as cmd/seed writes into
# accounts.tigerbeetle_account_id), then the initial-balance funding
# transfers, then the historical transaction import. See README Seed Data.
set -e

if ! ./seed-guard; then
  echo "Database already seeded, skipping seed pipeline."
  exit 0
fi

echo "Seeding PostgreSQL (users, accounts, transactions rows)..."
./seed

echo "Creating TigerBeetle accounts..."
./seed-tigerbeetle

echo "Seeding TigerBeetle initial balances..."
./seed-initial-balances

echo "Importing historical transactions into TigerBeetle..."
./seed-transactions

echo "Seed pipeline complete."
