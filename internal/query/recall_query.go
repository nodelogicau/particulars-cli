package query

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"

	"github.com/nodelogicau/particulars-cli/internal/apperr"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// DefaultQueryLimit is the number of particulars recall by query keeps when
// no limit is given: short terms such as "is" resolve nearly every particular.
const DefaultQueryLimit = 10

// Resolution says how a particular was resolved by a query, per DKF
// recall-query, best first.
type Resolution string

// The resolution values the baseline produces. A semantic stage would add
// "semantic" after these.
const (
	ResolutionLabel   Resolution = "label"
	ResolutionAlias   Resolution = "alias"
	ResolutionLexical Resolution = "lexical"
)

var resolutionRank = map[Resolution]int{ResolutionLabel: 0, ResolutionAlias: 1, ResolutionLexical: 2}

// QueryParticular is one resolved merge class and what recall by particular
// returns for it.
type QueryParticular struct {
	Particular string     `json:"particular"` // lowest id in the class
	Label      string     `json:"label"`
	Resolution Resolution `json:"resolution"`
	Class      []string   `json:"class,omitempty"`
	Entries    []Entry    `json:"entries"`
}

// QueryResult is the response of recall by query.
type QueryResult struct {
	Query       string            `json:"query"`
	Particulars []QueryParticular `json:"particulars"`
	Count       int               `json:"count"`
	Omitted     int               `json:"omitted"`
	Author      string            `json:"author,omitempty"` // the author filter's particular, when given
}

// isEdge reports whether r is trimmed from the edges of the query and its
// terms: Unicode general category P. It is the one place the edge class is
// defined (nodelogicau/particulars#37 may add White_Space).
func isEdge(r rune) bool { return unicode.IsPunct(r) }

// parsedQuery is a query as the baseline matches it: everything case-folded.
type parsedQuery struct {
	raw   string   // the whole query folded, untrimmed (for exact matching)
	whole string   // the whole query trimmed, then folded
	terms []string // distinct trimmed, folded terms
}

func (q parsedQuery) empty() bool { return q.whole == "" }

// parseQuery splits text into terms on White_Space, trims edge characters
// from each term and from the whole query as written, drops empty terms, and
// folds. Terms are deduplicated after folding.
func parseQuery(text string, fold func(string) string) parsedQuery {
	q := parsedQuery{raw: fold(text), whole: fold(strings.TrimFunc(text, isEdge))}
	seen := map[string]bool{}
	for _, t := range strings.FieldsFunc(text, func(r rune) bool { return unicode.Is(unicode.White_Space, r) }) {
		t = fold(strings.TrimFunc(t, isEdge))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		q.terms = append(q.terms, t)
	}
	return q
}

// newFolder returns full Unicode case folding. A Caser is stateful, so each
// call gets its own.
func newFolder() func(string) string {
	c := cases.Fold()
	return func(s string) string { return c.String(s) }
}

// matches reports whether a folded text contains the whole query or any term.
func (q parsedQuery) matches(text string) bool {
	if q.empty() {
		return false
	}
	if strings.Contains(text, q.whole) {
		return true
	}
	for _, t := range q.terms {
		if strings.Contains(text, t) {
			return true
		}
	}
	return false
}

// exactIn reports whether the folded name occurs in the folded query such
// that every character of the query outside that occurrence is an edge
// character. Every occurrence is tried, overlapping ones included.
func exactIn(query, name string) bool {
	if name == "" {
		return false
	}
	allEdge := func(s string) bool {
		for _, r := range s {
			if !isEdge(r) {
				return false
			}
		}
		return true
	}
	for from := 0; from <= len(query)-len(name); {
		i := strings.Index(query[from:], name)
		if i < 0 {
			return false
		}
		at := from + i
		if allEdge(query[:at]) && allEdge(query[at+len(name):]) {
			return true
		}
		_, size := utf8.DecodeRuneInString(query[at:])
		from = at + size
	}
	return false
}

// candidate is a class resolved in stage one.
type candidate struct {
	root       string
	resolution Resolution
}

// resolveQuery is stage one: the lexical baseline over every particular's
// label and aliases and every claim's and synthesis's content, ignoring all
// filters. Each hit is taken to its merge class, keeping the best resolution.
// A semantic stage would append candidates here and never remove any.
func resolveQuery(g *store.Graph, q parsedQuery, fold func(string) string) []candidate {
	if q.empty() {
		return nil
	}
	best := map[string]Resolution{}
	note := func(id string, r Resolution) {
		root := g.ClassOf(id)[0]
		if cur, ok := best[root]; !ok || resolutionRank[r] < resolutionRank[cur] {
			best[root] = r
		}
	}
	for _, p := range g.Particulars {
		label := fold(p.Label)
		if exactIn(q.raw, label) {
			note(p.ID, ResolutionLabel)
			continue
		}
		res := Resolution("")
		lexical := q.matches(label)
		for _, a := range p.Aliases {
			fa := fold(a)
			if exactIn(q.raw, fa) {
				res = ResolutionAlias
				break
			}
			lexical = lexical || q.matches(fa)
		}
		if res == "" && lexical {
			res = ResolutionLexical
		}
		if res != "" {
			note(p.ID, res)
		}
	}
	for _, a := range g.Assertions {
		if q.matches(fold(a.GetContent())) {
			note(a.SubjectID(), ResolutionLexical)
		}
	}
	out := make([]candidate, 0, len(best))
	for root, r := range best {
		out = append(out, candidate{root, r})
	}
	return out
}

// RecallQuery implements recall by query: stage one resolves the query to
// merge classes by the lexical baseline; stage two returns, for each, what
// Recall returns for that class under opts. opts.Subject must be empty;
// opts.Limit counts particulars (0 means DefaultQueryLimit).
func RecallQuery(g *store.Graph, text string, opts RecallOptions) (QueryResult, error) {
	if opts.Limit < 0 {
		return QueryResult{}, apperr.Usage("limit must not be negative")
	}
	limit := opts.Limit
	if limit == 0 {
		limit = DefaultQueryLimit
	}
	fold := newFolder()
	q := parseQuery(text, fold)
	res := QueryResult{Query: text, Particulars: []QueryParticular{}, Author: opts.Author}

	type ranked struct {
		QueryParticular
		whole bool
		terms int
	}
	var all []ranked
	for _, c := range resolveQuery(g, q, fold) {
		o := opts
		o.Subject, o.Limit = c.root, 0
		entries := Recall(g, o)
		if len(entries) == 0 {
			continue
		}
		var texts []string
		members := g.ClassOf(c.root)
		for _, m := range members {
			if p := g.Particular(m); p != nil {
				texts = append(texts, fold(p.Label))
				for _, a := range p.Aliases {
					texts = append(texts, fold(a))
				}
			}
		}
		for i := range entries {
			ft := fold(entries[i].Content)
			entries[i].Matched = q.matches(ft)
			texts = append(texts, ft)
		}
		r := ranked{QueryParticular: QueryParticular{Particular: c.root, Resolution: c.resolution, Entries: entries}}
		if p := g.Particular(c.root); p != nil {
			r.Label = p.Label
		}
		if len(members) > 1 {
			r.Class = members
		}
		for _, t := range texts {
			if strings.Contains(t, q.whole) {
				r.whole = true
				break
			}
		}
		for _, term := range q.terms {
			for _, t := range texts {
				if strings.Contains(t, term) {
					r.terms++
					break
				}
			}
		}
		all = append(all, r)
	}
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if ra, rb := resolutionRank[a.Resolution], resolutionRank[b.Resolution]; ra != rb {
			return ra < rb
		}
		if a.whole != b.whole {
			return a.whole
		}
		if a.terms != b.terms {
			return a.terms > b.terms
		}
		return a.Particular < b.Particular
	})
	if len(all) > limit {
		res.Omitted = len(all) - limit
		all = all[:limit]
	}
	for _, r := range all {
		res.Particulars = append(res.Particulars, r.QueryParticular)
	}
	res.Count = len(res.Particulars)
	return res, nil
}
