package cli

import (
	"errors"
	"fmt"
	"io"
	"sort"
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
		Use:   "validate [--notes]",
		Short: "Check the workspace for structural and referential problems",
		Long: `Reports errors (exit 4), warnings, and notes. Notes record something a
reader may want to know that is not a defect — provenance that could not be
machine-checked, for instance — and are summarised rather than listed unless
--notes is given. They are always present in --json.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				// A present-but-malformed dkf.yaml is a validation failure, not a crash.
				if !errors.Is(err, store.ErrNoWorkspace) && strings.Contains(err.Error(), store.ConfigFile) {
					code := query.CodeParseError
					if errors.Is(err, store.ErrInvalidBaseURI) {
						code = query.CodeInvalidBaseURI
					}
					fs := query.Findings{{Severity: query.SeverityError, Path: store.ConfigFile, Code: code, Message: err.Error()}}
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
	cmd.Flags().BoolVar(&a.showNotes, "notes", false, "list aggregated findings individually: notes, and warnings that record corpus facts (the legacy_* markers)")
	return cmd
}

func (a *app) emitFindings(fs query.Findings) error {
	if fs == nil {
		fs = query.Findings{}
	}
	errs, warns, notes := 0, 0, 0
	for _, f := range fs {
		switch f.Severity {
		case query.SeverityError:
			errs++
		case query.SeverityInfo:
			notes++
		default:
			warns++
		}
	}
	out := map[string]any{"findings": fs, "errors": errs, "warnings": warns}
	if notes > 0 {
		out["notes"] = notes
	}
	if err := a.emit(out, func(w io.Writer) {
		// Findings about an object list per object, because the object is the
		// unit of action — whatever their severity: defect_unverifiable is a
		// note, and it still names one retraction worth opening. Facts about
		// the corpus aggregate into one line per condition: permanent,
		// unactionable per object, and a per-object listing would recur on
		// every run forever, burying the findings that need acting on.
		// --json always carries every one.
		aggregated := map[string][]query.Finding{}
		var aggOrder []string
		for _, f := range fs {
			if !a.showNotes && query.IsCorpusFact(f.Code) {
				k := f.Severity + "\x00" + f.Code
				if f.Code == query.CodeAuthorUnresolved || f.Code == query.CodeAuthorAmbiguous {
					// One aggregate line per author value: these messages are
					// constructed uniform per value, so the message keys the group.
					k += "\x00" + f.Message
				}
				if _, seen := aggregated[k]; !seen {
					aggOrder = append(aggOrder, k)
				}
				aggregated[k] = append(aggregated[k], f)
				continue
			}
			fmt.Fprintf(w, "%-7s %s  %s: %s\n", f.Severity, f.Path, f.Code, f.Message)
		}
		// Warnings before observations, then by code, so the line a reader
		// should weigh first comes first.
		sort.Slice(aggOrder, func(i, j int) bool {
			si, sj := aggregated[aggOrder[i]][0], aggregated[aggOrder[j]][0]
			if si.Severity != sj.Severity {
				return si.Severity == query.SeverityWarning
			}
			return si.Code < sj.Code
		})
		for _, k := range aggOrder {
			group := aggregated[k]
			f := group[0]
			fmt.Fprintf(w, "%-7s %s  %s", f.Severity, plural(len(group), "object"), f.Code)
			// One fact, however many objects carry it: show the message when
			// every object says the same thing, and just the count when not.
			uniform := f.Message
			for _, g := range group[1:] {
				if g.Message != uniform {
					uniform = ""
					break
				}
			}
			if uniform != "" {
				fmt.Fprintf(w, ": %s", uniform)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s, %s", plural(errs, "error"), plural(warns, "warning"))
		if notes > 0 {
			fmt.Fprintf(w, ", %s", plural(notes, "note"))
		}
		if len(aggOrder) > 0 {
			fmt.Fprint(w, " (--notes to list)")
		}
		fmt.Fprintln(w)
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
