package tigerbeetle

import (
	"fmt"
	"math"
	"strings"

	tb "github.com/tigerbeetle/tigerbeetle-go"
)

// CentsFromDollars converts a dollar amount (as used throughout data.json
// and API request bodies) into the integer cents TigerBeetle amounts are
// represented in — see README Currency Representation.
func CentsFromDollars(dollars float64) uint64 {
	return uint64(math.Round(dollars * 100))
}

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

// Account codes. Ledger 700 is the single USD ledger (see README), so
// account_type is what code distinguishes between accounts on that ledger.
const (
	CodeSystem     uint16 = 1 // SystemAccountID only
	CodeChecking   uint16 = 10
	CodeSavings    uint16 = 11
	CodeInvestment uint16 = 12
)

// AccountTypeCode maps a data.json account_type to its TigerBeetle Code.
func AccountTypeCode(accountType string) (uint16, error) {
	switch accountType {
	case "checking":
		return CodeChecking, nil
	case "savings":
		return CodeSavings, nil
	case "investment":
		return CodeInvestment, nil
	default:
		return 0, fmt.Errorf("unknown account_type: %q", accountType)
	}
}

// AccountNumberToUserData128 encodes a "XXXX-XXXX-XXXX-XXXX" account_number
// into UserData128 by dropping the dashes and storing the 16 remaining
// digits as raw ASCII bytes — exactly filling the 16 bytes of a Uint128.
// This lets an account_number be recovered directly from a TigerBeetle
// account without a round trip through PostgreSQL.
func AccountNumberToUserData128(accountNumber string) (tb.Uint128, error) {
	digits := strings.ReplaceAll(accountNumber, "-", "")
	if len(digits) != 16 {
		return tb.Uint128{}, fmt.Errorf(
			"account_number %q must have 16 digits once dashes are removed, got %d",
			accountNumber, len(digits),
		)
	}

	var bytes [16]byte
	copy(bytes[:], digits)

	return tb.BytesToUint128(bytes), nil
}

// UserData128ToAccountNumber decodes a Uint128 produced by
// AccountNumberToUserData128 back into its "XXXX-XXXX-XXXX-XXXX" form.
func UserData128ToAccountNumber(userData tb.Uint128) string {
	bytes := userData.Bytes()
	digits := string(bytes[:])

	return fmt.Sprintf("%s-%s-%s-%s", digits[0:4], digits[4:8], digits[8:12], digits[12:16])
}

// Transfer codes. This is a separate namespace from Account codes (above) —
// TigerBeetle gives Account and Transfer independent Code fields — but is
// kept numerically distinct (100+) purely so the two never look ambiguous
// side by side in logs or queries. The specific value carries no meaning
// beyond being reserved and documented here.
const (
	// CodeInitialBalance marks the one-time funding transfer that seeds an
	// account's initial_balance from SystemAccountID when the account is
	// first provisioned. DebitAccountID is always SystemAccountID and
	// CreditAccountID the account being funded, so SystemAccountID's
	// DebitsPosted accumulates to sum(initial_balance) across all accounts.
	CodeInitialBalance uint16 = 100

	// The following mark historical transactions imported from
	// data/data.json (see cmd/seed-transactions). In every case,
	// DebitAccountID is where the money leaves from and CreditAccountID is
	// where it arrives — matching the convention already established by
	// CodeInitialBalance, where crediting an account increases its balance.
	CodeDeposit          uint16 = 101 // EXTERNAL (SystemAccountID) -> account
	CodeWithdrawal       uint16 = 102 // account -> EXTERNAL (SystemAccountID)
	CodeTransfer         uint16 = 103 // account -> another user's account
	CodeInternalTransfer uint16 = 104 // account -> the same user's other account
)

// TransferTypeName maps a transfer Code back to the data.json-style type
// string, for API responses that surface transfer history.
func TransferTypeName(code uint16) string {
	switch code {
	case CodeInitialBalance:
		return "initial_balance"
	case CodeDeposit:
		return "deposit"
	case CodeWithdrawal:
		return "withdrawal"
	case CodeTransfer:
		return "transfer"
	case CodeInternalTransfer:
		return "internal_transfer"
	default:
		return "unknown"
	}
}
