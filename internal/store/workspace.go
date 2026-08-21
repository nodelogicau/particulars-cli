package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// EnvWorkspace is the environment variable consulted when no --workspace flag is given.
const EnvWorkspace = "DKF_WORKSPACE"

// PointerFile is the optional redirect file honoured during discovery: its
// first non-blank, non-comment line is a path (relative to the file's
// directory, or absolute) to a directory containing dkf.yaml.
const PointerFile = ".dkf"

// Resolution records how a workspace was found.
type Resolution struct {
	Root    string `json:"root"`
	Via     string `json:"via"`               // flag | env | dkf.yaml | pointer
	Pointer string `json:"pointer,omitempty"` // absolute path of the .dkf file, when via=pointer
}

// Sentinel errors callers map to exit codes.
var (
	ErrNoWorkspace      = errors.New("no workspace found: run `particulars init` or set --workspace / DKF_WORKSPACE")
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("already exists")
	ErrAlreadyRetracted = errors.New("already retracted")
	ErrInvalidBaseURI   = errors.New("invalid base-uri")
)

// Workspace is an opened DKF directory.
type Workspace struct {
	Root   string
	Config Config
}

// Discover locates a workspace: explicit dir, then $DKF_WORKSPACE, then an
// upward search from the working directory for dkf.yaml or a .dkf pointer.
func Discover(explicit string) (*Workspace, error) {
	w, _, err := DiscoverWith(explicit)
	return w, err
}

// DiscoverWith is Discover plus a record of how the workspace was resolved.
func DiscoverWith(explicit string) (*Workspace, *Resolution, error) {
	if explicit != "" {
		w, err := Open(explicit)
		if err != nil {
			return nil, nil, err
		}
		return w, &Resolution{Root: w.Root, Via: "flag"}, nil
	}
	if env := os.Getenv(EnvWorkspace); env != "" {
		w, err := Open(env)
		if err != nil {
			return nil, nil, err
		}
		return w, &Resolution{Root: w.Root, Via: "env"}, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil {
			w, err := Open(dir)
			if err != nil {
				return nil, nil, err
			}
			return w, &Resolution{Root: w.Root, Via: "dkf.yaml"}, nil
		}
		pointer := filepath.Join(dir, PointerFile)
		if target, ok, perr := readPointer(pointer); perr != nil {
			return nil, nil, perr
		} else if ok {
			if _, err := os.Stat(filepath.Join(target, ConfigFile)); err != nil {
				return nil, nil, fmt.Errorf("%w: %s points at %s, which has no %s", ErrNoWorkspace, pointer, target, ConfigFile)
			}
			w, err := Open(target)
			if err != nil {
				return nil, nil, err
			}
			return w, &Resolution{Root: w.Root, Via: "pointer", Pointer: pointer}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil, ErrNoWorkspace
		}
		dir = parent
	}
}

// readPointer parses a .dkf file. ok is false when the file does not exist.
func readPointer(path string) (target string, ok bool, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(filepath.Dir(path), line)
		}
		return filepath.Clean(line), true, nil
	}
	return "", false, fmt.Errorf("%w: %s is empty", ErrNoWorkspace, path)
}

// WritePointer writes <dir>/.dkf containing target (as given). It refuses,
// with ErrAlreadyExists, to replace a pointer with different content.
func WritePointer(dir, target string) (string, error) {
	path := filepath.Join(dir, PointerFile)
	content := target + "\n"
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == content {
			return path, nil
		}
		return path, fmt.Errorf("%s %w with different content (%q)", path, ErrAlreadyExists, strings.TrimSpace(string(existing)))
	}
	return path, writeAtomic(path, []byte(content))
}

// Open loads the workspace rooted at dir.
func Open(dir string) (*Workspace, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	cfg, err := readConfig(filepath.Join(abs, ConfigFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w (no %s in %s)", ErrNoWorkspace, ConfigFile, abs)
	}
	if err != nil {
		return nil, err
	}
	return &Workspace{Root: abs, Config: cfg}, nil
}

// Init creates a new workspace at dir. It fails if dkf.yaml already exists.
func Init(dir string, cfg Config) (*Workspace, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(abs, ConfigFile)
	if _, err := os.Stat(cfgPath); err == nil {
		return nil, fmt.Errorf("%s %w in %s", ConfigFile, ErrAlreadyExists, abs)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	for _, t := range allTypes {
		if err := os.MkdirAll(filepath.Join(abs, dirFor(t)), 0o755); err != nil {
			return nil, err
		}
	}
	data, err := encodeConfig(cfg)
	if err != nil {
		return nil, err
	}
	if err := writeExclusive(cfgPath, data); err != nil {
		return nil, err
	}
	w := &Workspace{Root: abs, Config: cfg}
	if err := w.WriteIndex(&Index{Format: dkf.FormatVersion}); err != nil {
		return nil, err
	}
	return w, nil
}

// allTypes lists every object and record type in load order.
var allTypes = []dkf.Type{dkf.TypeParticular, dkf.TypeClaim, dkf.TypeSynthesis, dkf.TypeMerge}

func dirFor(t dkf.Type) string {
	switch t {
	case dkf.TypeParticular:
		return "particulars"
	case dkf.TypeClaim:
		return "claims"
	case dkf.TypeSynthesis:
		return "syntheses"
	case dkf.TypeMerge:
		return "merges"
	}
	return ""
}

// Dir returns the absolute directory holding objects of type t.
func (w *Workspace) Dir(t dkf.Type) string { return filepath.Join(w.Root, dirFor(t)) }

// Path returns the file path for an object id, deriving the directory from the prefix.
func (w *Workspace) Path(id string) (string, error) {
	t, err := dkf.TypeOfID(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(w.Dir(t), id+".yaml"), nil
}

// Rel returns p relative to the workspace root, for display.
func (w *Workspace) Rel(p string) string {
	if r, err := filepath.Rel(w.Root, p); err == nil {
		return filepath.ToSlash(r)
	}
	return p
}

// Exists reports whether an object file exists for id.
func (w *Workspace) Exists(id string) bool {
	p, err := w.Path(id)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Read loads one object by id.
func (w *Workspace) Read(id string) (dkf.Object, error) {
	p, err := w.Path(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	obj, err := dkf.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", w.Rel(p), err)
	}
	return obj, nil
}

// ReadAssertion loads a claim or synthesis by id.
func (w *Workspace) ReadAssertion(id string) (dkf.Assertion, error) {
	if !dkf.IsAssertionID(id) {
		return nil, fmt.Errorf("%s: not a claim or synthesis id", id)
	}
	obj, err := w.Read(id)
	if err != nil {
		return nil, err
	}
	a, ok := obj.(dkf.Assertion)
	if !ok {
		return nil, fmt.Errorf("%s: file does not contain a claim or synthesis", id)
	}
	return a, nil
}

// Create validates and writes a new object file with create-exclusive
// semantics, creating the type directory if needed. It never overwrites.
func (w *Workspace) Create(obj dkf.Object) error {
	if ps := dkf.ValidateObject(obj); len(ps) > 0 {
		return ps
	}
	p, err := w.Path(obj.ObjectID())
	if err != nil {
		return err
	}
	data, err := dkf.Encode(obj)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return writeExclusive(p, data)
}

// WriteParticular rewrites a particular file. Particulars are the only
// mutable object type ("create or update" in the spec).
func (w *Workspace) WriteParticular(p *dkf.Particular) error {
	if ps := dkf.ValidateParticular(p); len(ps) > 0 {
		return ps
	}
	path, err := w.Path(p.ID)
	if err != nil {
		return err
	}
	data, err := dkf.Encode(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeAtomic(path, data)
}

// Retract appends a retracted block to an existing claim, synthesis, or merge
// file without altering existing bytes, re-parses to confirm validity, and
// restores the original on failure. Returns the updated object.
func (w *Workspace) Retract(id string, r *dkf.Retracted) (dkf.Retractable, error) {
	if ps := dkf.ValidateRetracted(r); len(ps) > 0 {
		return nil, ps
	}
	path, err := w.Path(id)
	if err != nil {
		return nil, err
	}
	if !dkf.IsRetractableID(id) {
		return nil, fmt.Errorf("%s: only claims, syntheses, and merges can be retracted", id)
	}
	if t, _ := dkf.TypeOfID(id); t == dkf.TypeMerge && r.SupersededBy != "" {
		return nil, &dkf.Problem{Code: dkf.CodeInvalidID, Field: "superseded-by", Message: "a merge is undone, not superseded; --superseded-by is not allowed for merge records"}
	}
	original, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	obj, err := dkf.Decode(original)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", w.Rel(path), err)
	}
	existing, ok := obj.(dkf.Retractable)
	if !ok {
		return nil, fmt.Errorf("%s: not a retractable object", id)
	}
	if existing.GetRetracted() != nil {
		return nil, fmt.Errorf("%s: %w", id, ErrAlreadyRetracted)
	}
	block, err := dkf.EncodeRetracted(r)
	if err != nil {
		return nil, err
	}
	appended := make([]byte, 0, len(original)+len(block)+1)
	appended = append(appended, original...)
	if len(original) > 0 && !bytes.HasSuffix(original, []byte("\n")) {
		appended = append(appended, '\n')
	}
	appended = append(appended, block...)

	if err := writeAtomic(path, appended); err != nil {
		return nil, err
	}
	reread, err := os.ReadFile(path)
	if err == nil {
		var parsed dkf.Object
		parsed, err = dkf.Decode(reread)
		if err == nil {
			a, ok := parsed.(dkf.Retractable)
			switch {
			case !ok:
				err = fmt.Errorf("re-parse yielded %T", parsed)
			case a.GetRetracted() == nil:
				err = fmt.Errorf("retracted block not visible after append")
			default:
				if ps := dkf.ValidateObject(parsed); len(ps) > 0 {
					err = ps
				} else {
					return a, nil
				}
			}
		}
	}
	// Restore and report.
	if rerr := writeAtomic(path, original); rerr != nil {
		return nil, fmt.Errorf("retraction failed (%v) and restore failed (%v): %s may be corrupt", err, rerr, w.Rel(path))
	}
	return nil, fmt.Errorf("retraction of %s aborted, file restored: %w", id, err)
}

// UpsertParticular implements idempotent define: if a particular with uri
// exists it is updated (label replaced; aliases unioned with the old label and
// the supplied aliases), otherwise a new one is created. Returns created=true
// for a new particular.
func (w *Workspace) UpsertParticular(uri, label string, aliases []string) (*dkf.Particular, bool, error) {
	g, err := w.Load()
	if err != nil {
		return nil, false, err
	}
	if existing := g.ParticularByURI(uri); existing != nil {
		union := []string{}
		seen := map[string]bool{}
		add := func(a string) {
			a = strings.TrimSpace(a)
			if a == "" || seen[strings.ToLower(a)] || strings.EqualFold(a, label) {
				return
			}
			seen[strings.ToLower(a)] = true
			union = append(union, a)
		}
		for _, a := range existing.Aliases {
			add(a)
		}
		if existing.Label != label {
			add(existing.Label)
		}
		for _, a := range aliases {
			add(a)
		}
		existing.Label = label
		existing.Aliases = union
		if err := w.WriteParticular(existing); err != nil {
			return nil, false, err
		}
		if err := w.UpsertIndex(existing); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	p := &dkf.Particular{ID: dkf.NewID(dkf.TypeParticular), URI: uri, Label: label}
	seen := map[string]bool{}
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" || seen[strings.ToLower(a)] || strings.EqualFold(a, label) {
			continue
		}
		seen[strings.ToLower(a)] = true
		p.Aliases = append(p.Aliases, a)
	}
	if err := w.Create(p); err != nil {
		return nil, false, err
	}
	if err := w.UpsertIndex(p); err != nil {
		return nil, false, err
	}
	return p, true, nil
}

// CreateMerge writes a merge record joining two URIs. URIs are stored sorted
// so the same pair always serialises identically. It refuses a self-merge and
// a pair already joined by a non-retracted merge.
func (w *Workspace) CreateMerge(uriA, uriB, reason string, src dkf.Source, ts time.Time) (*dkf.Merge, error) {
	uriA, uriB = strings.TrimSpace(uriA), strings.TrimSpace(uriB)
	if uriA == "" || uriB == "" {
		return nil, &dkf.Problem{Code: dkf.CodeInvalidMerge, Field: "uris", Message: "both uris are required"}
	}
	if uriA == uriB {
		return nil, &dkf.Problem{Code: dkf.CodeInvalidMerge, Field: "uris", Message: "cannot merge a uri with itself"}
	}
	if uriA > uriB {
		uriA, uriB = uriB, uriA
	}
	g, err := w.Load()
	if err != nil {
		return nil, err
	}
	if existing := g.MergeBetween(uriA, uriB); existing != nil {
		return nil, &dkf.Problem{Code: dkf.CodeInvalidMerge, Field: "uris", Message: fmt.Sprintf("already joined by %s", existing.ID)}
	}
	m := &dkf.Merge{ID: dkf.NewID(dkf.TypeMerge), URIs: []string{uriA, uriB}, Reason: reason, Source: src, Timestamp: ts}
	if err := w.Create(m); err != nil {
		return nil, err
	}
	if err := w.UpsertIndex(m); err != nil {
		return nil, err
	}
	return m, nil
}

// --- low-level writes ----------------------------------------------------

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%s %w", filepath.Base(path), ErrAlreadyExists)
	}
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
