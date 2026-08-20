package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
)

func (a *app) topicsCmd() *cobra.Command {
	var includeRetracted bool
	var scope string
	cmd := &cobra.Command{
		Use:   "topics [<particular>] [--scope <s>] [--include-retracted]",
		Short: "List topics in use, with how many assertions and particulars carry each",
		Long: `Lists every topic tag across the workspace (or for one particular), so an
agent can reuse existing tags rather than inventing near-duplicates. Retracted
assertions are excluded unless --include-retracted is given.`,
		Args: cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			opts := query.RecallOptions{IncludeRetracted: includeRetracted}
			if len(args) == 1 {
				p, err := resolveSubject(g, args[0])
				if err != nil {
					return err
				}
				opts.Subject = p.ID
			}
			if scope != "" {
				if !dkf.ValidScope(dkf.Scope(scope)) {
					return usageErr("invalid --scope %q", scope)
				}
				opts.Scope = dkf.Scope(scope)
			}
			topics := query.Topics(g, opts)
			return a.emit(map[string]any{"topics": topics, "count": len(topics)}, func(w io.Writer) {
				if len(topics) == 0 {
					fmt.Fprintln(w, "(no topics)")
					return
				}
				width := 5
				for _, t := range topics {
					if len(t.Topic) > width {
						width = len(t.Topic)
					}
				}
				fmt.Fprintf(w, "%-*s  %10s  %11s\n", width, "topic", "assertions", "particulars")
				for _, t := range topics {
					fmt.Fprintf(w, "%-*s  %10d  %11d\n", width, t.Topic, t.Assertions, t.Particulars)
				}
			})
		}),
	}
	cmd.Flags().BoolVar(&includeRetracted, "include-retracted", false, "count retracted claims and syntheses too")
	cmd.Flags().StringVar(&scope, "scope", "", "only assertions with this scope")
	return cmd
}
