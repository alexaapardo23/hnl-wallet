package main

import (
	"context"
	"log"
	"os"

	"hnl-wallet/backend/database"
)

// seed-guard exits 0 (safe to seed) only when the users table is empty,
// and exits 1 otherwise. It exists so the one-shot `seed` service in
// docker-compose can skip re-running the seed pipeline against an
// already-seeded volume: re-running seed-initial-balances or
// seed-transactions would double-credit every account, since TigerBeetle
// transfer IDs are random (tb.ID()) rather than derived from the source
// data, so a second run isn't deduplicated the way the Postgres
// ON CONFLICT DO NOTHING inserts are.
func main() {
	pool, err := database.NewPostgresPool()
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	var count int
	if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		log.Fatal(err)
	}

	if count > 0 {
		log.Printf("users table already has %d rows — already seeded", count)
		os.Exit(1)
	}

	os.Exit(0)
}
