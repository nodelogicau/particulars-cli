package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (a *app) workspaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workspace",
		Short: "Show which workspace would be used and how it was found",
		Long: `Resolution order: --workspace, then $DKF_WORKSPACE, then the nearest ancestor
directory containing dkf.yaml or a .dkf pointer file. Exit 5 when none applies.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, res, err := store.DiscoverWith(a.workspace)
			if err != nil {
				return err
			}
			out := map[string]any{"root": res.Root, "via": res.Via, "id": ws.Config.Workspace.ID, "base_uri": ws.Config.Workspace.BaseURI}
			if res.Pointer != "" {
				out["pointer"] = res.Pointer
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "%s\n  via: %s", res.Root, res.Via)
				if res.Pointer != "" {
					fmt.Fprintf(w, " (%s)", res.Pointer)
				}
				fmt.Fprintf(w, "\n  id: %s\n", ws.Config.Workspace.ID)
				if ws.Config.Workspace.BaseURI != "" {
					fmt.Fprintf(w, "  base-uri: %s\n", ws.Config.Workspace.BaseURI)
				}
			})
		}),
	}
}
