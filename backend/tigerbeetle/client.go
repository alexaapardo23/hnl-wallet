package tigerbeetle

import (
	"fmt"
	"net"
	"os"

	tb "github.com/tigerbeetle/tigerbeetle-go"
)

func NewClient() (tb.Client, error) {
	clusterID := tb.Uint128{0}

	address := os.Getenv("TIGERBEETLE_ADDRESS")
	if address == "" {
		// Local dev default — docker-compose sets TIGERBEETLE_ADDRESS to
		// the tigerbeetle service's name (containers can't reach each
		// other via 127.0.0.1).
		address = "127.0.0.1:3000"
	}

	resolved, err := resolveAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve TigerBeetle address %q: %w", address, err)
	}

	client, err := tb.NewClient(
		clusterID,
		[]string{resolved},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TigerBeetle client: %w", err)
	}

	return client, nil
}

// resolveAddress turns a "host:port" into "ip:port". TigerBeetle's native
// client (tb_client_init, in C — see tigerbeetle-go's tb_client.go) parses
// addresses itself and doesn't do DNS resolution, so a Docker service name
// like "tigerbeetle:3000" fails there with ErrInvalidAddress even though
// it's a perfectly normal address anywhere else. Resolving it here first
// keeps TIGERBEETLE_ADDRESS itself hostname-friendly (matching every other
// *_URL/*_ADDRESS env var in this project) without patching the vendored
// client.
func resolveAddress(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", err
	}

	if net.ParseIP(host) != nil {
		return address, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no addresses found for host %q", host)
	}

	return net.JoinHostPort(ips[0].String(), port), nil
}
