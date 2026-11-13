// Package apperr defines the error codes shared by every front-end (CLI exit
// codes, MCP tool error codes) and maps core errors onto them.
package apperr

import (
	"errors"
	"fmt"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Exit codes (design D10 of bootstrap-dkf-cli).
const (
	ExitOK          = 0
	ExitRuntime     = 1
	ExitUsage       = 2
	ExitNotFound    = 3
	ExitCheckFailed = 4
	ExitNoWorkspace = 5
)

// Error carries an exit code and a machine-readable error code.
type Error struct {
	Code    int
	ErrCode string
	Err     error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// Usage is a caller mistake: bad flags, invalid values, ambiguity.
func Usage(format string, args ...any) error {
	return &Error{Code: ExitUsage, ErrCode: "usage", Err: fmt.Errorf(format, args...)}
}

// NotFound is an id, subject, or query that resolves to nothing.
func NotFound(format string, args ...any) error {
	return &Error{Code: ExitNotFound, ErrCode: "not_found", Err: fmt.Errorf(format, args...)}
}

// CheckFailed is a validate/check verdict.
func CheckFailed(format string, args ...any) error {
	return &Error{Code: ExitCheckFailed, ErrCode: "check_failed", Err: fmt.Errorf(format, args...)}
}

// Conflict is an attempt to redo something already done (exists, retracted).
func Conflict(format string, args ...any) error {
	return &Error{Code: ExitRuntime, ErrCode: "conflict", Err: fmt.Errorf(format, args...)}
}

// Classify maps any error to an *Error.
func Classify(err error) *Error {
	if err == nil {
		return nil
	}
	var ee *Error
	if errors.As(err, &ee) {
		return ee
	}
	var ps dkf.Problems
	var p *dkf.Problem
	switch {
	case errors.Is(err, store.ErrNoWorkspace):
		return &Error{Code: ExitNoWorkspace, ErrCode: "no_workspace", Err: err}
	case errors.Is(err, store.ErrNotFound):
		return &Error{Code: ExitNotFound, ErrCode: "not_found", Err: err}
	case errors.As(err, &ps), errors.As(err, &p):
		return &Error{Code: ExitUsage, ErrCode: "invalid", Err: err}
	case errors.Is(err, store.ErrAlreadyExists), errors.Is(err, store.ErrAlreadyRetracted), errors.Is(err, store.ErrMintedCollision):
		return &Error{Code: ExitRuntime, ErrCode: "conflict", Err: err}
	case errors.Is(err, store.ErrUnsupportedFormat):
		return &Error{Code: ExitRuntime, ErrCode: "unsupported_format", Err: err}
	case errors.Is(err, store.ErrMigrationRefused):
		return &Error{Code: ExitRuntime, ErrCode: "migration_refused", Err: err}
	case errors.Is(err, store.ErrSealedMonth):
		return &Error{Code: ExitRuntime, ErrCode: "sealed_month", Err: err}
	}
	return &Error{Code: ExitRuntime, ErrCode: "runtime", Err: err}
}

// IsDomain reports whether err is one of the recognised core/domain errors
// (as opposed to an arbitrary runtime failure).
func IsDomain(err error) bool {
	var ee *Error
	var ps dkf.Problems
	var p *dkf.Problem
	return errors.As(err, &ee) || errors.Is(err, store.ErrNoWorkspace) || errors.Is(err, store.ErrNotFound) ||
		errors.Is(err, store.ErrAlreadyExists) || errors.Is(err, store.ErrAlreadyRetracted) || errors.Is(err, store.ErrMintedCollision) ||
		errors.Is(err, store.ErrUnsupportedFormat) || errors.Is(err, store.ErrSealedMonth) || errors.Is(err, store.ErrMigrationRefused) ||
		errors.As(err, &ps) || errors.As(err, &p)
}

// DefineRefusal rewrites a *store.MintedCollisionError as a conflict whose
// message names the existing particular and what to do, using flag ("--uri"
// for the CLI, "uri" for MCP) for the explicit-URI parameter. Other errors are
// returned unchanged.
func DefineRefusal(err error, flag string) error {
	var mc *store.MintedCollisionError
	if !errors.As(err, &mc) {
		return err
	}
	via := ""
	if mc.Existing.URI != mc.URI {
		via = ", through a merge with " + mc.Existing.URI
	}
	return Conflict("label %q mints %s, which already names %s (%q%s); for a different thing pass an explicit %s, or refer to the same thing by its uri or id",
		mc.Label, mc.URI, mc.Existing.ID, mc.Existing.Label, via, flag)
}
