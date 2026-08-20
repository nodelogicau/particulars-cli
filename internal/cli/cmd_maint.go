package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (a *app) indexCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "index [--check]",
		Short: "Rebuild index.yaml from the object files, or verify it with --check",
		Long: `index.yaml is derived: every mutating command keeps it current, and this
command regenerates it from scratch (resolving merge conflicts by rebuild).
--check compares the committed index with a rebuild and exits 4 on drift.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			if check {
				diff, err := ws.CheckIndex()
				if err != nil {
					return err
				}
				if err := a.emit(map[string]any{"clean": diff.Clean(), "missing": diff.Missing, "extra": diff.Extra, "changed": diff.Changed, "bytes_differ": diff.BytesDiffer}, func(w io.Writer) {
					if diff.Clean() {
						fmt.Fprintln(w, "index.yaml is up to date")
						return
					}
					fmt.Fprintln(w, "index.yaml is stale")
					for _, id := range diff.Missing {
						fmt.Fprintf(w, "  missing  %s\n", id)
					}
					for _, id := range diff.Extra {
						fmt.Fprintf(w, "  extra    %s\n", id)
					}
					for _, id := range diff.Changed {
						fmt.Fprintf(w, "  changed  %s\n", id)
					}
					if diff.BytesDiffer && len(diff.Missing)+len(diff.Extra)+len(diff.Changed) == 0 {
						fmt.Fprintln(w, "  (entries match but formatting differs)")
					}
				}); err != nil {
					return err
				}
				if !diff.Clean() {
					return checkFailedErr("index.yaml differs from a rebuild; run `particulars index`")
				}
				return nil
			}
			idx, err := ws.RebuildIndex()
			if err != nil {
				return err
			}
			return a.emit(map[string]any{"entries": len(idx.Entries), "path": store.IndexFile}, func(w io.Writer) {
				fmt.Fprintf(w, "Rebuilt %s with %s\n", store.IndexFile, plural(len(idx.Entries), "entry"))
			})
		}),
	}
	cmd.Flags().BoolVar(&check, "check", false, "verify without writing; exit 4 on drift")
	return cmd
}

func (a *app) validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check the workspace for structural and referential problems",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				// A present-but-malformed dkf.yaml is a validation failure, not a crash.
				if !errors.Is(err, store.ErrNoWorkspace) && strings.Contains(err.Error(), store.ConfigFile) {
					fs := query.Findings{{Severity: query.SeverityError, Path: store.ConfigFile, Code: query.CodeParseError, Message: err.Error()}}
					return a.emitFindings(fs)
				}
				return err
			}
			fs, err := query.Validate(ws)
			if err != nil {
				return err
			}
			return a.emitFindings(fs)
		}),
	}
	return cmd
}

func (a *app) emitFindings(fs query.Findings) error {
	if fs == nil {
		fs = query.Findings{}
	}
	errs, warns := 0, 0
	for _, f := range fs {
		if f.Severity == query.SeverityError {
			errs++
		} else {
			warns++
		}
	}
	if err := a.emit(map[string]any{"findings": fs, "errors": errs, "warnings": warns}, func(w io.Writer) {
		for _, f := range fs {
			fmt.Fprintf(w, "%-7s %s  %s: %s\n", f.Severity, f.Path, f.Code, f.Message)
		}
		fmt.Fprintf(w, "%s, %s\n", plural(errs, "error"), plural(warns, "warning"))
	}); err != nil {
		return err
	}
	if errs > 0 {
		return checkFailedErr("validation failed with %s", plural(errs, "error"))
	}
	return nil
}

func (a *app) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version and supported DKF format",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			return a.emit(map[string]any{"version": version, "format": dkf.FormatVersion}, func(w io.Writer) {
				fmt.Fprintf(w, "particulars %s (%s)\n", version, dkf.FormatVersion)
			})
		}),
	}
}
