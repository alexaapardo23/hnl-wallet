package tigerbeetle

import (
	"fmt"
	"strings"

	tb "github.com/tigerbeetle/tigerbeetle-go"
)

// SystemAccountIDValue is the reserved TigerBeetle account ID representing
// the external funding source ("the bank"): deposits originate from it and
// withdrawals return to it. Real accounts start at 2, so it can never collide
// with a mapped account.
const SystemAccountIDValue uint64 = 1

// SystemAccountID is SystemAccountIDValue as a TigerBeetle Uint128.
var SystemAccountID = tb.ToUint128(SystemAccountIDValue)

// AccountID returns the deterministic TigerBeetle account ID for the account
// at the given zero-based index in the seed dataset (data.Accounts order).
// ID 1 is reserved for SystemAccountID, so real accounts occupy 2..N+1.
func AccountID(index int) tb.Uint128 {
	return tb.ToUint128(uint64(index) + SystemAccountIDValue + 1)
}

// UUIDString renders a TigerBeetle Uint128 as a canonical UUID string
// (e.g. "00000000-0000-0000-0000-000000000001"), so it can be stored in
// PostgreSQL's uuid column (accounts.tigerbeetle_account_id) and stay
// human-readable when inspected directly in psql.
//
// Only values that fit in 64 bits are supported, which covers every ID
// this mapping generates.
func UUIDString(id tb.Uint128) string {
	lo, _ := id.Uint64()
	return fmt.Sprintf("00000000-0000-0000-0000-%012x", lo)
}

// ParseUUIDString parses a UUID string produced by UUIDString back into the
// TigerBeetle Uint128 it represents.
func ParseUUIDString(s string) (tb.Uint128, error) {
	return tb.HexStringToUint128(strings.ReplaceAll(s, "-", ""))
}
