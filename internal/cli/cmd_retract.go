package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// retractCmd is the top-level verb; `claim retract` is built from the same
// function as a compatibility alias.
func (a *app) retractCmd() *cobra.Command { return a.buildRetractCmd(false) }

func (a *app) buildRetractCmd(alias bool) *cobra.Command {
	var reason, supersededBy string
	var prov provenanceFlags
	short := "Mark a claim, synthesis, or merge record retracted (append-only; never deletes)"
	if alias {
		short = "Alias of `particulars retract` (kept for compatibility)"
	}
	cmd := &cobra.Command{
		Use:   "retract <id> --reason <text> [--superseded-by <id>] [flags]",
		Short: short,
		Long: `Appends a retracted block to the object's own file; nothing is deleted or
rewritten. Claims, syntheses, and merge records can be retracted. For a
typo-grade correction, assert the corrected claim first and point at it with
--superseded-by; merges are undone, not superseded.`,
		Args: cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if strings.TrimSpace(reason) == "" {
				return usageErr("--reason is required")
			}
			if !dkf.IsRetractableID(id) {
				return usageErr("%q is not a claim, synthesis, or merge id", id)
			}
			t, _ := dkf.TypeOfID(id)
			if t == dkf.TypeMerge && supersededBy != "" {
				return usageErr("--superseded-by is not allowed for merge records; a merge is undone, not superseded")
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			if supersededBy != "" {
				if !dkf.IsAssertionID(supersededBy) {
					return usageErr("--superseded-by %q is not a claim or synthesis id", supersededBy)
				}
				if !ws.Exists(supersededBy) {
					return notFoundErr("--superseded-by %s does not exist", supersededBy)
				}
			}
			src := resolveSource(ws, prov)
			if err := requireProvenance(src, false); err != nil {
				return err
			}
			r := &dkf.Retracted{Timestamp: time.Now().UTC().Truncate(time.Second), Reason: reason, Source: src, SupersededBy: supersededBy}
			updated, err := ws.Retract(id, r)
			if err != nil {
				return err
			}
			if err := ws.UpsertIndex(updated); err != nil {
				return err
			}
			return a.emit(map[string]any{"id": id, "type": updated.ObjectType(), "retracted": r}, func(w io.Writer) {
				fmt.Fprintf(w, "Retracted %s: %s\n", id, reason)
				if supersededBy != "" {
					fmt.Fprintf(w, "  superseded by %s\n", supersededBy)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&reason, "reason", "", "why it is retracted (required)")
	cmd.Flags().StringVar(&supersededBy, "superseded-by", "", "id of the claim or synthesis that replaces it (not for merges)")
	cmd.Flags().StringVar(&prov.author, "author", "", "who is retracting (default: $DKF_AUTHOR, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.harness, "harness", "", "harness performing the retraction (default: $DKF_HARNESS, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.model, "model", "", "model performing the retraction")
	return cmd
}
