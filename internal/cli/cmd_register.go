package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/query"
)

func (a *app) registerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "List the particulars and topics this workspace already uses",
		Long: `Reports what the workspace already has names for, so a new claim reuses one
rather than minting a near-duplicate: every particular with its aliases, claim
count and how many of those claims no belief has yet reconciled, then every
topic with how often it is carried and which are used only once.

Derived from the object files alone, so it cannot go stale. An MCP client
receives it with the server's instructions and does not need to run this.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			reg := query.BuildRegister(g)
			return a.emit(reg, func(w io.Writer) {
				fmt.Fprint(w, reg.Render(query.RegisterCap))
			})
		}),
	}
	return cmd
}
