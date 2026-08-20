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

	// Field-level problems and canonical form.
	for _, obj := range g.Objects() {
		path := g.Files[obj.ObjectID()]
		for _, p := range dkf.ValidateObject(obj) {
			add(SeverityError, path, p.Code, p.Error())
		}
		if canon, err := dkf.Encode(obj); err == nil && !bytes.Equal(canon, g.Raw[obj.ObjectID()]) {
			add(SeverityWarning, path, CodeNonCanonical, "file differs from canonical serialisation; rewrite is not required but diffs will be noisier")
		}
	}

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
		stale := false
		for _, in := range s.Inputs {
			child := g.Assertion(in.ID)
			if child == nil {
				if dkf.IsAssertionID(in.ID) { // particular-typed ids are already reported as invalid_id
					add(SeverityError, path, CodeDanglingReference, fmt.Sprintf("input %s does not exist", in.ID))
				}
				continue
			}
			if child.GetRetracted() != nil {
				stale = true
			}
		}
		if stale && s.Retracted == nil {
			add(SeverityWarning, path, CodeStaleSynthesis, fmt.Sprintf("synthesis %s cites a retracted input; consider re-synthesis", s.ID))
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
