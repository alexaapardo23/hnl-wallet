package seed

import (
	"context"
	"fmt"

	"hnl-wallet/backend/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedPostgres(ctx context.Context, pool *pgxpool.Pool, data *models.Data) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	// Users
	for _, user := range data.Users {
		_, err := tx.Exec(
			ctx,
			`
			INSERT INTO users (
				id,
				email,
				password,
				full_name,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO NOTHING
			`,
			user.ID,
			user.Email,
			user.Password,
			user.FullName,
			user.CreatedAt,
		)

		if err != nil {
			return fmt.Errorf(
				"failed to insert user %s: %w",
				user.ID,
				err,
			)
		}
	}

	// Accounts
	for _, account := range data.Accounts {
		_, err := tx.Exec(
			ctx,
			`
			INSERT INTO accounts (
				account_number,
				user_id,
				initial_balance,
				currency,
				account_type
			)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (account_number) DO NOTHING
			`,
			account.AccountNumber,
			account.UserID,
			account.InitialBalance,
			account.Currency,
			account.AccountType,
		)

		if err != nil {
			return fmt.Errorf(
				"failed to insert account %s: %w",
				account.AccountNumber,
				err,
			)
		}
	}

	// Transactions
	for _, transaction := range data.Transactions {
		_, err := tx.Exec(
			ctx,
			`
			INSERT INTO transactions (
				id,
				from_account,
				to_account,
				amount,
				type,
				description,
				timestamp,
				status
			)
			VALUES (
				gen_random_uuid(),
				$1, $2, $3, $4, $5, $6, $7
			)
			`,
			transaction.FromAccount,
			transaction.ToAccount,
			transaction.Amount,
			transaction.Type,
			transaction.Description,
			transaction.Timestamp,
			transaction.Status,
		)

		if err != nil {
			return fmt.Errorf(
				"failed to insert transaction: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit seed transaction: %w", err)
	}

	return nil
}