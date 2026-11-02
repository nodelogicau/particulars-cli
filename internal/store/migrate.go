package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// Migration from dkf/0.1 to dkf/0.2 (workspace-layout, "Migration changes
// nothing an object asserts"): rewrite format, rename every object file
// byte-identical to the path its id derives, regenerate the index. The order
// is chosen so that a migration cut short is loud, never silent: format goes
// first, so unmoved files are misplaced — reported by every verb — rather
// than hidden in subdirectories a flat reader skips. Running it again
// finishes the job.

// ErrMigrationRefused is returned when a precondition fails; nothing has
// been written.
var ErrMigrationRefused = errors.New("migration refused")

// Move is one file a migration relocates, by root-relative path.
type Move struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	// copied marks a target already holding identical bytes: an earlier run
	// renamed it and the source is the stale half of an interrupted move.
	copied bool
}

// MigrationPlan is everything a migration would do, computed without writing.
type MigrationPlan struct {
	// FormatEdit is true while dkf.yaml still says dkf/0.1.
	FormatEdit bool     `json:"format_edit"`
	Moves      []Move   `json:"moves"`
	Legacy     []string `json:"legacy"`
	// Sealed lists the months whose segments the migrated index writes.
	Sealed   []string      `json:"sealed"`
	Problems []FileProblem `json:"-"`
	// formatLine is the byte range of the one line to rewrite.
	formatLine [2]int
	config     []byte
}

// Nothing reports whether the plan has no file to move and no format to edit.
func (p *MigrationPlan) Nothing() bool { return !p.FormatEdit && len(p.Moves) == 0 }

// formatLine01 is the one dkf.yaml line a migration rewrites. A comment after
// it is kept; anything more unusual is left to a person.
var formatLine01 = regexp.MustCompile(`(?m)^format:[ \t]*dkf/0\.1([ \t]+#.*)?[ \t]*$`)

// PlanMigration computes a migration to dkf/0.2 and checks its
// preconditions, reading the type directories' files directly: a workspace
// part-way through a migration cannot be loaded, since its unmoved files are
// misplaced.
func (w *Workspace) PlanMigration() (*MigrationPlan, error) {
	p := &MigrationPlan{Moves: []Move{}, Legacy: []string{}, Sealed: []string{}}
	cfgPath := filepath.Join(w.Root, ConfigFile)
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	p.config = cfg
	if w.Config.Format == dkf.Format01 {
		locs := formatLine01.FindAllIndex(cfg, -1)
		if len(locs) != 1 {
			p.Problems = append(p.Problems, FileProblem{Path: ConfigFile, Code: "format_line",
				Message: "expected exactly one plain `format: dkf/0.1` line; set `format: dkf/0.2` by hand and run `particulars migrate` again"})
		} else {
			p.FormatEdit, p.formatLine = true, [2]int{locs[0][0], locs[0][1]}
		}
	}

	var ms months
	monthsSeen := map[int]bool{}
	retractions := []int{}
	note := func(obj dkf.Object) {
		if m, ok := idMonth(obj.ObjectID()); ok {
			ms.see(m)
			monthsSeen[m] = true
		}
		if rm, ok := retractionMonth(obj); ok {
			retractions = append(retractions, rm)
		}
	}
	// Objects already in place on a resumed run count toward the months.
	if w.Sharded() {
		g, _ := w.Load()
		for _, obj := range g.Objects() {
			note(obj)
		}
	}

	for _, t := range allTypes {
		dir := w.Dir(t)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if e.IsDir() {
				if w.Sharded() && (monthDir.MatchString(name) || name == LegacyShard) {
					continue
				}
				files, _ := os.ReadDir(filepath.Join(dir, name))
				for _, f := range files {
					if objectFile(f) {
						p.Problems = append(p.Problems, FileProblem{Path: w.Rel(filepath.Join(dir, name, f.Name())), Code: "foreign_directory",
							Message: fmt.Sprintf("%s/ is not part of the layout; move or delete its files before migrating", w.Rel(filepath.Join(dir, name)))})
					}
				}
				continue
			}
			if !objectFile(e) {
				continue
			}
			from := filepath.Join(dir, name)
			obj, prob := readTopLevel(w, t, from)
			if prob != nil {
				p.Problems = append(p.Problems, *prob)
				continue
			}
			note(obj)
			id := obj.ObjectID()
			to := filepath.Join(dir, shard(id), name)
			mv := Move{ID: id, From: w.Rel(from), To: w.Rel(to)}
			if existing, err := os.ReadFile(to); err == nil {
				src, _ := os.ReadFile(from)
				if !bytes.Equal(existing, src) {
					p.Problems = append(p.Problems, FileProblem{Path: mv.From, Code: "target_exists",
						Message: fmt.Sprintf("%s already exists with different content", mv.To)})
					continue
				}
				mv.copied = true
			}
			p.Moves = append(p.Moves, mv)
			if _, ok := idMonth(id); !ok {
				p.Legacy = append(p.Legacy, id)
			}
		}
	}

	sealed := map[int]bool{}
	for m := range monthsSeen {
		if !ms.open(m) {
			sealed[m] = true
		}
	}
	for _, rm := range retractions {
		if !ms.open(rm) {
			sealed[rm] = true
		}
	}
	for m := range sealed {
		p.Sealed = append(p.Sealed, monthName(m))
	}
	sort.Strings(p.Sealed)
	sort.Slice(p.Moves, func(i, j int) bool { return p.Moves[i].From < p.Moves[j].From })
	sort.Strings(p.Legacy)
	return p, nil
}

// readTopLevel parses an object file at the top of type directory t with the
// checks Load makes, returning a problem instead of an object when they fail.
func readTopLevel(w *Workspace, t dkf.Type, path string) (dkf.Object, *FileProblem) {
	rel := w.Rel(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &FileProblem{Path: rel, Code: "parse_error", Message: err.Error()}
	}
	obj, err := dkf.Decode(data)
	if err != nil {
		return nil, &FileProblem{Path: rel, Code: "parse_error", Message: err.Error()}
	}
	if fileID := strings.TrimSuffix(filepath.Base(path), ".yaml"); obj.ObjectID() != fileID {
		return nil, &FileProblem{Path: rel, Code: "id_mismatch", Message: fmt.Sprintf("file name implies %s but id field is %q", fileID, obj.ObjectID())}
	}
	if idType, err := dkf.TypeOfID(obj.ObjectID()); err != nil || idType != t || obj.ObjectType() != t {
		return nil, &FileProblem{Path: rel, Code: "type_mismatch", Message: fmt.Sprintf("file is in %s/ but its id or type is not %s", dirFor(t), t)}
	}
	return obj, nil
}

// Migrate carries out a plan: the format line, then the moves, then the
// index. It refuses a plan with problems, before writing anything.
func (w *Workspace) Migrate(p *MigrationPlan) error {
	if len(p.Problems) > 0 {
		lines := make([]string, len(p.Problems))
		for i, pr := range p.Problems {
			lines[i] = pr.Error()
		}
		return fmt.Errorf("%w: %s", ErrMigrationRefused, strings.Join(lines, "; "))
	}
	if p.FormatEdit {
		cfg := append(append(append([]byte{}, p.config[:p.formatLine[0]]...),
			formatLine01.ReplaceAll(p.config[p.formatLine[0]:p.formatLine[1]], []byte("format: dkf/0.2${1}"))...), p.config[p.formatLine[1]:]...)
		if err := writeAtomic(filepath.Join(w.Root, ConfigFile), cfg); err != nil {
			return err
		}
		w.Config.Format = dkf.Format02
	}
	for _, m := range p.Moves {
		from, to := filepath.Join(w.Root, filepath.FromSlash(m.From)), filepath.Join(w.Root, filepath.FromSlash(m.To))
		if m.copied {
			if err := os.Remove(from); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		if err := os.Rename(from, to); err != nil {
			return err
		}
	}
	// The flat index still in place is read as the committed root, so its
	// unknown entry types and fields carry into the documents their ids place
	// them in. No sealed segment exists yet, so every one is written.
	_, err := w.rebuildSharded()
	return err
}
