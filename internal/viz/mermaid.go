package viz

import (
	"fmt"
	"strings"
)

// Mermaid renders the model as a Mermaid flowchart.
//
// The syntax used here is deliberately conservative: quoted labels, classDef,
// and the three arrow forms that have been stable across Mermaid versions.
// Mermaid's parser is stricter than Graphviz's and its accepted grammar has
// changed between releases, so nothing newer is used for effect.
func Mermaid(m *Model) string {
	var b strings.Builder
	b.WriteString("flowchart BT\n")
	for _, n := range m.Nodes {
		lhs, rhs := "[", "]"
		switch n.Kind {
		case KindSynthesis:
			lhs, rhs = "(", ")"
		case KindParticular:
			lhs, rhs = "([", "])"
		}
		fmt.Fprintf(&b, "  %s%s%s%s\n", n.ID, lhs, mermaidQuote(mermaidLabel(n)), rhs)
	}
	for _, e := range m.Edges {
		arrow, label := "-->", e.Role
		switch e.Kind {
		case EdgeSuperseded:
			arrow, label = "-.->", "superseded by"
		case EdgeMerge:
			arrow, label = "---", "same as"
		}
		if label != "" {
			fmt.Fprintf(&b, "  %s %s|%s| %s\n", e.From, arrow, mermaidEdgeLabel(label), e.To)
		} else {
			fmt.Fprintf(&b, "  %s %s %s\n", e.From, arrow, e.To)
		}
	}
	// Truncated labels carry their full text as a hover tooltip. The callback
	// form of the click statement is the one tooltip syntax Mermaid has ever
	// had; with no such function defined a click does nothing, and renderers
	// that strip interactivity (GitHub) parse and ignore the line.
	for _, n := range m.Nodes {
		if n.Tooltip == "" {
			continue
		}
		fmt.Fprintf(&b, "  click %s callback \"%s\"\n", n.ID, mermaidTooltip(n.Tooltip))
	}
	// One class per state, applied by listing members — assigning a class per
	// node line would repeat the definition and bloat the diff.
	for _, st := range []State{StateCurrent, StateUnsynthesised, StateStale, StateRetracted} {
		var ids []string
		for _, n := range m.Nodes {
			if n.State == st {
				ids = append(ids, n.ID)
			}
		}
		if len(ids) == 0 {
			continue
		}
		fmt.Fprintf(&b, "  class %s %s\n", strings.Join(ids, ","), st)
	}
	var foreign []string
	for _, n := range m.Nodes {
		if n.Foreign {
			foreign = append(foreign, n.ID)
		}
	}
	if len(foreign) > 0 {
		fmt.Fprintf(&b, "  class %s foreign\n", strings.Join(foreign, ","))
	}
	b.WriteString("  classDef current stroke:#22aa77,stroke-width:3px\n")
	b.WriteString("  classDef unsynthesised stroke:#ee8800,stroke-dasharray:4 3\n")
	b.WriteString("  classDef stale stroke:#cc3311,stroke-dasharray:2 2\n")
	b.WriteString("  classDef retracted stroke:#999999,color:#999999,stroke-dasharray:1 3\n")
	b.WriteString("  classDef foreign stroke-width:1px,stroke-dasharray:6 2\n")
	return b.String()
}

func mermaidLabel(n Node) string {
	if n.Kind == KindParticular {
		l := n.Label
		if n.Weight > 0 {
			l += "<br>" + plural(n.Weight, "claim")
		}
		if n.Priority > 0 {
			l += fmt.Sprintf(", priority %d", n.Priority)
		}
		return l
	}
	return Tag(n.ObjectID) + "<br>" + n.Label
}

// mermaidQuote wraps a label in quotes and removes what Mermaid cannot carry
// inside one. A quote character terminates the label whatever precedes it —
// Mermaid has no escape for it — so quotes become typographic quotes rather
// than being dropped, keeping the text readable.
func mermaidQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteRune('”')
		case '\n', '\r':
			b.WriteString("<br>")
		case '<':
			// Keep the <br> we inserted deliberately; escape anything else.
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '|':
			// Legal inside a quoted label, but the pipe is the edge-label
			// delimiter and Mermaid's grammar has shifted between releases;
			// an entity is safe in every version.
			b.WriteString("&#124;")
		default:
			b.WriteRune(r)
		}
	}
	out := b.String() + "\""
	// Restore the intentional line breaks that the escaping above encoded.
	return strings.ReplaceAll(out, "&lt;br&gt;", "<br>")
}

// mermaidEdgeLabel sanitises an edge label, where the pipe delimiter and the
// quote both terminate the label.
func mermaidEdgeLabel(s string) string {
	r := strings.NewReplacer("|", "/", "\"", "”", "\n", " ", "\r", " ")
	return r.Replace(s)
}

// mermaidTooltip sanitises tooltip text, where a quote terminates the string
// with no escape. Tooltips render as plain text, so no entities here.
func mermaidTooltip(s string) string {
	r := strings.NewReplacer("\"", "”", "\n", " ", "\r", " ")
	return r.Replace(s)
}
