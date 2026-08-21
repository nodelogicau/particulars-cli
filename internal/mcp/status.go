package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/particulars-cli/internal/query"
)

type statusIn struct{}

func (s *Server) workspaceStatus(ctx context.Context, req *sdk.CallToolRequest, _ statusIn) (*sdk.CallToolResult, any, error) {
	g, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	findings, err := query.Validate(s.ws)
	if err != nil {
		return errResult(err), nil, nil
	}
	errs, warns := 0, 0
	for _, f := range findings {
		if f.Severity == query.SeverityError {
			errs++
		} else {
			warns++
		}
	}
	claims, syntheses := 0, 0
	for _, a := range g.Assertions {
		if a.ObjectType() == "synthesis" {
			syntheses++
		} else {
			claims++
		}
	}
	out := map[string]any{
		"root": s.ws.Root, "id": s.ws.Config.Workspace.ID, "base_uri": s.ws.Config.Workspace.BaseURI,
		"counts":    map[string]int{"particulars": len(g.Particulars), "claims": claims, "syntheses": syntheses, "merges": len(g.Merges)},
		"validate":  map[string]any{"errors": errs, "warnings": warns, "unreadable": len(g.Problems)},
		"conflicts": query.Conflicts(g, ""),
	}
	if gs := gitStatus(ctx, s.ws.Root); gs != nil {
		out["git"] = gs
	}
	summary := fmt.Sprintf("%s: %d particulars, %d claims, %d syntheses, %d merges; validate %d errors/%d warnings", s.ws.Root, len(g.Particulars), claims, syntheses, len(g.Merges), errs, warns)
	if gs, ok := out["git"].(map[string]any); ok {
		summary += fmt.Sprintf("; %d uncommitted file(s)", len(gs["uncommitted"].([]string)))
	}
	return okResult(summary), out, nil
}

// gitStatus returns {"checkout": <repo root>, "uncommitted": [paths]} when the
// workspace lies inside a git checkout and git is available; nil otherwise.
// It only ever runs read-only commands.
func gitStatus(ctx context.Context, root string) map[string]any {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	top, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil
	}
	out, err := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "--untracked-files=all", "--", ".").Output()
	if err != nil {
		return nil
	}
	files := []string{}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		p := strings.TrimSpace(line[3:])
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		files = append(files, filepath.ToSlash(p))
	}
	_ = os.Getenv // keep os imported for future env-based opt-out
	return map[string]any{"checkout": strings.TrimSpace(string(top)), "uncommitted": files}
}
