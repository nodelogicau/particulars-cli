package viz

import (
	"fmt"
	"strings"
)

// DOT renders the model as a Graphviz digraph.
func DOT(m *Model) string {
	var b strings.Builder
	b.WriteString("digraph particulars {\n")
	b.WriteString("  rankdir=BT;\n")
	b.WriteString("  node [fontname=\"Helvetica\", fontsize=10, shape=box, style=rounded];\n")
	b.WriteString("  edge [fontname=\"Helvetica\", fontsize=9, color=\"#666666\"];\n")
	if m.View == ViewLineage && m.Subject != "" {
		fmt.Fprintf(&b, "  label=%s;\n  labelloc=t;\n", dotQuote(m.Subject))
	}
	for _, n := range m.Nodes {
		fmt.Fprintf(&b, "  %s [label=%s%s];\n", n.ID, dotQuote(dotLabel(n)), dotAttrs(n))
	}
	for _, e := range m.Edges {
		fmt.Fprintf(&b, "  %s -> %s%s;\n", e.From, e.To, dotEdgeAttrs(e))
	}
	b.WriteString("}\n")
	return b.String()
}

func dotLabel(n Node) string {
	if n.Kind == KindParticular {
		l := n.Label
		if n.Weight > 0 {
			l += "\\n" + plural(n.Weight, "claim")
		}
		if n.Priority > 0 {
			l += fmt.Sprintf(", priority %d", n.Priority)
		}
		return l
	}
	return Tag(n.ObjectID) + "\\n" + n.Label
}

// dotAttrs maps semantic state to Graphviz presentation. Every state is
// distinguishable without colour, so the diagram survives greyscale printing
// and colour-blind readers: shape, line style, and pen width carry it too.
func dotAttrs(n Node) string {
	var a []string
	switch n.Kind {
	case KindClaim:
		a = append(a, "shape=box")
	case KindSynthesis:
		a = append(a, "shape=box", "style=\"rounded,bold\"")
	case KindParticular:
		a = append(a, "shape=ellipse")
		if n.Weight > 0 {
			a = append(a, fmt.Sprintf("penwidth=%.1f", 1.0+float64(min(n.Weight, 10))/5.0))
		}
	}
	switch n.State {
	case StateCurrent:
		a = append(a, "color=\"#22aa77\"", "penwidth=3")
	case StateUnsynthesised:
		a = append(a, "color=\"#ee8800\"", "style=\"rounded,dashed\"")
	case StateStale:
		a = append(a, "color=\"#cc3311\"", "style=\"rounded,diagonals\"")
	case StateRetracted:
		a = append(a, "color=\"#999999\"", "fontcolor=\"#999999\"", "style=\"rounded,dotted\"")
	}
	if n.Foreign {
		a = append(a, "peripheries=2")
	}
	// The full text of a truncated label, shown on hover in SVG output.
	if n.Tooltip != "" {
		a = append(a, "tooltip="+dotQuote(n.Tooltip))
	}
	if len(a) == 0 {
		return ""
	}
	return ", " + strings.Join(a, ", ")
}

func dotEdgeAttrs(e Edge) string {
	var a []string
	if e.Role != "" {
		a = append(a, "label="+dotQuote(e.Role))
	}
	switch e.Kind {
	case EdgeSuperseded:
		a = append(a, "style=dashed", "arrowhead=empty", "label=\"superseded by\"")
	case EdgeMerge:
		a = append(a, "dir=none", "style=bold", "label=\"same as\"")
	}
	if len(a) == 0 {
		return ""
	}
	return " [" + strings.Join(a, ", ") + "]"
}

// dotQuote escapes for a Graphviz quoted string: backslashes and quotes only —
// "\n" sequences already in the label are intentional line breaks.
func dotQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			// Keep an intentional \n; escape any other backslash.
			if i+1 < len(s) && s[i+1] == 'n' {
				b.WriteString("\\n")
				i++
				continue
			}
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\n', '\r':
			b.WriteString("\\n")
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
