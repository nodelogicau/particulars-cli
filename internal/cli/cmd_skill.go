package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	skill "github.com/nodelogicau/particulars-cli/skills/particulars"
)

// skillRelPath is where Claude Code looks for a project or user skill.
const skillRelPath = ".claude/skills/particulars/SKILL.md"

func (a *app) skillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Show or install the agent-facing skill embedded in this binary",
		Long: `The skill teaches an agent the verbs and the recall-before-assert discipline.
It ships inside the binary, stamped with this version, so skill and verbs
cannot drift. Neither subcommand needs a workspace.`,
	}
	cmd.AddCommand(a.skillShowCmd(), a.skillInstallCmd())
	return cmd
}

func (a *app) skillShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the skill to stdout",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			content := skill.Render(version)
			return a.emit(map[string]any{"version": skill.NormaliseVersion(version), "content": string(content)}, func(w io.Writer) {
				_, _ = w.Write(content)
			})
		}),
	}
}

func (a *app) skillInstallCmd() *cobra.Command {
	var user, force, check bool
	var dir string
	cmd := &cobra.Command{
		Use:   "install [--user | --dir <path>] [--force] [--check]",
		Short: "Write SKILL.md where Claude Code loads skills (project by default)",
		Long: `Targets: ./` + skillRelPath + ` (default), ~/` + skillRelPath + ` (--user),
or <dir>/SKILL.md (--dir). A file this tool did not write (no marker line) is
never overwritten without --force. --check compares without writing and exits
4 on drift, ignoring the stamped version — use it in CI.`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if user && dir != "" {
				return usageErr("--user and --dir are mutually exclusive")
			}
			target, err := skillTarget(user, dir)
			if err != nil {
				return err
			}
			rendered := skill.Render(version)
			v := skill.NormaliseVersion(version)
			existing, readErr := os.ReadFile(target)
			exists := readErr == nil
			if readErr != nil && !os.IsNotExist(readErr) {
				return readErr
			}

			if check {
				status := "ok"
				switch {
				case !exists:
					status = "missing"
				case !skill.HasMarker(existing):
					status = "foreign"
				case !skill.BodyEqual(existing, rendered):
					status = "differs"
				}
				if err := a.emit(map[string]any{"path": target, "status": status, "version": v}, func(w io.Writer) {
					fmt.Fprintf(w, "%s: %s\n", target, status)
				}); err != nil {
					return err
				}
				if status != "ok" {
					return checkFailedErr("installed skill is %s; run `particulars skill install`", status)
				}
				return nil
			}

			result := map[string]any{"path": target, "version": v}
			switch {
			case exists && bytes.Equal(existing, rendered):
				result["unchanged"] = true
			case exists && !skill.HasMarker(existing) && !force:
				return &ExitError{Code: ExitRuntime, ErrCode: "conflict", Err: fmt.Errorf("%s exists and was not written by particulars; pass --force to replace it", target)}
			default:
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return err
				}
				if err := writeAtomicFile(target, rendered); err != nil {
					return err
				}
				if exists {
					result["updated"] = true
				} else {
					result["created"] = true
				}
			}
			return a.emit(result, func(w io.Writer) {
				switch {
				case result["created"] == true:
					fmt.Fprintf(w, "Installed skill (particulars %s) at %s\n", v, target)
				case result["updated"] == true:
					fmt.Fprintf(w, "Updated skill (particulars %s) at %s\n", v, target)
				default:
					fmt.Fprintf(w, "Skill already up to date at %s\n", target)
				}
			})
		}),
	}
	cmd.Flags().BoolVar(&user, "user", false, "install to ~/"+skillRelPath)
	cmd.Flags().StringVar(&dir, "dir", "", "install to <dir>/SKILL.md")
	cmd.Flags().BoolVar(&force, "force", false, "replace a skill file not written by particulars")
	cmd.Flags().BoolVar(&check, "check", false, "verify the installed skill without writing; exit 4 on drift")
	return cmd
}

func skillTarget(user bool, dir string) (string, error) {
	var p string
	switch {
	case dir != "":
		p = filepath.Join(dir, "SKILL.md")
	case user:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, filepath.FromSlash(skillRelPath))
	default:
		p = filepath.FromSlash(skillRelPath)
	}
	return filepath.Abs(p)
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
