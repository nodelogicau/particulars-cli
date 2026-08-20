package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// IndexFile is the derived manifest name.
const IndexFile = "index.yaml"

// Index is the derived manifest. It follows the spec shape plus additive
// fields (scope, topics, timestamp, retracted) that let recall filter without
// opening files. It is always regenerable from the object files.
type Index struct {
	Format  string  `yaml:"format"`
	Entries []Entry `yaml:"entries"`
}

// Entry is one index row. Field order is the on-disk order.
type Entry struct {
	ID        string   `yaml:"id" json:"id"`
	Type      dkf.Type `yaml:"type" json:"type"`
	URI       string   `yaml:"uri,omitempty" json:"uri,omitempty"`
	Subject   string   `yaml:"subject,omitempty" json:"subject,omitempty"`
	Scope     string   `yaml:"scope,omitempty" json:"scope,omitempty"`
	Topics    []string `yaml:"topics,omitempty" json:"topics,omitempty"`
	Timestamp string   `yaml:"timestamp,omitempty" json:"timestamp,omitempty"`
	Inputs    []string `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Retracted bool     `yaml:"retracted,omitempty" json:"retracted,omitempty"`
}

// EntryFor derives the index row for an object.
func EntryFor(obj dkf.Object) Entry {
	switch o := obj.(type) {
	case *dkf.Particular:
		return Entry{ID: o.ID, Type: dkf.TypeParticular, URI: o.URI}
	case dkf.Assertion:
		ctx := o.GetContext()
		e := Entry{
			ID:        o.ObjectID(),
			Type:      o.ObjectType(),
			Subject:   o.SubjectID(),
			Scope:     string(ctx.Scope),
			Topics:    ctx.Topics,
			Timestamp: dkf.FormatTime(o.GetTimestamp()),
			Inputs:    o.InputIDs(),
			Retracted: o.GetRetracted() != nil,
		}
		if o.GetTimestamp().IsZero() {
			e.Timestamp = ""
		}
		return e
	}
	return Entry{ID: obj.ObjectID(), Type: obj.ObjectType()}
}

// BuildIndex derives a complete index from a loaded graph.
func BuildIndex(g *Graph) *Index {
	idx := &Index{Format: dkf.FormatVersion, Entries: []Entry{}}
	for _, obj := range g.Objects() {
		idx.Entries = append(idx.Entries, EntryFor(obj))
	}
	idx.sort()
	return idx
}

func (idx *Index) sort() {
	sort.Slice(idx.Entries, func(i, j int) bool { return idx.Entries[i].ID < idx.Entries[j].ID })
}

// Upsert inserts or replaces an entry by id.
func (idx *Index) Upsert(e Entry) {
	for i := range idx.Entries {
		if idx.Entries[i].ID == e.ID {
			idx.Entries[i] = e
			idx.sort()
			return
		}
	}
	idx.Entries = append(idx.Entries, e)
	idx.sort()
}

// EncodeIndex renders the index deterministically.
func EncodeIndex(idx *Index) ([]byte, error) {
	if idx.Entries == nil {
		idx.Entries = []Entry{}
	}
	var root yaml.Node
	if err := root.Encode(idx); err != nil {
		return nil, err
	}
	bareTimestamps(&root)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// bareTimestamps re-tags `timestamp` values so they render unquoted, matching
// the object files (yaml.v3 would otherwise quote timestamp-looking strings).
func bareTimestamps(n *yaml.Node) {
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Value == "timestamp" && v.Kind == yaml.ScalarNode && v.Value != "" {
				v.Tag = "!!timestamp"
				v.Style = 0
			}
		}
	}
	for _, c := range n.Content {
		bareTimestamps(c)
	}
}

// IndexPath returns the absolute index file path.
func (w *Workspace) IndexPath() string { return filepath.Join(w.Root, IndexFile) }

// ReadIndex parses the committed index. Returns ErrNotFound if absent.
func (w *Workspace) ReadIndex() (*Index, []byte, error) {
	data, err := os.ReadFile(w.IndexPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("%s: %w", IndexFile, ErrNotFound)
	}
	if err != nil {
		return nil, nil, err
	}
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, data, fmt.Errorf("%s: %w", IndexFile, err)
	}
	return &idx, data, nil
}

// WriteIndex writes the index atomically.
func (w *Workspace) WriteIndex(idx *Index) error {
	data, err := EncodeIndex(idx)
	if err != nil {
		return err
	}
	return writeAtomic(w.IndexPath(), data)
}

// RebuildIndex regenerates index.yaml from the object files, ignoring any
// existing content. Files that fail to load are skipped (they are reported by
// validate) so a rebuild always succeeds on an otherwise readable tree.
func (w *Workspace) RebuildIndex() (*Index, error) {
	g, err := w.Load()
	if err != nil {
		return nil, err
	}
	idx := BuildIndex(g)
	if err := w.WriteIndex(idx); err != nil {
		return nil, err
	}
	return idx, nil
}

// UpsertIndex updates the entry for obj. If the index is missing or
// unparseable it falls back to a full rebuild.
func (w *Workspace) UpsertIndex(obj dkf.Object) error {
	idx, _, err := w.ReadIndex()
	if err != nil {
		_, err = w.RebuildIndex()
		return err
	}
	idx.Format = dkf.FormatVersion
	idx.Upsert(EntryFor(obj))
	return w.WriteIndex(idx)
}

// IndexDiff describes how the committed index differs from a rebuild.
type IndexDiff struct {
	Missing []string `json:"missing"` // in files, not in index
	Extra   []string `json:"extra"`   // in index, not in files
	Changed []string `json:"changed"` // present in both, content differs
	// BytesDiffer is true when the files differ byte-for-byte even if the
	// entry sets are equal (ordering, formatting, or format field).
	BytesDiffer bool `json:"bytes_differ"`
}

// Clean reports whether the committed index matches a rebuild exactly.
func (d IndexDiff) Clean() bool {
	return !d.BytesDiffer && len(d.Missing) == 0 && len(d.Extra) == 0 && len(d.Changed) == 0
}

// CheckIndex compares the committed index with a fresh rebuild without writing.
// A missing index is reported as every entry missing.
func (w *Workspace) CheckIndex() (IndexDiff, error) {
	g, err := w.Load()
	if err != nil {
		return IndexDiff{}, err
	}
	want := BuildIndex(g)
	wantBytes, err := EncodeIndex(want)
	if err != nil {
		return IndexDiff{}, err
	}
	var d IndexDiff
	d.Missing, d.Extra, d.Changed = []string{}, []string{}, []string{}
	got, gotBytes, err := w.ReadIndex()
	if err != nil {
		for _, e := range want.Entries {
			d.Missing = append(d.Missing, e.ID)
		}
		d.BytesDiffer = true
		return d, nil
	}
	d.BytesDiffer = !bytes.Equal(wantBytes, gotBytes)
	gotBy := map[string]Entry{}
	for _, e := range got.Entries {
		gotBy[e.ID] = e
	}
	wantBy := map[string]Entry{}
	for _, e := range want.Entries {
		wantBy[e.ID] = e
		ge, ok := gotBy[e.ID]
		switch {
		case !ok:
			d.Missing = append(d.Missing, e.ID)
		case !entriesEqual(ge, e):
			d.Changed = append(d.Changed, e.ID)
		}
	}
	for _, e := range got.Entries {
		if _, ok := wantBy[e.ID]; !ok {
			d.Extra = append(d.Extra, e.ID)
		}
	}
	sort.Strings(d.Missing)
	sort.Strings(d.Extra)
	sort.Strings(d.Changed)
	return d, nil
}

func entriesEqual(a, b Entry) bool {
	ab, _ := yaml.Marshal(a)
	bb, _ := yaml.Marshal(b)
	return bytes.Equal(ab, bb)
}
