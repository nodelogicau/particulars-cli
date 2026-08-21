// Package cli wires the DKF core to a cobra command tree. It owns the
// agent-facing contract: no prompts, --json everywhere, documented exit codes.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// version is set at build time via -ldflags "-X .../internal/cli.version=...".
var version = "dev"

// Exit codes (see design D10).
const (
	ExitOK          = 0
	ExitRuntime     = 1
	ExitUsage       = 2
	ExitNotFound    = 3
	ExitCheckFailed = 4
	ExitNoWorkspace = 5
)

// ExitError carries an exit code and a machine-readable error code.
type ExitError struct {
	Code    int
	ErrCode string
	Err     error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

func usageErr(format string, args ...any) error {
	return &ExitError{Code: ExitUsage, ErrCode: "usage", Err: fmt.Errorf(format, args...)}
}

func notFoundErr(format string, args ...any) error {
	return &ExitError{Code: ExitNotFound, ErrCode: "not_found", Err: fmt.Errorf(format, args...)}
}

func checkFailedErr(format string, args ...any) error {
	return &ExitError{Code: ExitCheckFailed, ErrCode: "check_failed", Err: fmt.Errorf(format, args...)}
}

// classify maps core errors to exit codes.
func classify(err error) *ExitError {
	if err == nil {
		return nil
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee
	}
	var ps dkf.Problems
	var p *dkf.Problem
	switch {
	case errors.Is(err, store.ErrNoWorkspace):
		return &ExitError{Code: ExitNoWorkspace, ErrCode: "no_workspace", Err: err}
	case errors.Is(err, store.ErrNotFound):
		return &ExitError{Code: ExitNotFound, ErrCode: "not_found", Err: err}
	case errors.As(err, &ps), errors.As(err, &p):
		return &ExitError{Code: ExitUsage, ErrCode: "invalid", Err: err}
	case errors.Is(err, store.ErrAlreadyExists), errors.Is(err, store.ErrAlreadyRetracted):
		return &ExitError{Code: ExitRuntime, ErrCode: "conflict", Err: err}
	}
	return &ExitError{Code: ExitRuntime, ErrCode: "runtime", Err: err}
}

// app holds per-invocation state shared by all commands.
type app struct {
	jsonOut   bool
	workspace string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	// stdinIsTerminal is consulted before reading stdin; the CLI never reads a TTY.
	stdinIsTerminal func() bool
}

// Execute runs the CLI with the given arguments and streams, returning the
// process exit code. It never calls os.Exit, so tests can drive it in-process.
func Execute(args []string, stdin io.Reader, stdout, stderr io.Writer, stdinIsTerminal func() bool) int {
	a := &app{stdin: stdin, stdout: stdout, stderr: stderr, stdinIsTerminal: stdinIsTerminal}
	root := a.rootCmd()
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)

	err := root.Execute()
	if err == nil {
		return ExitOK
	}
	ee := classify(err)
	// Errors that cobra produced itself (unknown command, bad args) arrive
	// untyped; they are usage errors.
	if !isTyped(err) {
		ee = &ExitError{Code: ExitUsage, ErrCode: "usage", Err: err}
	}
	a.printError(ee)
	return ee.Code
}

func isTyped(err error) bool {
	var ee *ExitError
	if errors.As(err, &ee) {
		return true
	}
	var ps dkf.Problems
	var p *dkf.Problem
	return errors.Is(err, store.ErrNoWorkspace) || errors.Is(err, store.ErrNotFound) ||
		errors.Is(err, store.ErrAlreadyExists) || errors.Is(err, store.ErrAlreadyRetracted) ||
		errors.As(err, &ps) || errors.As(err, &p) || errors.Is(err, errRuntime)
}

// errRuntime wraps arbitrary core errors so Execute can tell them apart from
// cobra's own usage errors.
var errRuntime = errors.New("runtime")

// run wraps a command body so that every error leaving it is typed.
func (a *app) run(fn func(cmd *cobra.Command, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := fn(cmd, args)
		if err == nil || isTyped(err) {
			return err
		}
		return fmt.Errorf("%w: %v", errRuntime, err)
	}
}

func (a *app) printError(ee *ExitError) {
	msg := ee.Err.Error()
	msg = strings.TrimPrefix(msg, errRuntime.Error()+": ")
	if a.jsonOut {
		enc := json.NewEncoder(a.stderr)
		_ = enc.Encode(map[string]any{"error": map[string]any{"code": ee.ErrCode, "message": msg}})
		return
	}
	fmt.Fprintf(a.stderr, "error: %s\n", msg)
}

// emit writes the command result: JSON when --json, otherwise via text.
func (a *app) emit(v any, text func(w io.Writer)) error {
	if a.jsonOut {
		enc := json.NewEncoder(a.stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(v)
	}
	text(a.stdout)
	return nil
}

func (a *app) openWorkspace() (*store.Workspace, error) {
	return store.Discover(a.workspace)
}

func (a *app) rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "particulars",
		Short: "Capture and query dialectical knowledge (DKF) in a YAML workspace",
		Long: `particulars reads and writes Dialectical Knowledge Format (DKF) workspaces:
directories of YAML files holding particulars, claims, and syntheses.

It is built to be driven by an LLM agent and reviewed by humans through git:
every verb is non-interactive, supports --json, and uses these exit codes:

  0 success   1 runtime error   2 usage error
  3 not found 4 check failed    5 no workspace

Provenance defaults: --author/--harness/--model flags, then DKF_AUTHOR,
DKF_HARNESS, DKF_MODEL environment variables, then dkf.yaml defaults. Every
source needs at least one of author or harness; syntheses always need harness.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&a.jsonOut, "json", false, "emit a single JSON object on stdout (errors as JSON on stderr)")
	root.PersistentFlags().StringVar(&a.workspace, "workspace", "", "workspace directory (default: $DKF_WORKSPACE, else nearest ancestor with dkf.yaml or a .dkf pointer)")
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error { return usageErr("%v", err) })
	root.CompletionOptions.DisableDefaultCmd = true

	root.AddCommand(
		a.initCmd(),
		a.workspaceCmd(),
		a.particularCmd(),
		a.claimCmd(),
		a.synthesisCmd(),
		a.retractCmd(),
		a.recallCmd(),
		a.topicsCmd(),
		a.lineageCmd(),
		a.conflictsCmd(),
		a.indexCmd(),
		a.validateCmd(),
		a.skillCmd(),
		a.versionCmd(),
	)
	return root
}
