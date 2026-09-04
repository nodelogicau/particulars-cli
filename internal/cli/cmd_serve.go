package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	pmcp "github.com/nodelogicau/particulars-cli/internal/mcp"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (a *app) serveCmd() *cobra.Command {
	var useMCP bool
	var prov provenanceFlags
	cmd := &cobra.Command{
		Use:   "serve --mcp [--workspace <dir>] [--author] [--harness] [--model]",
		Short: "Serve this workspace to an MCP client over stdio",
		Long: `Runs a Model Context Protocol server bound to one workspace (resolved as every
verb does: --workspace, $DKF_WORKSPACE, then dkf.yaml/.dkf discovery from the
working directory; an explicit directory may hold a .dkf pointer instead of
dkf.yaml). Tools follow the DKF specification's names; results equal
the CLI's --json output. Stdout carries only the protocol; diagnostics go to
stderr. Configure a second workspace as a second server entry in your client.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if !useMCP {
				return usageErr("serve requires --mcp (the only transport in this version)")
			}
			ws, res, err := store.DiscoverWith(a.workspace)
			if err != nil {
				return err
			}
			if warn := ws.ConventionsWarning(); warn != "" {
				fmt.Fprintln(os.Stderr, "warning: "+warn)
			}
			if rel, _, cerr := ws.Conventions(); cerr != nil {
				fmt.Fprintf(os.Stderr, "warning: workspace.conventions %s: %v — omitted from instructions\n", rel, cerr)
			}
			srv := pmcp.New(pmcp.Options{Workspace: ws, Version: version, Author: prov.author, Harness: prov.harness, Model: prov.model})
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			_, _ = os.Stderr.WriteString("particulars " + version + " serving " + res.Root + " over stdio (via " + res.Via + ")\n")
			return srv.Run(ctx, &sdk.StdioTransport{})
		}),
	}
	cmd.Flags().BoolVar(&useMCP, "mcp", false, "speak MCP over stdio (required)")
	cmd.Flags().StringVar(&prov.author, "author", "", "default source.author (else $DKF_AUTHOR, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.harness, "harness", "", "default source.harness (else $DKF_HARNESS, dkf.yaml, then the client's name)")
	cmd.Flags().StringVar(&prov.model, "model", "", "default source.model")
	return cmd
}
