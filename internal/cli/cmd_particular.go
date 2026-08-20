package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
)

func (a *app) particularCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "particular",
		Short: "Define and resolve particulars (the things claims are about)",
	}
	cmd.AddCommand(a.particularDefineCmd(), a.particularResolveCmd())
	return cmd
}

func (a *app) particularDefineCmd() *cobra.Command {
	var label, uri string
	var aliases []string
	cmd := &cobra.Command{
		Use:   "define --label <label> [--uri <uri>] [--alias <alias>]...",
		Short: "Create a particular, or update it if one with the same URI exists (idempotent on URI)",
		Long: `Without --uri the URI is minted as <base-uri><slug> when dkf.yaml sets
workspace.base-uri, otherwise urn:dkf:<workspace-id>:<slug>, where slug is
derived from the label. Defining the same label twice therefore hits the same
particular. Prefer an existing global URI (Wikidata, ORCID, a GitHub URL) when
the thing has one.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			label = strings.TrimSpace(label)
			if label == "" {
				return usageErr("--label is required")
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			if uri == "" {
				slug := dkf.Slugify(label)
				if slug == "" {
					return usageErr("label %q yields an empty slug; pass --uri explicitly", label)
				}
				uri = dkf.MintURI(ws.Config.Workspace.BaseURI, ws.Config.Workspace.ID, slug)
			}
			p, created, err := ws.UpsertParticular(uri, label, aliases)
			if err != nil {
				return err
			}
			return a.emit(map[string]any{"particular": p, "created": created}, func(w io.Writer) {
				verb := "Updated"
				if created {
					verb = "Created"
				}
				fmt.Fprintf(w, "%s %s  %s\n  uri: %s\n", verb, p.ID, p.Label, p.URI)
				if len(p.Aliases) > 0 {
					fmt.Fprintf(w, "  aliases: %s\n", strings.Join(p.Aliases, ", "))
				}
			})
		}),
	}
	cmd.Flags().StringVar(&label, "label", "", "human-readable label (required)")
	cmd.Flags().StringVar(&uri, "uri", "", "canonical URI; minted from the label when omitted")
	cmd.Flags().StringArrayVar(&aliases, "alias", nil, "alternative name (repeatable)")
	return cmd
}

func (a *app) particularResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve <id|uri|label|alias>",
		Short: "Find particulars by id, URI, label, or alias (label/alias case-insensitive)",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			matches := query.Resolve(g, args[0])
			if len(matches) == 0 {
				return notFoundErr("no particular matches %q", args[0])
			}
			return a.emit(map[string]any{"matches": matches}, func(w io.Writer) {
				for _, p := range matches {
					fmt.Fprintf(w, "%s  %s\n  uri: %s\n", p.ID, p.Label, p.URI)
					if len(p.Aliases) > 0 {
						fmt.Fprintf(w, "  aliases: %s\n", strings.Join(p.Aliases, ", "))
					}
				}
			})
		}),
	}
	return cmd
}
