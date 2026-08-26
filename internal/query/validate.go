package query

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Finding severities.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	// SeverityInfo records something a reader may want to know that is not a
	// defect: provenance that could not be machine-checked, for instance.
	// Notes never affect the exit code.
	SeverityInfo = "info"
)

// Finding codes beyond the field-level ones in package dkf.
const (
	CodeParseError        = "parse_error"
	CodeIDMismatch        = "id_mismatch"
	CodeTypeMismatch      = "type_mismatch"
	CodeDanglingReference = "dangling_reference"
	CodeCycle             = "cycle"
	CodeDuplicateURI      = "duplicate_uri"
	CodeIndexStale        = "index_stale"
	CodeIndexMissing      = "index_missing"
	CodeStaleSynthesis    = "stale_synthesis"
	CodeOrphanParticular  = "orphan_particular"
	CodeNonCanonical      = "non_canonical"
	CodeLegacyProducedBy  = "legacy_produced_by"
	CodeLegacyID          = "legacy_id"
	CodeUnknownMergeURI   = "unknown_merge_uri"
	CodeDuplicateMerge    = "duplicate_merge"
	// CodePromotionOfRetracted: a live promotion covers a withdrawn object. It
	// grants nothing, but it usually means the retraction came after.
	CodePromotionOfRetracted = "promotion_of_retracted"
	// CodeDuplicatePromotion: two live promotions grant the same object the
	// same scope. Valid — promotion may only widen, and this widens nothing —
	// but redundant.
	CodeDuplicatePromotion = "duplicate_promotion"
	// CodeUndeclared: the claim predates the evidential. Not a fourth value
	// and not a defect — the warrant cannot now be established, and claims
	// are immutable, so the distinction ages out rather than being migrated.
	CodeUndeclared = "undeclared"
	// CodeConfidenceOnUndeclared: a probability whose meaning cannot be
	// established, because the warrant it would grade is unknown.
	CodeConfidenceOnUndeclared = "confidence_on_undeclared"
	// CodeUnknownMethod: a synthesis method outside the closed vocabulary.
	CodeUnknownMethod = "unknown_method"
	// CodeQuotedSource: a claim reproduces source text verbatim and is shared
	// more widely than the author's own notes.
	CodeQuotedSource = "quoted_source"
	// CodeLegacyDocumentURI: a document mapping was read from the pre-rename
	// `uri` key. The file can never be rewritten, so a reader accepts it in
	// perpetuity and this is the only way anyone learns it is there.
	CodeLegacyDocumentURI = "legacy_document_uri"
	// CodeDefectUnverifiable: a retraction declares the claim misread its
	// source, and that source has since changed — so the text the claim is
	// said to have misread is no longer the text a reviewer can read.
	CodeDefectUnverifiable = "defect_unverifiable"
	CodeInvalidBaseURI     = "invalid_base_uri"
)

// Finding is one validation result.
type Finding struct {
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// Findings is a sortable, summarisable list.
type Findings []Finding

// HasErrors reports whether any finding is an error.
func (fs Findings) HasErrors() bool {
	for _, f := range fs {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Validate checks the whole workspace. It never writes.
// IsCorpusFact reports whether a finding code records a fact about the corpus
// rather than a finding about an object someone might act on. A corpus fact is
// permanent — the files carrying it can never be rewritten, since appending a
// retraction is the only modification the format permits — so it can never be
// cleared, and its discovery value is spent the first time anyone sees it,
// while a per-object listing recurs on every run forever. Renderers report
// these in aggregate: one line carrying a count discovers exactly as well as
// eighty-eight. Info-severity findings are corpus facts by construction and
// are aggregated by severity rather than being listed here.
func IsCorpusFact(code string) bool {
	switch code {
	case CodeLegacyProducedBy, CodeLegacyID, CodeLegacyDocumentURI:
		return true
	}
	return false
}

func Validate(w *store.Workspace) (Findings, error) {
	g, err := w.Load()
	if err != nil {
		return nil, err
	}
	var fs Findings
	add := func(sev, path, code, msg string) {
		fs = append(fs, Finding{Severity: sev, Path: path, Code: code, Message: msg})
	}

	// Files that could not be loaded at all.
	for _, p := range g.Problems {
		add(SeverityError, p.Path, p.Code, p.Message)
	}

	// Field-level problems, legacy forms, and canonical form.
	for _, obj := range g.Objects() {
		path := g.Files[obj.ObjectID()]
		for _, p := range dkf.ValidateObject(obj) {
			add(SeverityError, path, p.Code, p.Error())
		}
		if !dkf.IsCanonicalID(obj.ObjectID()) {
			add(SeverityWarning, path, CodeLegacyID, fmt.Sprintf("id %s is not a canonical UUIDv7; it was written by another implementation", obj.ObjectID()))
		}
		if c, ok := obj.(*dkf.Claim); ok && c.Evidential == "" {
			add(SeverityInfo, path, CodeUndeclared, "warrant undeclared: the claim predates the evidential field and cannot be backfilled")
			if c.Confidence != nil {
				add(SeverityInfo, path, CodeConfidenceOnUndeclared, "confidence on an undeclared claim cannot be interpreted: the warrant it would grade is unknown")
			}
		}
		if syn, ok := obj.(*dkf.Synthesis); ok && syn.Method != "" && !dkf.ValidMethod(syn.Method) {
			add(SeverityWarning, path, CodeUnknownMethod, fmt.Sprintf("method %q is outside the vocabulary (reconciliation, qualification, positions); read leniently", syn.Method))
		}
		if s, ok := obj.(*dkf.Synthesis); ok && s.LegacyProducedBy {
			add(SeverityWarning, path, CodeLegacyProducedBy, "provenance was read from a legacy produced-by block; new syntheses write source")
			continue // its bytes necessarily differ from the canonical form
		}
		if a, ok := obj.(dkf.Assertion); ok && a.GetSource().Document.LegacyURI() {
			// Reported once, by the finding that names the cause: a legacy
			// `uri` key necessarily differs from the canonical form, and
			// saying so twice tells the reader nothing the first said.
			add(SeverityWarning, path, CodeLegacyDocumentURI,
				"document was read from a legacy `uri` key; new documents write `ref`, which also holds a workspace path or an unfetchable source")
			continue
		}
		if canon, err := dkf.Encode(obj); err == nil && !bytes.Equal(canon, g.Raw[obj.ObjectID()]) {
			add(SeverityWarning, path, CodeNonCanonical, "file differs from canonical serialisation; rewrite is not required but diffs will be noisier")
		}
	}

	// Merge records: unknown URIs and duplicate pairs.
	pairs := map[[2]string][]string{}
	for _, m := range g.SortedMerges() {
		if m.Retracted != nil || len(m.URIs) != 2 {
			continue
		}
		path := g.Files[m.ID]
		for _, u := range m.URIs {
			if g.ParticularByURI(u) == nil {
				add(SeverityWarning, path, CodeUnknownMergeURI, fmt.Sprintf("uri %q has no local particular; the merge bridges to another source", u))
			}
		}
		a, b := m.URIs[0], m.URIs[1]
		if a > b {
			a, b = b, a
		}
		pairs[[2]string{a, b}] = append(pairs[[2]string{a, b}], m.ID)
	}
	// Promotions: the record's own fields are checked by dkf.ValidatePromotion
	// on load; here we check what needs the workspace.
	type promoKey struct{ id, scope string }
	promoted := map[promoKey][]string{}
	for _, pr := range g.SortedPromotions() {
		if pr.Retracted != nil {
			continue
		}
		path := g.Files[pr.ID]
		for _, id := range pr.Claims {
			a := g.Assertion(id)
			if a == nil {
				if dkf.IsAssertionID(id) {
					add(SeverityError, path, CodeDanglingReference, fmt.Sprintf("promoted object %s does not exist", id))
				}
				continue
			}
			// Widen-only is measured against the ASSERTED scope, so that a
			// record's validity never depends on what else has been written.
			if asserted := a.GetContext().Scope; dkf.ScopeRank(pr.Scope) < dkf.ScopeRank(asserted) {
				add(SeverityError, path, dkf.CodeInvalidPromotion, fmt.Sprintf(
					"promotion may only widen: %s is asserted %s, wider than this record's %s", id, asserted, pr.Scope))
			}
			if a.GetRetracted() != nil {
				add(SeverityWarning, path, CodePromotionOfRetracted, fmt.Sprintf(
					"%s is retracted, so this promotion grants nothing; retract the promotion too if that was intended", id))
			}
			k := promoKey{id, string(pr.Scope)}
			promoted[k] = append(promoted[k], pr.ID)
		}
	}
	for k, ids := range promoted {
		if len(ids) > 1 {
			for _, id := range ids {
				add(SeverityWarning, g.Files[id], CodeDuplicatePromotion, fmt.Sprintf(
					"%s is promoted to %s by %d records: %v", k.id, k.scope, len(ids), ids))
			}
		}
	}

	for pair, ids := range pairs {
		if len(ids) > 1 {
			for _, id := range ids {
				add(SeverityWarning, g.Files[id], CodeDuplicateMerge, fmt.Sprintf("uris %q and %q are joined by %d merges: %v", pair[0], pair[1], len(ids), ids))
			}
		}
	}

	// Provenance. Verification is offline: a document is checked only when it
	// resolves to a file in this workspace, and everything else is reported as
	// not checked rather than as a problem.
	for _, a := range g.SortedAssertions() {
		doc := a.GetSource().Document
		if doc.IsZero() {
			continue
		}
		path := g.Files[a.ObjectID()]
		// Retracted objects are checked too: the unverifiable-defect finding
		// below is about the retraction, not about a live claim.
		st := VerifyDocument(w, doc)
		retracted := a.GetRetracted()
		switch {
		case st.Code == "":
		case st.Code == CodeUnverifiedDocument:
			add(SeverityInfo, path, st.Code, st.Message)
		case retracted != nil:
			// Drift under a withdrawn claim is an observation, not something
			// to act on — the claim is already gone.
			add(SeverityInfo, path, st.Code, st.Message)
		default:
			add(SeverityWarning, path, st.Code, st.Message)
		}
		// Drift is reported alongside a declared kind, never checked against
		// it — except in the one direction that is a statement about what can
		// be checked rather than a guess at intent.
		if retracted != nil && retracted.Kind == dkf.KindDefect && st.Drifted() {
			add(SeverityInfo, path, CodeDefectUnverifiable, fmt.Sprintf(
				"retracted as a defect, but %s has changed since the claim was written, so the text it is said to have misread is no longer the text a reviewer can read",
				doc.Ref))
		}
		if retracted != nil {
			continue
		}
		// A quote reproduces its source completely, where a synthesis only
		// summarises. Say so where the exposure is wider than the author's
		// own notes, so a reviewer weighing scope can see it.
		if doc.Quote != "" && dkf.ScopeRank(g.EffectiveScope(a.ObjectID())) > dkf.ScopeRank(dkf.ScopePersonal) {
			add(SeverityInfo, path, CodeQuotedSource, fmt.Sprintf(
				"carries a verbatim quote from %s and is %s, so that source text is disclosed in full",
				doc.Ref, g.EffectiveScope(a.ObjectID())))
		}
	}

	// Deliberately absent: the spec SHOULDs cross-checking retracted.kind
	// against document drift — a supersession over an unchanged hash being
	// "suspect". Drift is a signal about the source joint and supersession is
	// the world joint, so the check only holds for documents that describe
	// current state. Fifteen claims in the reference workspace cite
	// commit-pinned URLs whose hash cannot change by construction, and every
	// one would be flagged the day it is superseded — penalising the best
	// sourcing discipline available. Raised on particulars-cli#3; awaiting a
	// ruling. If one is wanted, the sound direction is the opposite: a defect
	// against a drifted document is unverifiable, not suspect.

	// Referential integrity.
	uriOwners := map[string][]string{}
	for _, p := range g.SortedParticulars() {
		uriOwners[p.URI] = append(uriOwners[p.URI], p.ID)
		if len(g.BySubject[p.ID]) == 0 {
			add(SeverityWarning, g.Files[p.ID], CodeOrphanParticular, fmt.Sprintf("particular %s has no claims", p.ID))
		}
	}
	for uri, ids := range uriOwners {
		if len(ids) > 1 {
			sort.Strings(ids)
			for _, id := range ids {
				add(SeverityError, g.Files[id], CodeDuplicateURI, fmt.Sprintf("uri %q is shared by %v", uri, ids))
			}
		}
	}
	staleMemo := map[string]bool{}
	for _, a := range g.SortedAssertions() {
		path := g.Files[a.ObjectID()]
		if a.SubjectID() != "" && g.Particular(a.SubjectID()) == nil {
			add(SeverityError, path, CodeDanglingReference, fmt.Sprintf("subject %s does not exist", a.SubjectID()))
		}
		if r := a.GetRetracted(); r != nil && r.SupersededBy != "" && g.Assertion(r.SupersededBy) == nil {
			add(SeverityError, path, CodeDanglingReference, fmt.Sprintf("retracted.superseded-by %s does not exist", r.SupersededBy))
		}
		s, ok := a.(*dkf.Synthesis)
		if !ok {
			continue
		}
		for _, in := range s.Inputs {
			if g.Assertion(in.ID) == nil && dkf.IsAssertionID(in.ID) { // other ids are already reported as invalid_id
				add(SeverityError, path, CodeDanglingReference, fmt.Sprintf("input %s does not exist", in.ID))
			}
		}
		if msg := ScopeWiderThanInputs(g, s); msg != "" {
			add(SeverityWarning, path, CodeScopeWiderThanInputs, msg)
		}
		if s.Retracted == nil && CitesRetracted(g, s, staleMemo) {
			add(SeverityWarning, path, CodeStaleSynthesis, fmt.Sprintf("synthesis %s cites a retracted input (directly or transitively); consider re-synthesis", s.ID))
		}
	}

	// Cycles.
	for _, id := range findCycles(g) {
		add(SeverityError, g.Files[id], CodeCycle, fmt.Sprintf("%s participates in an input cycle", id))
	}

	// Index consistency.
	diff, err := w.CheckIndex()
	if err != nil {
		return nil, err
	}
	if _, _, rerr := w.ReadIndex(); errors.Is(rerr, store.ErrNotFound) {
		add(SeverityWarning, store.IndexFile, CodeIndexMissing, "index.yaml is absent; run `particulars index`")
	} else if !diff.Clean() {
		add(SeverityError, store.IndexFile, CodeIndexStale, fmt.Sprintf("index.yaml differs from a rebuild (missing %d, extra %d, changed %d); run `particulars index`", len(diff.Missing), len(diff.Extra), len(diff.Changed)))
	}

	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].Path != fs[j].Path {
			return fs[i].Path < fs[j].Path
		}
		return fs[i].Code < fs[j].Code
	})
	return fs, nil
}

// findCycles returns the ids of syntheses that lie on an input cycle.
func findCycles(g *store.Graph) []string {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	colour := map[string]int{}
	onCycle := map[string]bool{}
	var visit func(id string, stack []string)
	visit = func(id string, stack []string) {
		colour[id] = grey
		stack = append(stack, id)
		if s, ok := g.Assertion(id).(*dkf.Synthesis); ok {
			for _, in := range s.Inputs {
				switch colour[in.ID] {
				case white:
					if g.Assertion(in.ID) != nil {
						visit(in.ID, stack)
					}
				case grey:
					for i := len(stack) - 1; i >= 0; i-- {
						onCycle[stack[i]] = true
						if stack[i] == in.ID {
							break
						}
					}
				}
			}
		}
		colour[id] = black
	}
	for _, a := range g.SortedAssertions() {
		if colour[a.ObjectID()] == white {
			visit(a.ObjectID(), nil)
		}
	}
	out := make([]string, 0, len(onCycle))
	for id := range onCycle {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
