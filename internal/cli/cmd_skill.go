package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	skill "github.com/nodelogicau/particulars-cli/skills/particulars"
)

type targetKind int

const (
	kindSkillFile targetKind = iota
	kindCursorRule
	kindAgentsSection
)

// preset describes where and in what shape a harness reads the skill.
type preset struct {
	name    string
	project string // relative to cwd
	user    string // relative to $HOME; "" = no user variant
	kind    targetKind
	note    string
}

var presets = []preset{
	{"claude", ".claude/skills/particulars/SKILL.md", ".claude/skills/particulars/SKILL.md", kindSkillFile, "Claude Code (also read by GitHub Copilot)"},
	{"copilot", ".github/skills/particulars/SKILL.md", ".copilot/skills/particulars/SKILL.md", kindSkillFile, "GitHub Copilot"},
	{"agents", ".agents/skills/particulars/SKILL.md", ".agents/skills/particulars/SKILL.md", kindSkillFile, "vendor-neutral Agent Skills location (read by Copilot)"},
	{"cursor", ".cursor/rules/particulars.mdc", "", kindCursorRule, "Cursor project rule"},
	{"agents-md", "AGENTS.md", "", kindAgentsSection, "section in AGENTS.md (Codex, Jules, Gemini CLI, Cursor, Copilot, …)"},
}

// copilotLocations are the project paths GitHub Copilot loads skills from;
// installing to more than one loads the skill twice.
var copilotLocations = []string{".claude/skills/particulars/SKILL.md", ".github/skills/particulars/SKILL.md", ".agents/skills/particulars/SKILL.md"}

func presetByName(name string) (preset, bool) {
	for _, p := range presets {
		if p.name == name {
			return p, true
		}
	}
	return preset{}, false
}

func presetNames() string {
	names := make([]string, len(presets))
	for i, p := range presets {
		names[i] = p.name
	}
	return strings.Join(names, "|")
}

func renderFor(kind targetKind) []byte {
	switch kind {
	case kindCursorRule:
		return skill.RenderCursorRule(version)
	case kindAgentsSection:
		return skill.RenderAgentsSection(version)
	}
	return skill.Render(version)
}

func (a *app) skillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Show or install the agent-facing skill embedded in this binary",
		Long: `The skill teaches an agent the verbs and the recall-before-assert discipline.
It ships inside the binary, stamped with this version, so skill and verbs
cannot drift. Neither subcommand needs a workspace.

Harness presets (--harness):
  claude     .claude/skills/particulars/SKILL.md   (default; Claude Code, also read by Copilot)
  copilot    .github/skills/particulars/SKILL.md   (GitHub Copilot)
  agents     .agents/skills/particulars/SKILL.md   (vendor-neutral; read by Copilot)
  cursor     .cursor/rules/particulars.mdc         (Cursor rule)
  agents-md  a bounded section in AGENTS.md        (Codex, Jules, Gemini CLI, Cursor, Copilot, …)

GitHub Copilot reads all three skills directories — install to exactly one.`,
	}
	cmd.AddCommand(a.skillShowCmd(), a.skillInstallCmd())
	return cmd
}

func (a *app) skillShowCmd() *cobra.Command {
	var harness string
	cmd := &cobra.Command{
		Use:   "show [--harness <preset>]",
		Short: "Print the skill to stdout (exactly what install would write)",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			p, ok := presetByName(harness)
			if !ok {
				return usageErr("unknown --harness %q; one of %s", harness, presetNames())
			}
			content := renderFor(p.kind)
			return a.emit(map[string]any{"version": skill.NormaliseVersion(version), "harness": p.name, "content": string(content)}, func(w io.Writer) {
				_, _ = w.Write(content)
			})
		}),
	}
	cmd.Flags().StringVar(&harness, "harness", "claude", "preset to render: "+presetNames())
	return cmd
}

type target struct {
	harness string
	path    string
	kind    targetKind
	user    bool
}

func (a *app) skillInstallCmd() *cobra.Command {
	var user, force, check bool
	var dir, file string
	var harnesses []string
	cmd := &cobra.Command{
		Use:   "install [--harness <preset>]... [--user | --dir <path>] [--file <path>] [--force] [--check]",
		Short: "Write the skill where a harness loads it (Claude Code project by default)",
		Long: `Writes the skill for each selected preset (see 'particulars skill --help').
A file this tool did not write (no marker line) is never overwritten without
--force. For agents-md only a marker-bounded section of the file is owned;
everything else in AGENTS.md is left byte-for-byte as it was.
--check compares without writing and exits 4 on drift, ignoring the stamped
version — use it in CI.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			targets, err := resolveTargets(harnesses, user, dir, file)
			if err != nil {
				return err
			}
			v := skill.NormaliseVersion(version)
			var results []map[string]any
			var warnings []string
			failed := false
			for _, t := range targets {
				res := map[string]any{"path": t.path, "harness": t.harness, "version": v}
				if check {
					status, err := checkTarget(t)
					if err != nil {
						return err
					}
					res["status"] = status
					if status != "ok" {
						failed = true
					}
				} else {
					outcome, err := installTarget(t, force)
					if err != nil {
						return err
					}
					res[outcome] = true
					warnings = append(warnings, duplicateWarnings(t)...)
				}
				results = append(results, res)
			}
			out := map[string]any{"targets": results, "version": v}
			if len(results) == 1 { // backward-compatible single-target shape
				for k, val := range results[0] {
					out[k] = val
				}
			}
			if warnings == nil {
				warnings = []string{}
			}
			if !check {
				out["warnings"] = warnings
			}
			if err := a.emit(out, func(w io.Writer) {
				for _, r := range results {
					switch {
					case check:
						fmt.Fprintf(w, "%s (%s): %s\n", r["path"], r["harness"], r["status"])
					case r["created"] == true:
						fmt.Fprintf(w, "Installed skill (particulars %s, %s) at %s\n", v, r["harness"], r["path"])
					case r["updated"] == true:
						fmt.Fprintf(w, "Updated skill (particulars %s, %s) at %s\n", v, r["harness"], r["path"])
					default:
						fmt.Fprintf(w, "Skill already up to date (%s) at %s\n", r["harness"], r["path"])
					}
				}
				for _, wmsg := range warnings {
					fmt.Fprintf(w, "warning: %s\n", wmsg)
				}
			}); err != nil {
				return err
			}
			if check && failed {
				return checkFailedErr("installed skill is out of date; run `particulars skill install`")
			}
			return nil
		}),
	}
	cmd.Flags().StringArrayVar(&harnesses, "harness", nil, "preset(s) to install: "+presetNames()+" (default claude; repeatable)")
	cmd.Flags().BoolVar(&user, "user", false, "install to the preset's personal location under $HOME")
	cmd.Flags().StringVar(&dir, "dir", "", "install SKILL.md into <dir> (cannot combine with --harness/--user)")
	cmd.Flags().StringVar(&file, "file", "", "with --harness agents-md: the file to hold the section (default AGENTS.md)")
	cmd.Flags().BoolVar(&force, "force", false, "replace a skill file not written by particulars")
	cmd.Flags().BoolVar(&check, "check", false, "verify the installed skill without writing; exit 4 on drift")
	return cmd
}

func resolveTargets(harnesses []string, user bool, dir, file string) ([]target, error) {
	if dir != "" {
		if len(harnesses) > 0 || user {
			return nil, usageErr("--dir cannot be combined with --harness or --user")
		}
		if file != "" {
			return nil, usageErr("--file is only valid with --harness agents-md")
		}
		p, err := filepath.Abs(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			return nil, err
		}
		return []target{{harness: "dir", path: p, kind: kindSkillFile}}, nil
	}
	if len(harnesses) == 0 {
		harnesses = []string{"claude"}
	}
	hasAgentsMD := false
	for _, h := range harnesses {
		if h == "agents-md" {
			hasAgentsMD = true
		}
	}
	if file != "" && !hasAgentsMD {
		return nil, usageErr("--file is only valid with --harness agents-md")
	}
	var out []target
	seen := map[string]bool{}
	for _, h := range harnesses {
		p, ok := presetByName(h)
		if !ok {
			return nil, usageErr("unknown --harness %q; one of %s", h, presetNames())
		}
		if seen[h] {
			continue
		}
		seen[h] = true
		var rel string
		switch {
		case user && p.user == "":
			return nil, usageErr("--harness %s has no personal (--user) location", p.name)
		case user:
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			rel = filepath.Join(home, filepath.FromSlash(p.user))
		case p.kind == kindAgentsSection && file != "":
			rel = file
		default:
			rel = filepath.FromSlash(p.project)
		}
		abs, err := filepath.Abs(rel)
		if err != nil {
			return nil, err
		}
		out = append(out, target{harness: p.name, path: abs, kind: p.kind, user: user})
	}
	return out, nil
}

// installTarget writes one target and reports created|updated|unchanged.
func installTarget(t target, force bool) (string, error) {
	rendered := renderFor(t.kind)
	existing, err := os.ReadFile(t.path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	var next []byte
	switch t.kind {
	case kindAgentsSection:
		if exists {
			start, end, ok, broken := skill.FindSection(existing)
			if broken {
				return "", &ExitError{Code: ExitRuntime, ErrCode: "conflict", Err: fmt.Errorf("%s: %v", t.path, skill.ErrBrokenSection)}
			}
			if ok && bytes.Equal(existing[start:end], rendered) {
				return "unchanged", nil
			}
		}
		spliced, err := skill.SpliceSection(existing, rendered)
		if err != nil {
			return "", err
		}
		next = spliced
	default:
		if exists && bytes.Equal(existing, rendered) {
			return "unchanged", nil
		}
		if exists && !skill.HasMarker(existing) && !force {
			return "", &ExitError{Code: ExitRuntime, ErrCode: "conflict", Err: fmt.Errorf("%s exists and was not written by particulars; pass --force to replace it", t.path)}
		}
		next = rendered
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return "", err
	}
	if err := writeAtomicFile(t.path, next); err != nil {
		return "", err
	}
	if exists {
		return "updated", nil
	}
	return "created", nil
}

// checkTarget reports ok|missing|differs|foreign without writing.
func checkTarget(t target) (string, error) {
	rendered := renderFor(t.kind)
	existing, err := os.ReadFile(t.path)
	if os.IsNotExist(err) {
		return "missing", nil
	}
	if err != nil {
		return "", err
	}
	if t.kind == kindAgentsSection {
		start, end, ok, broken := skill.FindSection(existing)
		switch {
		case broken:
			return "foreign", nil
		case !ok:
			return "missing", nil
		case !skill.BodyEqual(existing[start:end], rendered):
			return "differs", nil
		}
		return "ok", nil
	}
	switch {
	case !skill.HasMarker(existing):
		return "foreign", nil
	case !skill.BodyEqual(existing, rendered):
		return "differs", nil
	}
	return "ok", nil
}

// duplicateWarnings names other Copilot-readable project locations that
// already hold a particulars skill, so the user does not load it twice.
func duplicateWarnings(t target) []string {
	if t.user || t.kind != kindSkillFile || t.harness == "dir" {
		return nil
	}
	var others []string
	for _, loc := range copilotLocations {
		abs, err := filepath.Abs(filepath.FromSlash(loc))
		if err != nil || abs == t.path {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			others = append(others, loc)
		}
	}
	sort.Strings(others)
	if len(others) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("GitHub Copilot also reads %s; it will load the particulars skill more than once — keep one of %s", strings.Join(others, " and "), strings.Join(copilotLocations, ", "))}
}

func writeAtomicFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}
