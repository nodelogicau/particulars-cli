package cli

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (a *app) particularCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "particular",
		Short: "Define and resolve particulars (the things claims are about)",
	}
	cmd.AddCommand(a.particularDefineCmd(), a.particularResolveCmd(), a.particularMergeCmd())
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

// resolveMergeSide turns a merge argument into a URI: a local particular's
// id/uri/label/alias, or, when nothing local matches, a bare URI.
func resolveMergeSide(g *store.Graph, arg string) (uri, localID string, err error) {
	matches := query.Resolve(g, arg)
	switch len(matches) {
	case 1:
		return matches[0].URI, matches[0].ID, nil
	case 0:
		if u, perr := url.Parse(strings.TrimSpace(arg)); perr == nil && u.Scheme != "" && !strings.ContainsAny(arg, " \t") {
			return strings.TrimSpace(arg), "", nil
		}
		return "", "", notFoundErr("%q matches no particular and is not a URI", arg)
	}
	ids := make([]string, len(matches))
	for i, m := range matches {
		ids[i] = m.ID
	}
	return "", "", usageErr("%q is ambiguous; it matches %s — use an id or uri", arg, strings.Join(ids, ", "))
}

func (a *app) particularMergeCmd() *cobra.Command {
	var reason string
	var prov provenanceFlags
	cmd := &cobra.Command{
		Use:   "merge <a> <b> [--reason <text>] [flags]",
		Short: "Declare that two URIs denote the same particular (writes a merge record)",
		Long: `Writes merges/mrg_….yaml joining two URIs; nothing else is rewritten. Each
side may be a local particular (id, uri, label, or alias) or a bare URI with
no local particular — merges routinely bridge to another source. Merged
particulars form an equivalence class that recall, conflicts, and lineage
operate over. Undo a merge with 'particulars retract <mrg_id>'.`,
		Args: cobra.ExactArgs(2),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			uriA, idA, err := resolveMergeSide(g, args[0])
			if err != nil {
				return err
			}
			uriB, idB, err := resolveMergeSide(g, args[1])
			if err != nil {
				return err
			}
			if uriA == uriB {
				return usageErr("both arguments resolve to %s; nothing to merge", uriA)
			}
			if g.MergeBetween(uriA, uriB) != nil {
				return usageErr("%s and %s are already joined by %s", uriA, uriB, g.MergeBetween(uriA, uriB).ID)
			}
			src := resolveSource(ws, prov)
			if err := requireProvenance(src, false); err != nil {
				return err
			}
			m, err := ws.CreateMerge(uriA, uriB, reason, src, time.Now().UTC().Truncate(time.Second))
			if err != nil {
				return err
			}
			sides := []map[string]any{{"uri": uriA, "particular": idA}, {"uri": uriB, "particular": idB}}
			path, _ := ws.Path(m.ID)
			return a.emit(map[string]any{"merge": m, "sides": sides, "path": ws.Rel(path)}, func(w io.Writer) {
				fmt.Fprintf(w, "Merged %s\n  %s", m.ID, uriA)
				if idA != "" {
					fmt.Fprintf(w, " (%s)", idA)
				}
				fmt.Fprintf(w, "\n  %s", uriB)
				if idB != "" {
					fmt.Fprintf(w, " (%s)", idB)
				} else {
					fmt.Fprintf(w, " (no local particular)")
				}
				fmt.Fprintf(w, "\n  %s\n", ws.Rel(path))
			})
		}),
	}
	cmd.Flags().StringVar(&reason, "reason", "", "why the two URIs are the same thing")
	cmd.Flags().StringVar(&prov.author, "author", "", "source.author (default: $DKF_AUTHOR, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.harness, "harness", "", "source.harness (default: $DKF_HARNESS, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.model, "model", "", "source.model (default: $DKF_MODEL, then dkf.yaml)")
	return cmd
}
