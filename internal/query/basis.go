package query

import (
	"fmt"

	"github.com/nodelogicau/particulars-cli/internal/apperr"
	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// CheckBasis applies the write-time rules for a requested basis, shared by
// `claim assert` and MCP claim_assert; field names the parameter in messages.
// It returns the warnings the write should carry: stale_claim for each
// retracted member — citing one is legal, and the claim is stale from the
// start, which is the honest report — and inferred_without_basis for an
// inferred claim that names none, which is still written: refusing would push
// authors toward held, which is worse.
//
// Ids are never resolved from labels: a basis names exact objects, as
// promotion does.
func CheckBasis(g *store.Graph, ev dkf.Evidential, field string, ids []string) ([]string, error) {
	if len(ids) == 0 {
		if ev == dkf.EvidentialInferred {
			return []string{CodeInferredWithoutBasis + ": the claim is inferred but names no basis; pass the claims and syntheses it was inferred from so its warrant can be walked and it goes stale when they are retracted"}, nil
		}
		return nil, nil
	}
	if ev != dkf.EvidentialInferred {
		return nil, apperr.Usage("%s is only for an inferred claim: an observed claim's evidence is its document, and a held claim has none", field)
	}
	var warnings []string
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return nil, apperr.Usage("%s lists %s more than once", field, id)
		}
		seen[id] = true
		if !dkf.IsAssertionID(id) {
			return nil, apperr.Usage("%s %q is not a claim or synthesis id; a basis names objects by id, never by label", field, id)
		}
		a := g.Assertion(id)
		if a == nil {
			return nil, apperr.NotFound("%s %s does not exist", field, id)
		}
		if a.GetRetracted() != nil {
			warnings = append(warnings, fmt.Sprintf("%s: basis member %s is retracted, so this claim is stale from the start", CodeStaleClaim, id))
		}
	}
	return warnings, nil
}
