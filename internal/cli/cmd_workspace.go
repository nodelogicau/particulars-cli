package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/apperr"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (a *app) workspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
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
			convRel, _, convErr := ws.Conventions()
			if convRel != "" {
				out["conventions"] = convRel
				if convErr != nil {
					out["conventions_missing"] = true
				}
			}
			convWarn := ws.ConventionsWarning()
			if convWarn != "" {
				out["conventions_invalid"] = ws.Config.Workspace.Conventions
			}
			if convWarn != "" {
				a.warnings = append(a.warnings, convWarn)
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
				if convRel != "" {
					if convErr != nil {
						fmt.Fprintf(w, "  conventions: %s (missing)\n", convRel)
					} else {
						fmt.Fprintf(w, "  conventions: %s\n", convRel)
					}
				}
			})
		}),
	}
	cmd.AddCommand(a.workspacePointerCmd())
	return cmd
}

// workspacePointerCmd writes the .dkf pointer that `init --pointer` writes at
// creation time, for a workspace that already exists.
func (a *app) workspacePointerCmd() *cobra.Command {
	var at string
	var force bool
	cmd := &cobra.Command{
		Use:   "pointer [workspace-dir]",
		Short: "Write a .dkf pointer so this directory resolves to the workspace",
		Long: `Writes <dir>/.dkf naming the workspace, so ` + "`particulars`" + ` run anywhere at or
below <dir> finds it without --workspace or $DKF_WORKSPACE. This is what
` + "`init --pointer`" + ` writes when the workspace is created; use this verb when the
workspace already exists.

With no argument the pointer names the workspace that would be used now
(--workspace, then $DKF_WORKSPACE, then discovery). The path is written relative
when the workspace lies inside <dir> — so the file survives being cloned
elsewhere — and absolute when it does not, in which case it is machine-specific
and should not be committed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			dir := at
			if dir == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				dir = cwd
			}
			dir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
				return usageErr("--at %s is not a directory", dir)
			}
			var ws *store.Workspace
			if len(args) == 1 {
				ws, err = store.Open(args[0])
			} else {
				ws, err = a.openWorkspace()
			}
			if err != nil {
				return err
			}
			if ws.Root == dir {
				return usageErr("%s is the workspace itself; its %s is already found from here", dir, store.ConfigFile)
			}
			if _, err := os.Stat(filepath.Join(dir, store.ConfigFile)); err == nil {
				return usageErr("%s already holds a %s, which wins over a pointer at the same level", dir, store.ConfigFile)
			}
			target := ws.Root
			relative := false
			if rel, err := filepath.Rel(dir, ws.Root); err == nil && !strings.HasPrefix(rel, "..") {
				target, relative = filepath.ToSlash(rel), true
			}
			path := filepath.Join(dir, store.PointerFile)
			if force {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
			path, err = store.WritePointer(dir, target)
			if err != nil {
				if apperr.IsDomain(err) {
					return fmt.Errorf("%w; pass --force to replace it", err)
				}
				return err
			}
			out := map[string]any{"pointer": path, "root": ws.Root, "target": target, "relative": relative}
			return a.emit(out, func(w io.Writer) {
				fmt.Fprintf(w, "Wrote %s → %s\n", path, target)
				if !relative {
					fmt.Fprintf(w, "  absolute: the workspace is outside %s, so this pointer is machine-specific — do not commit it\n", dir)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&at, "at", "", "directory to write .dkf in (default: the current directory)")
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing pointer that names a different workspace")
	return cmd
}
