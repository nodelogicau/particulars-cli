package dkf

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Identifier prefixes per object type.
const (
	PrefixParticular = "par"
	PrefixClaim      = "clm"
	PrefixSynthesis  = "syn"
	PrefixMerge      = "mrg"
	PrefixPublish    = "pub"
)

var (
	// lenientID accepts any prefixed opaque identifier so that workspaces
	// written by other implementations remain readable.
	lenientID = regexp.MustCompile(`^(par|clm|syn|mrg|pub)_[A-Za-z0-9-]+$`)
	// canonicalID is what this implementation mints: prefix + lowercase UUIDv7.
	canonicalID = regexp.MustCompile(`^(par|clm|syn|mrg|pub)_[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// Prefix returns the identifier prefix for an object type.
func Prefix(t Type) string {
	switch t {
	case TypeParticular:
		return PrefixParticular
	case TypeClaim:
		return PrefixClaim
	case TypeSynthesis:
		return PrefixSynthesis
	case TypeMerge:
		return PrefixMerge
	case TypePublish:
		return PrefixPublish
	}
	return ""
}

// NewUUID mints a lowercase canonical UUIDv7. google/uuid's NewV7 uses a
// per-millisecond sequence in rand_a, so ids minted by one process within the
// same millisecond remain lexically ordered by creation.
func NewUUID() string {
	u, err := uuid.NewV7()
	if err != nil {
		// NewV7 only fails if the random source fails; treat as fatal.
		panic(fmt.Sprintf("dkf: uuid v7: %v", err))
	}
	return strings.ToLower(u.String())
}

// uuidForm is the hyphenated 36-character UUID string form, in either case.
var uuidForm = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`)

// MonthOf returns the UTC calendar month (YYYY-MM) of the minting instant in
// id's UUIDv7, the shard a dkf/0.2 workspace files it under. It reports false
// when no month derives — an identifier part that is not a hyphenated UUID of
// version 7 and RFC 4122 variant — and such an id lives under legacy/. The
// instant is the UUID's first 48 bits, Unix milliseconds.
func MonthOf(id string) (string, bool) {
	t, ok := MintedAt(id)
	if !ok {
		return "", false
	}
	return t.Format("2006-01"), true
}

// MintedAt returns the minting instant encoded in id's UUIDv7, in UTC, or
// false when the id carries none (see MonthOf).
func MintedAt(id string) (time.Time, bool) {
	_, part, found := strings.Cut(id, "_")
	if !found || !uuidForm.MatchString(part) {
		return time.Time{}, false
	}
	u, err := uuid.Parse(part)
	if err != nil || u.Version() != 7 || u.Variant() != uuid.RFC4122 {
		return time.Time{}, false
	}
	ms := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 | int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5])
	return time.UnixMilli(ms).UTC(), true
}

// NewID mints a new identifier for the given object type.
func NewID(t Type) string {
	return Prefix(t) + "_" + NewUUID()
}

// TypeOfID returns the object type implied by an identifier's prefix, using
// the lenient grammar. It errors on malformed or unknown-prefix identifiers.
func TypeOfID(id string) (Type, error) {
	m := lenientID.FindStringSubmatch(id)
	if m == nil {
		return "", fmt.Errorf("invalid identifier %q", id)
	}
	switch m[1] {
	case PrefixParticular:
		return TypeParticular, nil
	case PrefixClaim:
		return TypeClaim, nil
	case PrefixSynthesis:
		return TypeSynthesis, nil
	case PrefixMerge:
		return TypeMerge, nil
	case PrefixPublish:
		return TypePublish, nil
	}
	return "", fmt.Errorf("invalid identifier %q", id)
}

// IsValidID reports whether id satisfies the lenient grammar.
func IsValidID(id string) bool { return lenientID.MatchString(id) }

// IsCanonicalID reports whether id is one this implementation would mint.
func IsCanonicalID(id string) bool { return canonicalID.MatchString(id) }

// IsAssertionID reports whether id names a claim or synthesis.
func IsAssertionID(id string) bool {
	t, err := TypeOfID(id)
	return err == nil && (t == TypeClaim || t == TypeSynthesis)
}

// IsRetractableID reports whether id names a claim, synthesis, merge, or
// promotion — everything that carries a retracted block.
func IsRetractableID(id string) bool {
	t, err := TypeOfID(id)
	return err == nil && (t == TypeClaim || t == TypeSynthesis || t == TypeMerge || t == TypePublish)
}
