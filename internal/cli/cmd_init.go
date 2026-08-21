package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (a *app) initCmd() *cobra.Command {
	var baseURI, author, harness, scope string
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a new DKF workspace (dkf.yaml, particulars/, claims/, syntheses/, index.yaml)",
		Args:  cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			cfg := store.NewConfig()
			normalised := dkf.NormaliseBaseURI(baseURI)
			wasNormalised := normalised != strings.TrimSpace(baseURI)
			cfg.Workspace.BaseURI = normalised
			cfg.Defaults.Source.Author = author
			cfg.Defaults.Source.Harness = harness
			if scope != "" {
				if !dkf.ValidScope(dkf.Scope(scope)) {
					return usageErr("invalid --scope %q: must be personal, organisation, or public", scope)
				}
				cfg.Defaults.Scope = dkf.Scope(scope)
			}
			ws, err := store.Init(dir, cfg)
			if err != nil {
				return err
			}
			created := []string{store.ConfigFile, "particulars/", "claims/", "syntheses/", "merges/", store.IndexFile}
			out := map[string]any{
				"workspace": map[string]any{"root": ws.Root, "id": ws.Config.Workspace.ID, "base_uri": ws.Config.Workspace.BaseURI, "format": ws.Config.Format},
				"created":   created,
			}
			if wasNormalised {
				out["normalised"] = true
			}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Initialised DKF workspace at %s (id %s)\n", ws.Root, ws.Config.Workspace.ID)
				for _, c := range created {
					fmt.Fprintf(w, "  %s\n", filepath.ToSlash(c))
				}
				if wasNormalised {
					fmt.Fprintf(w, "  base-uri normalised to %s (must end in /)\n", ws.Config.Workspace.BaseURI)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&baseURI, "base-uri", "", "base URI under which particular URIs are minted (e.g. https://example.com/particulars/)")
	cmd.Flags().StringVar(&author, "author", "", "default source.author for claims")
	cmd.Flags().StringVar(&harness, "harness", "", "default harness for claims and syntheses")
	cmd.Flags().StringVar(&scope, "scope", "", "default scope: personal|organisation|public (default personal)")
	return cmd
}
