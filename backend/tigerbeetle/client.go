package tigerbeetle

import (
	"fmt"

	tb "github.com/tigerbeetle/tigerbeetle-go"
)

func NewClient() (tb.Client, error) {
	clusterID := tb.Uint128{0}

	client, err := tb.NewClient(
		clusterID,
		[]string{"127.0.0.1:3000"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TigerBeetle client: %w", err)
	}

	return client, nil
}