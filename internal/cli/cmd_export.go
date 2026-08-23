package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/graph"
	"github.com/nodelogicau/particulars-cli/internal/viz"
)

// Export formats. FormatGraph feeds Microsoft 365 Copilot; the other two are
// drawings of the workspace for a human to read.
const (
	FormatGraph   = "graph"
	FormatDOT     = "dot"
	FormatMermaid = "mermaid"
)

func visualFormat(f string) bool { return f == FormatDOT || f == FormatMermaid }

func (a *app) exportCmd() *cobra.Command {
	var format, sourceURL, out, manifest, scope, connection, name, description, subject string
	var schema, includeRetracted bool
	var depth int
	cmd := &cobra.Command{
		Use:   "export --format graph|dot|mermaid [--subject <particular>] [--out <path>]",
		Short: "Emit the workspace for an external index, or as a diagram",
		Long: `Writes Microsoft Graph externalItem payloads as NDJSON — one item per
particular, carrying the current belief, what it could not reconcile, and the
claims that support it. Retracted objects are never exported, and personal
knowledge never leaves the workspace: only organisation and public scopes are
emitted.

This command emits; it never contacts Microsoft. Push the items with your own
job (see docs/graph.md). With --schema it emits the connection and schema
registration payloads instead.

--format dot and --format mermaid draw the workspace instead (see
docs/visualise.md). With --subject, the dialectic for that particular: claims
and syntheses as nodes, inputs as edges labelled with their role, the current
belief emphasised and unreconciled or stale assertions marked. Without one, a
map of every particular, weighted by what is known and joined by merge records.
Neither needs Graphviz installed — they emit text for you to render.

Unlike --format graph, the drawings include personal knowledge by default: a
local file is not a transfer to anyone, and a diagram missing half the graph
would misrepresent the reasoning. A diagram discloses whatever it contains, so
publishing one is the same judgement as publishing the objects.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			switch format {
			case FormatGraph, FormatDOT, FormatMermaid:
			default:
				return usageErr("--format is required and must be one of %q, %q, or %q", FormatGraph, FormatDOT, FormatMermaid)
			}
			if format == FormatGraph {
				// Reject the drawing flags rather than accepting them silently:
				// a flag that appears to work and does nothing is worse than one
				// that is refused.
				for _, f := range []struct {
					name string
					set  bool
				}{{"--subject", subject != ""}, {"--depth", depth != 0}, {"--include-retracted", includeRetracted}} {
					if f.set {
						return usageErr("%s applies to --format dot and mermaid, not %q", f.name, FormatGraph)
					}
				}
			}
			if visualFormat(format) {
				for _, f := range []struct {
					name string
					set  bool
				}{{"--schema", schema}, {"--manifest", manifest != ""}, {"--source-url", sourceURL != ""}} {
					if f.set {
						return usageErr("%s applies to --format %s, not %q", f.name, FormatGraph, format)
					}
				}
				return a.exportVisual(format, subject, scope, out, depth, includeRetracted)
			}
			if schema {
				if connection == "" {
					return usageErr("--schema requires --connection <id>")
				}
				reg := graph.NewRegistration(connection, name, description)
				return a.emit(reg, func(w io.Writer) {
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					_ = enc.Encode(reg)
				})
			}
			var only dkf.Scope
			if scope != "" {
				only = dkf.Scope(scope)
				switch {
				case only == dkf.ScopePersonal:
					return usageErr("personal knowledge is never exported; --scope accepts organisation or public")
				case !dkf.ValidScope(only):
					return usageErr("invalid --scope %q: must be organisation or public", scope)
				}
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			lines := graph.Build(g, ws, graph.Options{SourceURL: sourceURL, Scope: only})

			var buf strings.Builder
			ids := make([]string, 0, len(lines))
			for _, l := range lines {
				b, err := json.Marshal(l)
				if err != nil {
					return err
				}
				buf.Write(b)
				buf.WriteByte('\n')
				ids = append(ids, l.ID)
			}
			sort.Strings(ids)

			target := ""
			if out != "" {
				if abs, err := filepath.Abs(out); err == nil {
					target = abs
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(target, []byte(buf.String()), 0o644); err != nil {
					return err
				}
			}
			if manifest != "" {
				if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(manifest, []byte(strings.Join(ids, "\n")+"\n"), 0o644); err != nil {
					return err
				}
			}
			skipped := len(g.Particulars) - len(lines)
			if a.jsonOut || out != "" {
				res := map[string]any{"exported": len(lines), "skipped": skipped}
				if target != "" {
					res["path"] = target
				}
				if manifest != "" {
					res["manifest"] = manifest
				}
				return a.emit(res, func(w io.Writer) {
					fmt.Fprintf(w, "Exported %s (%s skipped: no exportable knowledge)\n", plural(len(lines), "item"), plural(skipped, "particular"))
					if target != "" {
						fmt.Fprintf(w, "  %s\n", target)
					}
				})
			}
			bw := bufio.NewWriter(a.stdout)
			defer func() { _ = bw.Flush() }()
			_, err = bw.WriteString(buf.String())
			return err
		}),
	}
	cmd.Flags().StringVar(&format, "format", "", "output format: "+FormatGraph+", "+FormatDOT+", or "+FormatMermaid)
	cmd.Flags().StringVar(&subject, "subject", "", "dot/mermaid: draw one particular's lineage (id, URI, label, or alias) instead of the workspace map")
	cmd.Flags().IntVar(&depth, "depth", 0, "dot/mermaid: limit the lineage to this many levels of inputs (0 = all)")
	cmd.Flags().BoolVar(&includeRetracted, "include-retracted", false, "dot/mermaid: draw retracted objects, marked as such")
	cmd.Flags().StringVar(&sourceURL, "source-url", "", "base URL of the workspace in source control, so citations link to the reviewed file")
	cmd.Flags().StringVar(&out, "out", "", "write NDJSON to this file instead of stdout")
	cmd.Flags().StringVar(&manifest, "manifest", "", "also write the exported item ids, one per line, for deletion diffing")
	cmd.Flags().StringVar(&scope, "scope", "", "export only this scope; graph accepts organisation or public, dot and mermaid also accept personal")
	cmd.Flags().BoolVar(&schema, "schema", false, "emit the connection and schema registration payloads")
	cmd.Flags().StringVar(&connection, "connection", "", "with --schema: the external connection id")
	cmd.Flags().StringVar(&name, "name", "", "with --schema: the connection display name")
	cmd.Flags().StringVar(&description, "description", "", "with --schema: the connection description")
	return cmd
}

// exportVisual renders the workspace as DOT or Mermaid.
func (a *app) exportVisual(format, subject, scope, out string, depth int, includeRetracted bool) error {
	if depth < 0 {
		return usageErr("--depth must not be negative")
	}
	opts := viz.Options{Depth: depth, IncludeRetracted: includeRetracted}
	if scope != "" {
		// personal is refused for --format graph, where it would send private
		// knowledge to a third-party index. A drawing goes to a local file.
		if !dkf.ValidScope(dkf.Scope(scope)) {
			return usageErr("invalid --scope %q: must be personal, organisation, or public", scope)
		}
		opts.Scope = dkf.Scope(scope)
	}
	ws, err := a.openWorkspace()
	if err != nil {
		return err
	}
	g, err := loadGraph(ws)
	if err != nil {
		return err
	}
	var p *dkf.Particular
	if subject != "" {
		if p, err = resolveSubject(g, subject); err != nil {
			return err
		}
	}
	m := viz.Build(g, p, opts)
	text := viz.DOT(m)
	if format == FormatMermaid {
		text = viz.Mermaid(m)
	}

	target := ""
	if out != "" {
		if target, err = filepath.Abs(out); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(text), 0o644); err != nil {
			return err
		}
	}
	if a.jsonOut || target != "" {
		res := map[string]any{"format": format, "view": string(m.View), "nodes": len(m.Nodes), "edges": len(m.Edges)}
		if m.Subject != "" {
			res["subject"] = m.Subject
		}
		if target != "" {
			res["path"] = target
		}
		return a.emit(res, func(w io.Writer) {
			fmt.Fprintf(w, "Wrote %s: %s of %s, %s\n", target, m.View, plural(len(m.Nodes), "node"), plural(len(m.Edges), "edge"))
		})
	}
	bw := bufio.NewWriter(a.stdout)
	defer func() { _ = bw.Flush() }()
	_, err = bw.WriteString(text)
	return err
}
