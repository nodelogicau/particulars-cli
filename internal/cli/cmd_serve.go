package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	pmcp "github.com/nodelogicau/particulars-cli/internal/mcp"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// EnvHTTPToken carries the bearer token for serve --http. It is never a flag
// value, which would show in process listings.
const EnvHTTPToken = "DKF_HTTP_TOKEN"

func (a *app) serveCmd() *cobra.Command {
	var useMCP bool
	var httpAddr, tokenFile string
	var prov provenanceFlags
	cmd := &cobra.Command{
		Use:   "serve --mcp [--workspace <dir>] [--author] [--harness] [--model] [--http <addr>] [--http-token-file <path>]",
		Short: "Serve this workspace to an MCP client over stdio or HTTP",
		Long: `Runs a Model Context Protocol server bound to one workspace (resolved as every
verb does: --workspace, $DKF_WORKSPACE, then dkf.yaml/.dkf discovery from the
working directory; an explicit directory may hold a .dkf pointer instead of
dkf.yaml). Tools follow the DKF specification's names; results equal
the CLI's --json output. Stdout carries only the protocol; diagnostics go to
stderr. Configure a second workspace as a second server entry in your client.

Without --http the server speaks stdio. With --http <host:port> it speaks MCP
Streamable HTTP at /mcp instead, and writes nothing to stdout; a bare :port
listens on every interface. Every session shares the one workspace and its
write lock.

HTTP requests to /mcp need "Authorization: Bearer <token>". The token is the
trimmed contents of --http-token-file, else $` + EnvHTTPToken + `; it is never a
flag value. With no token the server listens only on 127.0.0.1, ::1 or
localhost, and refuses any other address. TLS is left to a proxy or platform
in front of the server.

GET /healthz answers {"status":"ok","version":...} without the token, for
readiness probes; every other path is 404. A session whose client sends no
name gets source.harness "http". On SIGTERM or SIGINT the server stops
accepting connections, lets calls in flight finish for up to 30 s, and exits 0.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if !useMCP {
				return usageErr("serve requires --mcp")
			}
			useHTTP := cmd.Flags().Changed("http")
			if !useHTTP && cmd.Flags().Changed("http-token-file") {
				return usageErr("--http-token-file needs --http")
			}
			var token string
			if useHTTP {
				var err error
				if token, err = httpToken(tokenFile); err != nil {
					return err
				}
				loop, err := pmcp.IsLoopback(httpAddr)
				if err != nil {
					return usageErr("--http %q: want host:port (%v)", httpAddr, err)
				}
				if token == "" && !loop {
					return usageErr("--http %s: a token is required off loopback; set %s or pass --http-token-file, or listen on 127.0.0.1", httpAddr, EnvHTTPToken)
				}
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
			srv := pmcp.New(pmcp.Options{Workspace: ws, Version: version, Author: prov.author, Harness: prov.harness, Model: prov.model, Remote: useHTTP})
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			if !useHTTP {
				_, _ = os.Stderr.WriteString("particulars " + version + " serving " + res.Root + " over stdio (via " + res.Via + ")\n")
				return srv.Run(ctx, &sdk.StdioTransport{})
			}
			ln, err := net.Listen("tcp", httpAddr)
			if err != nil {
				return err
			}
			auth := "token required"
			if token == "" {
				auth = "no token (loopback only)"
			}
			fmt.Fprintf(os.Stderr, "particulars %s serving %s over http at %s/mcp, %s (via %s)\n", version, res.Root, ln.Addr(), auth, res.Via)
			return srv.RunHTTP(ctx, ln, token)
		}),
	}
	cmd.Flags().BoolVar(&useMCP, "mcp", false, "speak MCP (required)")
	cmd.Flags().StringVar(&httpAddr, "http", "", "speak Streamable HTTP at host:port instead of stdio")
	cmd.Flags().StringVar(&tokenFile, "http-token-file", "", "file holding the bearer token (else $"+EnvHTTPToken+")")
	cmd.Flags().StringVar(&prov.author, "author", "", "default source.author (else $DKF_AUTHOR, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.harness, "harness", "", "default source.harness (else $DKF_HARNESS, dkf.yaml, then the client's name)")
	cmd.Flags().StringVar(&prov.model, "model", "", "default source.model")
	return cmd
}

// httpToken is the trimmed contents of file when one is named, else
// $DKF_HTTP_TOKEN. An empty token counts as none.
func httpToken(file string) (string, error) {
	if file == "" {
		return strings.TrimSpace(os.Getenv(EnvHTTPToken)), nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", usageErr("--http-token-file: %v", err)
	}
	return strings.TrimSpace(string(b)), nil
}
