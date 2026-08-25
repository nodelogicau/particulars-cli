package query

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// CodeScopeWiderThanInputs is the condition's name in the DKF specification:
// a synthesis is shareable more widely than an assertion it reasons from, so
// its content can carry that assertion's substance past the boundary that
// withholds it.
//
// The comparison is between *effective* scopes on both sides, which matters in
// both directions: an input promoted to match the synthesis is no longer a
// concern, while a synthesis promoted past inputs that were never widened is.
// That also means the condition belongs to workspace state rather than to the
// synthesis file — a promotion can create it, or clear it, without either file
// changing.
const CodeScopeWiderThanInputs = "scope_wider_than_inputs"

// ScopeWiderThanInputs reports the condition for one synthesis, or nil. It is
// the single implementation behind all three evaluation points the spec names:
// when a synthesis is created, when a promotion is written, and during
// validation. It never returns an error: a wider synthesis is permitted, and
// whether its prose actually discloses its inputs is a judgement for whoever
// reviews the change.
func ScopeWiderThanInputs(g *store.Graph, s *dkf.Synthesis) string {
	if s == nil || s.Retracted != nil {
		return ""
	}
	mine := g.EffectiveScope(s.ID)
	var narrower []string
	for _, in := range s.Inputs {
		child := g.Assertion(in.ID)
		if child == nil {
			continue
		}
		if theirs := g.EffectiveScope(in.ID); dkf.ScopeRank(theirs) < dkf.ScopeRank(mine) {
			narrower = append(narrower, in.ID+" ("+describeScope(g, in.ID, theirs)+")")
		}
	}
	if len(narrower) == 0 {
		return ""
	}
	sort.Strings(narrower)
	return fmt.Sprintf(
		"synthesis %s is %s but reasons from narrower input(s) %s; its content can carry their substance somewhere those inputs are withheld",
		s.ID, describeScope(g, s.ID, mine), strings.Join(narrower, ", "))
}

// describeScope names a scope and, when a promotion is why it is that wide,
// the record responsible — so a reader is sent to the file that caused the
// condition rather than to the object, which may not have changed.
func describeScope(g *store.Graph, id string, effective dkf.Scope) string {
	a := g.Assertion(id)
	if a == nil || a.GetContext().Scope == effective {
		return string(effective)
	}
	prs := g.PromotionsFor(id)
	for _, pr := range prs {
		if pr.Scope == effective {
			return fmt.Sprintf("%s, promoted from %s by %s", effective, a.GetContext().Scope, pr.ID)
		}
	}
	return fmt.Sprintf("%s, promoted from %s", effective, a.GetContext().Scope)
}

// QuoteDisclosuresForPromotion names the promoted objects that carry a
// verbatim quote. A synthesis summarises its inputs; a quote reproduces its
// source completely, so widening a quoted claim publishes that source text in
// full. Reported, never refused: whether the quoted material may travel is the
// promoter's judgement, but they should be told they are making it.
func QuoteDisclosuresForPromotion(g *store.Graph, pr *dkf.Promotion) []string {
	if pr == nil {
		return nil
	}
	var out []string
	for _, id := range pr.Claims {
		a := g.Assertion(id)
		if a == nil {
			continue
		}
		if doc := a.GetSource().Document; doc.Quote != "" {
			out = append(out, fmt.Sprintf(
				"%s carries a verbatim quote from %s, so promoting it to %s publishes that source text in full",
				id, doc.Ref, pr.Scope))
		}
	}
	sort.Strings(out)
	return out
}

// ScopeFindingsForPromotion returns the condition for every synthesis a
// promotion could have changed: the syntheses it promoted, and every
// non-retracted synthesis citing anything it covers — because promoting an
// input can clear the condition on a synthesis nobody named, and promoting a
// synthesis can create it.
func ScopeFindingsForPromotion(g *store.Graph, pr *dkf.Promotion) []string {
	if pr == nil {
		return nil
	}
	covered := map[string]bool{}
	for _, id := range pr.Claims {
		covered[id] = true
	}
	var out []string
	for _, a := range g.SortedAssertions() {
		s, ok := a.(*dkf.Synthesis)
		if !ok || s.Retracted != nil {
			continue
		}
		relevant := covered[s.ID]
		for _, in := range s.Inputs {
			if covered[in.ID] {
				relevant = true
			}
		}
		if !relevant {
			continue
		}
		if msg := ScopeWiderThanInputs(g, s); msg != "" {
			out = append(out, msg)
		}
	}
	return out
}
