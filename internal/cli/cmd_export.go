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
)

// FormatGraph is the only export format today.
const FormatGraph = "graph"

func (a *app) exportCmd() *cobra.Command {
	var format, sourceURL, out, manifest, scope, connection, name, description string
	var schema bool
	cmd := &cobra.Command{
		Use:   "export --format graph [--source-url <base>] [--out <path>] [--manifest <path>]",
		Short: "Emit the workspace for an external index (Microsoft Graph connector)",
		Long: `Writes Microsoft Graph externalItem payloads as NDJSON — one item per
particular, carrying the current belief, what it could not reconcile, and the
claims that support it. Retracted objects are never exported, and personal
knowledge never leaves the workspace: only organisation and public scopes are
emitted.

This command emits; it never contacts Microsoft. Push the items with your own
job (see docs/graph.md). With --schema it emits the connection and schema
registration payloads instead.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if format != FormatGraph {
				return usageErr("--format is required and must be %q", FormatGraph)
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
	cmd.Flags().StringVar(&format, "format", "", "output format: "+FormatGraph)
	cmd.Flags().StringVar(&sourceURL, "source-url", "", "base URL of the workspace in source control, so citations link to the reviewed file")
	cmd.Flags().StringVar(&out, "out", "", "write NDJSON to this file instead of stdout")
	cmd.Flags().StringVar(&manifest, "manifest", "", "also write the exported item ids, one per line, for deletion diffing")
	cmd.Flags().StringVar(&scope, "scope", "", "export only this scope: organisation or public")
	cmd.Flags().BoolVar(&schema, "schema", false, "emit the connection and schema registration payloads")
	cmd.Flags().StringVar(&connection, "connection", "", "with --schema: the external connection id")
	cmd.Flags().StringVar(&name, "name", "", "with --schema: the connection display name")
	cmd.Flags().StringVar(&description, "description", "", "with --schema: the connection description")
	return cmd
}
