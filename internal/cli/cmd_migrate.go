package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (a *app) migrateCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "migrate [--dry-run]",
		Short: "Move a dkf/0.1 workspace to the dkf/0.2 layout, sharded by month",
		Long: `Rewrites format in dkf.yaml, renames every object file — byte-identical — to
<type-dir>/<YYYY-MM>/<id>.yaml (legacy/ for ids that are not UUIDv7), and
regenerates the index as a dkf/0.2 root and sealed segments. Nothing an object
asserts changes, and git sees renames.

It refuses, writing nothing, if a file fails to load, a type directory holds
other subdirectories with YAML in them, or a target exists with different
content. Run on a dkf/0.2 workspace it finishes an interrupted migration; on a
migrated one it reports so and writes nothing. Merge or close open branches
first: a branch opened before the migration adds flat files, and needs
` + "`particulars migrate`" + ` run on it after rebasing.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			plan, err := ws.PlanMigration()
			if err != nil {
				return err
			}
			if len(plan.Problems) > 0 {
				return ws.Migrate(plan) // refuses, naming every problem
			}
			if plan.Nothing() {
				if d, err := ws.CheckIndex(); err == nil && d.Clean() {
					return a.emit(map[string]any{"already_migrated": true, "format": dkf.Format02}, func(w io.Writer) {
						fmt.Fprintln(w, "Already dkf/0.2; nothing to do")
					})
				}
			}
			out := map[string]any{"dry_run": dryRun, "format": dkf.Format02, "moved": len(plan.Moves), "sealed": plan.Sealed, "legacy": plan.Legacy, "already_migrated": false}
			if dryRun {
				out["moves"] = plan.Moves
				return a.emit(out, func(w io.Writer) {
					fmt.Fprintf(w, "Would move %s to dkf/0.2\n", plural(len(plan.Moves), "file"))
					for _, m := range plan.Moves {
						fmt.Fprintf(w, "  %s -> %s\n", m.From, m.To)
					}
					report(w, plan)
				})
			}
			if err := ws.Migrate(plan); err != nil {
				return err
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Migrated to dkf/0.2: moved %s and rebuilt %s\n", plural(len(plan.Moves), "file"), store.IndexFile)
				report(w, plan)
				fmt.Fprintln(w, "Run `particulars validate`, then commit the renames, dkf.yaml, and the index together.")
				fmt.Fprintln(w, "Restart any running particulars MCP server (quit Claude Desktop fully) — it read dkf.yaml at startup and still sees the old layout.")
			})
		}),
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would move and seal without writing")
	return cmd
}

// report lists a migration's sealed months and legacy ids.
func report(w io.Writer, plan *store.MigrationPlan) {
	if len(plan.Sealed) > 0 {
		fmt.Fprintf(w, "  sealed: %s\n", strings.Join(plan.Sealed, ", "))
	}
	if len(plan.Legacy) > 0 {
		fmt.Fprintf(w, "  legacy ids (to legacy/): %s\n", strings.Join(plan.Legacy, ", "))
	}
}
