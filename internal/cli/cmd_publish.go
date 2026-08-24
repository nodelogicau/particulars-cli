package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
)

func (a *app) publishCmd() *cobra.Command {
	var scope, reason string
	var prov provenanceFlags
	cmd := &cobra.Command{
		Use:   "publish <id>... --scope <scope> [--reason <text>] [flags]",
		Short: "Share claims and syntheses more widely (writes a promotion record)",
		Long: `Writes publishes/pub_….yaml naming the objects whose scope is being widened.
Claims are immutable, so scope is never rewritten: an object's effective scope
is its asserted scope widened by the promotions covering it.

Promotion may only widen. A scope narrower than an object's asserted scope is
refused — reduce exposure by retracting the promotion (or the object), not by
promoting downwards. Promotion does not cascade: promoting a synthesis leaves
its inputs where they are, so a reader may receive a conclusion citing inputs
it cannot resolve. Promote the inputs too when the chain should be traversable.

Objects are named by id only. Promotion is the one operation whose mistakes
cannot be taken back — retracting the record stops future readers, but not the
ones who already fetched it — so this verb will not guess which claim you
meant from a label. Get ids from 'particulars recall <thing> --json'.`,
		Args: cobra.MinimumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if scope == "" {
				return usageErr("--scope is required: personal, organisation, or public")
			}
			if !dkf.ValidScope(dkf.Scope(scope)) {
				return usageErr("invalid --scope %q: must be personal, organisation, or public", scope)
			}
			for _, id := range args {
				if !dkf.IsValidID(id) {
					return usageErr("%q is not an id; publish names claims and syntheses by id, not by label (try `particulars recall %q --json`)", id, id)
				}
				if !dkf.IsAssertionID(id) {
					return usageErr("%s is not a claim or synthesis; only claims and syntheses carry a scope to promote", id)
				}
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			src := resolveSource(ws, prov)
			if err := requireProvenance(src, false); err != nil {
				return err
			}
			pr, err := ws.CreatePromotion(args, dkf.Scope(scope), reason, src, time.Now().UTC().Truncate(time.Second))
			if err != nil {
				return err
			}
			// Promoting can create the condition on the promoted synthesis, and
			// clear it on one nobody named — so report it over the whole set.
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			warnings := query.ScopeFindingsForPromotion(g, pr)
			path, _ := ws.Path(pr.ID)
			out := map[string]any{"promotion": pr, "path": ws.Rel(path)}
			if len(warnings) > 0 {
				out["warnings"] = warnings
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Promoted %s to %s\n", plural(len(pr.Claims), "object"), pr.Scope)
				for _, id := range pr.Claims {
					fmt.Fprintf(w, "  %s\n", id)
				}
				fmt.Fprintf(w, "  %s\n", ws.Rel(path))
				for _, msg := range warnings {
					fmt.Fprintf(w, "warning: %s\n", msg)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&scope, "scope", "", "the scope to widen to: personal, organisation, or public")
	cmd.Flags().StringVar(&reason, "reason", "", "why these objects may be shared more widely")
	cmd.Flags().StringVar(&prov.author, "author", "", "source.author (default: $DKF_AUTHOR, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.harness, "harness", "", "source.harness (default: $DKF_HARNESS, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.model, "model", "", "source.model (default: $DKF_MODEL, then dkf.yaml)")
	return cmd
}
