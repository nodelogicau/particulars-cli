package query

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// RegisterCap bounds the rendered register. The conventions document is
// allowed 16 KiB; the register is smaller because it is a list rather than
// prose, and because what matters is that the head survives. Both dogfood
// workspaces render well under this.
const RegisterCap = 12 * 1024

// RegisterParticular is one row of the register's first half: a thing the
// workspace already has a name for.
type RegisterParticular struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Aliases []string `json:"aliases,omitempty"`
	URI     string   `json:"uri,omitempty"`
	// Global is true when the URI was not minted from the label by this
	// workspace — a Wikidata entity, a repository, a home page. It is the
	// difference between a name only this workspace knows and one another
	// workspace could reuse.
	Global bool `json:"global"`
	Claims int  `json:"claims"`
	// Syntheses and Unsynthesised are class-wide where the particular has
	// been merged, because a merge says the members are one thing.
	Syntheses     int  `json:"syntheses"`
	Unsynthesised int  `json:"unsynthesised"`
	HasCurrent    bool `json:"has_current"`
}

// RegisterTopic is one row of the register's second half.
type RegisterTopic struct {
	Topic      string `json:"topic"`
	Assertions int    `json:"assertions"`
	// Singleton records that exactly one assertion carries the topic. It is
	// a fact, not a fault: a tag just coined is indistinguishable from one
	// that failed to catch on, and only the writer knows which this is.
	Singleton bool `json:"singleton"`
}

// Register is what the workspace already uses: the names available to reuse,
// and how heavily each is used. It is derived from the object files alone —
// it never reads the conventions document, which is the human's commentary on
// these facts rather than a source of them.
type Register struct {
	Particulars        []RegisterParticular `json:"particulars"`
	Topics             []RegisterTopic      `json:"topics"`
	OmittedParticulars int                  `json:"omitted_particulars,omitempty"`
	OmittedTopics      int                  `json:"omitted_topics,omitempty"`
}

// BuildRegister computes the register from a loaded graph. Counts cover live
// objects only: a retracted claim is not a name in use.
func BuildRegister(g *store.Graph) *Register {
	reg := &Register{Particulars: []RegisterParticular{}, Topics: []RegisterTopic{}}
	for _, p := range g.Particulars {
		row := RegisterParticular{
			ID: p.ID, Label: p.Label, Aliases: p.Aliases, URI: p.URI,
			Global: p.URI != "" && !strings.HasPrefix(p.URI, "urn:dkf:"),
		}
		for _, a := range g.BySubject[p.ID] {
			if a.GetRetracted() != nil {
				continue
			}
			if _, ok := a.(*dkf.Synthesis); ok {
				row.Syntheses++
			} else {
				row.Claims++
			}
		}
		// Analyse already computes what conflict_detect ranks by; the
		// register borrows the number rather than recomputing the closure.
		r := Analyse(g, p)
		row.Unsynthesised, row.HasCurrent = len(r.Unsynthesised), r.Current != ""
		reg.Particulars = append(reg.Particulars, row)
	}
	sort.Slice(reg.Particulars, func(i, j int) bool {
		a, b := reg.Particulars[i], reg.Particulars[j]
		if a.Claims != b.Claims {
			return a.Claims > b.Claims
		}
		return a.Label < b.Label
	})
	for _, t := range Topics(g, RecallOptions{}) {
		reg.Topics = append(reg.Topics, RegisterTopic{Topic: t.Topic, Assertions: t.Assertions, Singleton: t.Assertions == 1})
	}
	sort.Slice(reg.Topics, func(i, j int) bool {
		a, b := reg.Topics[i], reg.Topics[j]
		if a.Assertions != b.Assertions {
			return a.Assertions > b.Assertions
		}
		return a.Topic < b.Topic
	})
	return reg
}

// Render writes the register as text, bounded by cap. Where it does not fit,
// entries are dropped from the tail of each ordering — the least-used first,
// because an entry used once is the least likely to be the one worth reusing
// and the most likely to be the mistake this register exists to prevent.
func (reg *Register) Render(cap int) string {
	view := *reg
	for {
		out := view.render()
		if cap <= 0 || len(out) <= cap || (len(view.Particulars) == 0 && len(view.Topics) == 0) {
			return out
		}
		// Drop from the longer list, topics first on a tie: the topic tail is
		// singletons, which carry the least and cost a line each.
		if len(view.Topics) >= len(view.Particulars) && len(view.Topics) > 0 {
			view.Topics = view.Topics[:len(view.Topics)-1]
			view.OmittedTopics++
		} else if len(view.Particulars) > 0 {
			view.Particulars = view.Particulars[:len(view.Particulars)-1]
			view.OmittedParticulars++
		}
	}
}

func (reg *Register) render() string {
	var b strings.Builder
	b.WriteString("These are the particulars and topics this workspace already uses. Reuse a name before minting one: a near-miss particular is a duplicate, and a topic can never be renamed.\n\n")
	b.WriteString("## Particulars\n\n")
	if len(reg.Particulars) == 0 {
		b.WriteString("None yet.\n")
	}
	for _, p := range reg.Particulars {
		fmt.Fprintf(&b, "- **%s**", p.Label)
		if len(p.Aliases) > 0 {
			fmt.Fprintf(&b, " (aliases: %s)", strings.Join(p.Aliases, ", "))
		}
		fmt.Fprintf(&b, " — %s", plural(p.Claims, "claim"))
		if p.Syntheses > 0 {
			fmt.Fprintf(&b, ", %s", plural(p.Syntheses, "synthesis"))
		}
		// Unsynthesised counts assertions, so it can exceed the claim count;
		// naming the syntheses above keeps the arithmetic legible, and the
		// phrasing says what the number means rather than what it counts.
		switch {
		case !p.HasCurrent && p.Claims+p.Syntheses > 0:
			b.WriteString(", no belief yet")
		case p.Unsynthesised > 0:
			fmt.Fprintf(&b, ", %d outside the current belief", p.Unsynthesised)
		}
		if !p.Global && p.URI != "" {
			b.WriteString(", workspace-minted URI")
		}
		b.WriteString("\n")
	}
	if reg.OmittedParticulars > 0 {
		fmt.Fprintf(&b, "- … and %d more, least used omitted\n", reg.OmittedParticulars)
	}
	b.WriteString("\n## Topics in use\n\n")
	if len(reg.Topics) == 0 {
		b.WriteString("None yet.\n")
	} else {
		parts := make([]string, 0, len(reg.Topics))
		single := 0
		for _, t := range reg.Topics {
			parts = append(parts, fmt.Sprintf("%s (%d)", t.Topic, t.Assertions))
			if t.Singleton {
				single++
			}
		}
		b.WriteString(strings.Join(parts, ", ") + "\n")
		if single > 0 {
			fmt.Fprintf(&b, "\n%d of %d used once. A tag earns its place by being reused; prefer one of the above to a new one.\n", single, len(reg.Topics))
		}
	}
	if reg.OmittedTopics > 0 {
		fmt.Fprintf(&b, "\n%d further topics omitted, least used first.\n", reg.OmittedTopics)
	}
	return b.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	if word == "synthesis" {
		return fmt.Sprintf("%d syntheses", n)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
