// Package graph renders a DKF workspace as Microsoft Graph connector payloads
// so that merged, organisation-scoped knowledge can be indexed by Microsoft 365
// Copilot. It emits only: nothing here talks to Microsoft, and no Graph SDK or
// authentication library enters this codebase.
//
// One item is produced per particular, carrying the current belief rather than
// individual claims — a flat index of claims has no notion of current belief
// (in DKF that is computed, never stored), so it could surface a retracted or
// superseded claim as though it stood.
package graph

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// ACL is one access control entry on an external item.
type ACL struct {
	Type       string `json:"type"`
	Value      string `json:"value"`
	AccessType string `json:"accessType"`
}

// Content is the item body Microsoft indexes semantically.
type Content struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

// Item is a microsoft.graph.externalConnectors.externalItem payload. Property
// order is fixed by marshalling an ordered map so output is deterministic.
type Item struct {
	ACL        []ACL      `json:"acl"`
	Properties Properties `json:"properties"`
	Content    Content    `json:"content"`
}

// Line is one NDJSON record: the item and the id it is PUT under.
type Line struct {
	ID   string `json:"id"`
	Item Item   `json:"item"`
}

// grantEveryone is the ACL for organisation- and public-scoped knowledge.
func grantEveryone() []ACL {
	return []ACL{{Type: "everyone", Value: "everyone", AccessType: "grant"}}
}

// Options control an export.
type Options struct {
	// SourceURL, when set, is joined with a workspace-relative path to form
	// the item's url property (e.g. a GitHub blob base).
	SourceURL string
	// Scope narrows the export further. Empty means organisation and public.
	// Personal is never exported and is rejected by the caller.
	Scope dkf.Scope
}

// exportable reports whether an assertion may leave the workspace: never
// retracted, never personal, and within Scope when one is given. Scope here is
// the EFFECTIVE scope — asserted, widened by any non-retracted promotion — so
// a workspace written entirely at personal scope becomes exportable by
// promotion, without re-asserting anything.
func (o Options) exportable(g *store.Graph, a dkf.Assertion) bool {
	if a.GetRetracted() != nil {
		return false
	}
	sc := g.EffectiveScope(a.ObjectID())
	if sc == dkf.ScopePersonal || sc == "" {
		return false
	}
	if o.Scope != "" && sc != o.Scope {
		return false
	}
	return true
}

// Build renders one item per particular with exportable assertions, ordered by
// particular id.
func Build(g *store.Graph, ws *store.Workspace, o Options) []Line {
	var out []Line
	for _, p := range g.SortedParticulars() {
		// Only the particular's own assertions: merge classes would attribute
		// another particular's claims to this item's title and url.
		var kept []dkf.Assertion
		for _, a := range g.BySubject[p.ID] {
			if o.exportable(g, a) {
				kept = append(kept, a)
			}
		}
		if len(kept) == 0 {
			continue
		}
		out = append(out, Line{ID: p.ID, Item: item(g, ws, p, kept, o)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// belief picks the current synthesis among the exportable assertions and the
// set reconciled into it.
func belief(g *store.Graph, kept []dkf.Assertion) (cur *dkf.Synthesis, reconciled map[string]bool) {
	reconciled = map[string]bool{}
	for _, a := range kept {
		s, ok := a.(*dkf.Synthesis)
		if !ok {
			continue
		}
		if cur == nil || s.Timestamp.After(cur.Timestamp) || (s.Timestamp.Equal(cur.Timestamp) && s.ID > cur.ID) {
			cur = s
		}
	}
	if cur != nil {
		query.Closure(g, cur, reconciled)
	}
	return cur, reconciled
}

func item(g *store.Graph, ws *store.Workspace, p *dkf.Particular, kept []dkf.Assertion, o Options) Item {
	cur, reconciled := belief(g, kept)
	return Item{
		ACL:        grantEveryone(),
		Properties: properties(g, ws, p, kept, cur, reconciled, o),
		Content:    Content{Value: Brief(p, kept, cur, reconciled), Type: "text"},
	}
}

// Brief renders the item body: the belief, what it could not reconcile, and
// the supporting claims with their evidence. It is what Copilot summarises.
func Brief(p *dkf.Particular, kept []dkf.Assertion, cur *dkf.Synthesis, reconciled map[string]bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  (%s)\n", p.Label, p.URI)

	claims, unsynth := 0, 0
	for _, a := range kept {
		if a.ObjectType() == dkf.TypeClaim {
			claims++
		}
		if cur == nil || (a.ObjectID() != cur.ID && !reconciled[a.ObjectID()]) {
			unsynth++
		}
	}

	if cur != nil {
		fmt.Fprintf(&b, "\nCURRENT BELIEF (%s, %s%s)\n%s\n", cur.ID, dkf.FormatTime(cur.Timestamp), confSuffix(cur.Confidence), strings.TrimSpace(cur.Content))
		if u := strings.TrimSpace(cur.Unresolved); u != "" {
			fmt.Fprintf(&b, "\nNOT RECONCILED\n%s\n", u)
		}
	} else {
		fmt.Fprintf(&b, "\nNO SYNTHESIS YET — %s not yet reconciled\n", plural(unsynth, "assertion"))
	}

	var lines []string
	for _, a := range kept {
		if cur != nil && a.ObjectID() == cur.ID {
			continue
		}
		line := "- " + oneLine(a.GetContent()) + confSuffix(a.GetConfidence())
		// The register travels with the text: without a marker, a consumer
		// citing the brief cannot distinguish a fluent position from an
		// observed fact — which is what the evidential exists for downstream.
		if c, ok := a.(*dkf.Claim); ok {
			switch c.Evidential {
			case dkf.EvidentialHeld:
				line += " [position]"
			case "":
				line += " [undeclared]"
			}
		}
		if doc := strings.TrimSpace(a.GetSource().Document.Ref); doc != "" {
			line += " — evidence: " + doc
		}
		if cur != nil && !reconciled[a.ObjectID()] {
			line += "   [unsynthesised]"
		}
		lines = append(lines, line)
	}
	if len(lines) > 0 {
		b.WriteString("\nSUPPORTING\n")
		b.WriteString(strings.Join(lines, "\n"))
		b.WriteString("\n")
	}
	return b.String()
}

func confSuffix(c *float64) string {
	if c == nil {
		return ""
	}
	return ", confidence " + strconv.FormatFloat(*c, 'g', -1, 64)
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")), " ")
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func properties(g *store.Graph, ws *store.Workspace, p *dkf.Particular, kept []dkf.Assertion, cur *dkf.Synthesis, reconciled map[string]bool, o Options) Properties {
	var topics, authors []string
	seenTopic, seenAuthor := map[string]bool{}, map[string]bool{}
	var latest time.Time
	var newestClaim dkf.Assertion
	scope := dkf.ScopeOrganisation
	claims, open := 0, 0

	for _, a := range kept {
		ctx := a.GetContext()
		for _, t := range ctx.Topics {
			if t = strings.TrimSpace(t); t != "" && !seenTopic[t] {
				seenTopic[t] = true
				topics = append(topics, t)
			}
		}
		if au := strings.TrimSpace(a.GetSource().Author); au != "" && !seenAuthor[au] {
			seenAuthor[au] = true
			authors = append(authors, au)
		}
		if g.EffectiveScope(a.ObjectID()) == dkf.ScopePublic {
			scope = dkf.ScopePublic
		}
		if ts := a.GetTimestamp(); ts.After(latest) {
			latest = ts
		}
		if a.ObjectType() == dkf.TypeClaim {
			claims++
			if newestClaim == nil || a.ObjectID() > newestClaim.ObjectID() {
				newestClaim = a
			}
		}
		if cur == nil || (a.ObjectID() != cur.ID && !reconciled[a.ObjectID()]) {
			open++
		}
	}
	sort.Strings(topics)
	sort.Strings(authors)

	props := Properties{
		Title:         p.Label,
		ParticularURI: p.URI,
		Scope:         string(scope),
		Topics:        topics,
		Authors:       authors,
		ClaimCount:    claims,
		OpenQuestions: open,
	}
	if !latest.IsZero() {
		props.LastModified = dkf.FormatTime(latest)
	}
	if cur != nil {
		props.CurrentSynthesis = cur.ID
	}
	if o.SourceURL != "" {
		target := newestClaim
		if cur != nil {
			target = cur
		}
		if target != nil {
			if path, err := ws.Path(target.ObjectID()); err == nil {
				props.URL = joinURL(o.SourceURL, ws.Rel(path))
			}
		}
	}
	return props
}

func joinURL(base, rel string) string {
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + strings.TrimPrefix(rel, "/")
}
