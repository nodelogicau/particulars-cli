package dkf

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Identifier prefixes per object type.
const (
	PrefixParticular = "par"
	PrefixClaim      = "clm"
	PrefixSynthesis  = "syn"
	PrefixMerge      = "mrg"
)

var (
	// lenientID accepts any prefixed opaque identifier so that workspaces
	// written by other implementations remain readable.
	lenientID = regexp.MustCompile(`^(par|clm|syn|mrg)_[A-Za-z0-9-]+$`)
	// canonicalID is what this implementation mints: prefix + lowercase UUIDv7.
	canonicalID = regexp.MustCompile(`^(par|clm|syn|mrg)_[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
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

// IsRetractableID reports whether id names a claim, synthesis, or merge.
func IsRetractableID(id string) bool {
	t, err := TypeOfID(id)
	return err == nil && (t == TypeClaim || t == TypeSynthesis || t == TypeMerge)
}
