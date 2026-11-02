package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// The dkf/0.2 index (workspace-layout, index-manifest): a root index.yaml
// holding the open months — the newest id's month and the one before it —
// and one segment per sealed month, plus index/legacy.yaml for ids that
// carry no month. A segment, once written, is never rewritten.

// LegacyIndex is the dkf/0.2 document for ids from which no month derives.
const LegacyIndex = "index/legacy.yaml"

// SegmentFor names the segment that holds id's entry once its month seals.
func SegmentFor(id string) string {
	if m, ok := idMonth(id); ok {
		return segmentPath(m)
	}
	return LegacyIndex
}

// segmentPath is a sealed month's document, relative to the workspace root.
func segmentPath(month int) string { return "index/" + monthName(month) + ".yaml" }

// ErrSealedMonth refuses a write into a month the workspace has sealed. Only
// a clock more than a month behind the workspace can reach it.
var ErrSealedMonth = errors.New("sealed month")

// SealedSegmentsError reports sealed segments a rebuild found differing from
// the files and left alone, since a sealed segment is never rewritten.
type SealedSegmentsError struct{ Paths []string }

func (e *SealedSegmentsError) Error() string {
	return fmt.Sprintf("%s differ%s from the files and %s sealed; a sealed segment is never rewritten — run `particulars validate`",
		strings.Join(e.Paths, ", "), map[bool]string{true: "s", false: ""}[len(e.Paths) == 1], map[bool]string{true: "is", false: "are"}[len(e.Paths) == 1])
}

// monthNum counts calendar months, so sealing is integer comparison.
func monthNum(t time.Time) int {
	t = t.UTC()
	return t.Year()*12 + int(t.Month()) - 1
}

func monthName(n int) string { return fmt.Sprintf("%04d-%02d", n/12, n%12+1) }

// idMonth is the minting month of a UUIDv7 id.
func idMonth(id string) (int, bool) {
	t, ok := dkf.MintedAt(id)
	if !ok {
		return 0, false
	}
	return monthNum(t), true
}

// months places objects for one workspace state. With no UUIDv7 id there is
// no current month and nothing is sealed.
type months struct {
	cur int
	has bool
}

// open reports whether month m is open (or later): the current month, the
// one before it, or anything after.
func (p months) open(m int) bool { return !p.has || m >= p.cur-1 }

// see widens the current month to include m.
func (p *months) see(m int) {
	if !p.has || m > p.cur {
		p.cur, p.has = m, true
	}
}

// ShardedIndex is a complete dkf/0.2 index: the root and every other
// document, keyed by path relative to the workspace root.
type ShardedIndex struct {
	Root *Index
	Docs map[string]*Index
	// months is the state the documents were built for.
	months months
}

func newSharded(format string) *ShardedIndex {
	return &ShardedIndex{Root: &Index{Format: format, kind: kindRoot}, Docs: map[string]*Index{}}
}

// doc returns the document at p, creating it with the kind its path implies.
func (s *ShardedIndex) doc(p string) *Index {
	if d, ok := s.Docs[p]; ok {
		return d
	}
	k := kindSegment
	if p == LegacyIndex {
		k = kindLegacy
	}
	d := &Index{Format: s.Root.Format, kind: k}
	s.Docs[p] = d
	return d
}

// home returns the document an id's entry belongs in.
func (s *ShardedIndex) home(p months, id string) *Index {
	m, ok := idMonth(id)
	switch {
	case !ok:
		return s.doc(LegacyIndex)
	case p.open(m):
		return s.Root
	}
	return s.doc(segmentPath(m))
}

// finish sorts every list and regenerates the root's segment list: legacy
// first, then sealed months ascending, each with its entry count.
func (s *ShardedIndex) finish() {
	paths := make([]string, 0, len(s.Docs))
	for p, d := range s.Docs {
		d.sort()
		sort.Strings(d.Retracted)
		if p == LegacyIndex && len(d.Entries)+len(d.Unknown) == 0 {
			delete(s.Docs, p)
			continue
		}
		paths = append(paths, p)
	}
	s.Root.sort()
	sort.Strings(s.Root.Retracted)
	sort.Slice(paths, func(i, j int) bool {
		if (paths[i] == LegacyIndex) != (paths[j] == LegacyIndex) {
			return paths[i] == LegacyIndex
		}
		return paths[i] < paths[j]
	})
	s.Root.Segments = []SegmentRef{}
	for _, p := range paths {
		d := s.Docs[p]
		s.Root.Segments = append(s.Root.Segments, SegmentRef{Path: p, Count: len(d.Entries) + len(d.Unknown)})
	}
}

// retractionMonth is the month an object's retraction is dated in.
func retractionMonth(obj dkf.Object) (int, bool) {
	r, ok := obj.(dkf.Retractable)
	if !ok || r.GetRetracted() == nil {
		return 0, false
	}
	return monthNum(r.GetRetracted().Timestamp), true
}

// shardedEntry is an object's dkf/0.2 entry: retraction lives in the lists.
func shardedEntry(obj dkf.Object) Entry {
	e := EntryFor(obj)
	e.Retracted = false
	return e
}

// BuildSharded derives the complete dkf/0.2 index from a loaded graph — a
// function of the files alone, the open months following the newest id.
func BuildSharded(g *Graph, format string) *ShardedIndex {
	var p months
	for _, obj := range g.Objects() {
		if m, ok := idMonth(obj.ObjectID()); ok {
			p.see(m)
		}
	}
	s := newSharded(format)
	s.months = p
	for _, obj := range g.Objects() {
		d := s.home(p, obj.ObjectID())
		d.Entries = append(d.Entries, shardedEntry(obj))
		if rm, ok := retractionMonth(obj); ok {
			if p.open(rm) {
				s.Root.Retracted = append(s.Root.Retracted, obj.ObjectID())
			} else {
				d := s.doc(segmentPath(rm))
				d.Retracted = append(d.Retracted, obj.ObjectID())
			}
		}
	}
	s.finish()
	return s
}

// months of a built or committed index: the newest month among the root's
// entries, preserved rows included. A root with none has no current month.
func rootMonths(root *Index) months {
	var p months
	for _, e := range root.Entries {
		if m, ok := idMonth(e.ID); ok {
			p.see(m)
		}
	}
	for _, n := range root.Unknown {
		if m, ok := idMonth(nodeField(n, "id")); ok {
			p.see(m)
		}
	}
	return p
}

// preserve carries a committed index's unknown entry types and unknown entry
// fields into a rebuilt one, each placed by its id as the rebuild places
// known entries.
func (s *ShardedIndex) preserve(committed []*Index) {
	p := s.months
	extra := map[string]Entry{}
	for _, c := range committed {
		for _, n := range c.Unknown {
			d := s.home(p, nodeField(n, "id"))
			d.Unknown = append(d.Unknown, n)
		}
		for _, e := range c.Entries {
			if e.Unknown != nil {
				extra[e.ID] = e
			}
		}
	}
	for _, d := range append([]*Index{s.Root}, mapValues(s.Docs)...) {
		for i := range d.Entries {
			if pe, ok := extra[d.Entries[i].ID]; ok {
				d.Entries[i].Unknown, d.Entries[i].raw = pe.Unknown, pe.raw
			}
		}
	}
	s.finish()
}

func mapValues(m map[string]*Index) []*Index {
	out := make([]*Index, 0, len(m))
	for _, d := range m {
		out = append(out, d)
	}
	return out
}

// readDoc parses one dkf/0.2 document by its path relative to the root.
func (w *Workspace) readDoc(rel string) (*Index, []byte, error) {
	data, err := os.ReadFile(filepath.Join(w.Root, filepath.FromSlash(rel)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("%s: %w", rel, ErrNotFound)
	}
	if err != nil {
		return nil, nil, err
	}
	var d Index
	if err := yaml.Unmarshal(data, &d); err != nil {
		return nil, data, fmt.Errorf("%s: %w", rel, err)
	}
	d.kind = kindSegment
	if rel == LegacyIndex {
		d.kind = kindLegacy
	}
	return &d, data, nil
}

// readRoot parses index.yaml as a dkf/0.2 root.
func (w *Workspace) readRoot() (*Index, []byte, error) {
	root, data, err := w.ReadIndex()
	if err != nil {
		return nil, data, err
	}
	root.kind = kindRoot
	return root, data, nil
}

// writeDoc writes one document atomically, and only if its bytes changed, so
// an operation that leaves a document as it was writes nothing.
func (w *Workspace) writeDoc(rel string, d *Index) error {
	data, err := EncodeIndex(d)
	if err != nil {
		return err
	}
	p := filepath.Join(w.Root, filepath.FromSlash(rel))
	if old, err := os.ReadFile(p); err == nil && bytes.Equal(old, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return writeAtomic(p, data)
}

// rebuildSharded regenerates the dkf/0.2 index from the files. It writes the
// root, the legacy document, and every sealed segment not yet written; a
// sealed segment that exists is never rewritten, and one that differs from
// the files is returned in a *SealedSegmentsError after the rest is written.
func (w *Workspace) rebuildSharded() (*ShardedIndex, error) {
	g, err := w.Load()
	if err != nil {
		return nil, err
	}
	s := BuildSharded(g, w.Config.Format)
	var committed []*Index
	committedCount := map[string]int{}
	if root, _, err := w.readRoot(); err == nil {
		committed = append(committed, root)
		for _, seg := range root.Segments {
			committedCount[seg.Path] = seg.Count
			if d, _, err := w.readDoc(seg.Path); err == nil {
				committed = append(committed, d)
				committedCount[seg.Path] = len(d.Entries) + len(d.Unknown)
			}
		}
	}
	s.preserve(committed)

	var differ []string
	for rel, d := range s.Docs {
		if rel == LegacyIndex {
			if err := w.writeDoc(rel, d); err != nil {
				return nil, err
			}
			continue
		}
		data, err := EncodeIndex(d)
		if err != nil {
			return nil, err
		}
		old, err := os.ReadFile(filepath.Join(w.Root, filepath.FromSlash(rel)))
		switch {
		case errors.Is(err, os.ErrNotExist):
			if err := w.writeDoc(rel, d); err != nil {
				return nil, err
			}
		case err != nil:
			return nil, err
		case !bytes.Equal(old, data):
			differ = append(differ, rel)
		}
	}
	// A segment left alone keeps the count of what it holds, so the late merge
	// stays visible to the next count check rather than being papered over.
	for i, seg := range s.Root.Segments {
		if slices.Contains(differ, seg.Path) {
			if n, ok := committedCount[seg.Path]; ok {
				s.Root.Segments[i].Count = n
			}
		}
	}
	if _, ok := s.Docs[LegacyIndex]; !ok {
		if err := os.Remove(filepath.Join(w.Root, filepath.FromSlash(LegacyIndex))); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	if err := w.writeDoc(IndexFile, s.Root); err != nil {
		return nil, err
	}
	if len(differ) > 0 {
		sort.Strings(differ)
		return s, &SealedSegmentsError{Paths: differ}
	}
	return s, nil
}

// upsertSharded applies one write to the dkf/0.2 index, touching only what
// the write touches: the root, the legacy document, or — when a mint opens a
// later month — the segments of the months that seal. It falls back to a
// rebuild whenever the committed index cannot be updated in place.
func (w *Workspace) upsertSharded(obj dkf.Object) error {
	rebuild := func() error {
		_, err := w.rebuildSharded()
		var sealed *SealedSegmentsError
		if errors.As(err, &sealed) {
			return nil // the write is done; validate reports the segment
		}
		return err
	}
	root, _, err := w.readRoot()
	if err != nil {
		return rebuild()
	}
	s := &ShardedIndex{Root: root, Docs: map[string]*Index{}}
	root.Format = w.Config.Format
	p := rootMonths(root)
	id := obj.ObjectID()

	if m, ok := idMonth(id); ok && (!p.has || m > p.cur) {
		next := months{cur: m, has: true}
		sealed, err := w.sealRoot(s, p, next)
		if err != nil {
			return err
		}
		if sealed == nil {
			return rebuild()
		}
		p = next
	}

	e := shardedEntry(obj)
	m, minted := idMonth(id)
	switch {
	case !minted:
		legacy, _, err := w.readDoc(LegacyIndex)
		if errors.Is(err, ErrNotFound) {
			legacy, err = &Index{Format: root.Format, kind: kindLegacy}, nil
		}
		if err != nil {
			return rebuild()
		}
		legacy.Format = root.Format
		legacy.Upsert(e)
		s.Docs[LegacyIndex] = legacy
	case p.open(m):
		root.Upsert(e)
	default:
		// A sealed month's entry mirrors immutable fields only; an update to
		// such an object (a particular's aliases, a retraction) changes no
		// entry, and its segment is never rewritten.
	}
	if rm, ok := retractionMonth(obj); ok && p.open(rm) && !slices.Contains(root.Retracted, id) {
		root.Retracted = append(root.Retracted, id)
	}
	sort.Strings(root.Retracted)

	// Segment counts: legacy may have grown; every listed path keeps its own.
	for rel, d := range s.Docs {
		if err := w.writeDoc(rel, d); err != nil {
			return err
		}
	}
	refs := map[string]int{}
	for _, seg := range root.Segments {
		refs[seg.Path] = seg.Count
	}
	for rel, d := range s.Docs {
		refs[rel] = len(d.Entries) + len(d.Unknown)
	}
	root.Segments = sortedRefs(refs)
	return w.writeDoc(IndexFile, root)
}

// sealRoot moves out of the root everything in a month that the step from p
// to next seals, into new segments in s.Docs. It returns nil (and nothing
// moved) when a segment for such a month already exists, or a retracted
// id's object cannot be read; the caller then rebuilds.
func (w *Workspace) sealRoot(s *ShardedIndex, p, next months) (map[string]*Index, error) {
	root := s.Root
	sealing := func(m int) bool { return p.open(m) && !next.open(m) }
	out := map[string]*Index{}
	seg := func(m int) *Index {
		rel := segmentPath(m)
		if d, ok := out[rel]; ok {
			return d
		}
		d := &Index{Format: root.Format, kind: kindSegment}
		out[rel] = d
		return d
	}
	var keep []Entry
	for _, e := range root.Entries {
		if m, ok := idMonth(e.ID); ok && sealing(m) {
			seg(m).Entries = append(seg(m).Entries, e)
			continue
		}
		keep = append(keep, e)
	}
	var keepUnknown []*yaml.Node
	for _, n := range root.Unknown {
		if m, ok := idMonth(nodeField(n, "id")); ok && sealing(m) {
			seg(m).Unknown = append(seg(m).Unknown, n)
			continue
		}
		keepUnknown = append(keepUnknown, n)
	}
	var keepRetracted []string
	for _, id := range root.Retracted {
		obj, err := w.Read(id)
		if err != nil {
			return nil, nil
		}
		if rm, ok := retractionMonth(obj); ok && sealing(rm) {
			seg(rm).Retracted = append(seg(rm).Retracted, id)
			continue
		}
		keepRetracted = append(keepRetracted, id)
	}
	for rel := range out {
		if _, err := os.Stat(filepath.Join(w.Root, filepath.FromSlash(rel))); err == nil {
			return nil, nil
		}
	}
	root.Entries, root.Unknown, root.Retracted = keep, keepUnknown, keepRetracted
	for rel, d := range out {
		d.sort()
		sort.Strings(d.Retracted)
		s.Docs[rel] = d
	}
	return out, nil
}

// sortedRefs orders segment references legacy first, then months ascending.
func sortedRefs(counts map[string]int) []SegmentRef {
	refs := make([]SegmentRef, 0, len(counts))
	for p, c := range counts {
		refs = append(refs, SegmentRef{Path: p, Count: c})
	}
	sort.Slice(refs, func(i, j int) bool {
		if (refs[i].Path == LegacyIndex) != (refs[j].Path == LegacyIndex) {
			return refs[i].Path == LegacyIndex
		}
		return refs[i].Path < refs[j].Path
	})
	return refs
}

// currentMonth is the newest month directory holding a file, across the type
// directories — the newest id's month, found without the index.
func (w *Workspace) currentMonth() months {
	var p months
	for _, t := range allTypes {
		dirs, err := os.ReadDir(w.Dir(t))
		if err != nil {
			continue
		}
		for _, d := range dirs {
			if !d.IsDir() || !monthDir.MatchString(d.Name()) {
				continue
			}
			tm, err := time.Parse("2006-01", d.Name())
			if err != nil || (p.has && monthNum(tm) <= p.cur) {
				continue
			}
			files, _ := os.ReadDir(filepath.Join(w.Dir(t), d.Name()))
			if slices.ContainsFunc(files, objectFile) {
				p.see(monthNum(tm))
			}
		}
	}
	return p
}

// sealedGuard refuses a write dated at t into a sealed month.
func (w *Workspace) sealedGuard(t time.Time, what string) error {
	p := w.currentMonth()
	if m := monthNum(t); !p.open(m) {
		return fmt.Errorf("%w: %s %s falls in %s, which is sealed (the workspace's newest id is from %s); check this machine's clock",
			ErrSealedMonth, what, t.UTC().Format(time.RFC3339), monthName(m), monthName(p.cur))
	}
	return nil
}

// checkSharded compares the committed dkf/0.2 index with the files: counts
// first for each sealed month, parsing a segment only when its count
// differs, then the root, the legacy document, and every differing segment.
func (w *Workspace) checkSharded() (IndexDiff, error) {
	g, err := w.Load()
	if err != nil {
		return IndexDiff{}, err
	}
	want := BuildSharded(g, w.Config.Format)
	d := IndexDiff{Missing: []string{}, Extra: []string{}, Changed: []string{}, LateMerge: []string{},
		SegmentsMissing: []string{}, SegmentsExtra: []string{}, RetractedMissing: []string{}, RetractedExtra: []string{}}
	root, rootBytes, err := w.readRoot()
	if err != nil {
		for _, doc := range append([]*Index{want.Root}, mapValues(want.Docs)...) {
			for _, e := range doc.Entries {
				d.Missing = append(d.Missing, e.ID)
			}
		}
		sort.Strings(d.Missing)
		d.BytesDiffer = true
		return d, nil
	}

	// Files per sealed month, from what Load read: no further I/O.
	files := map[string][]string{}
	for _, obj := range g.Objects() {
		if m, ok := idMonth(obj.ObjectID()); ok {
			rel := segmentPath(m)
			if _, sealed := want.Docs[rel]; sealed {
				files[rel] = append(files[rel], obj.ObjectID())
			}
		}
	}
	committedRefs := map[string]int{}
	for _, seg := range root.Segments {
		committedRefs[seg.Path] = seg.Count
	}
	var committed []*Index
	compare := []string{}
	for _, seg := range root.Segments {
		if _, ok := want.Docs[seg.Path]; !ok {
			d.SegmentsExtra = append(d.SegmentsExtra, seg.Path)
			continue
		}
		if seg.Path != LegacyIndex && seg.Count == len(files[seg.Path]) {
			if _, err := os.Stat(filepath.Join(w.Root, filepath.FromSlash(seg.Path))); err != nil {
				d.SegmentsMissing = append(d.SegmentsMissing, seg.Path)
				missingDoc(&d, want.Docs[seg.Path])
			}
			continue // a matching count: no file arrived; not parsed
		}
		compare = append(compare, seg.Path)
	}
	for rel := range want.Docs {
		if _, ok := committedRefs[rel]; !ok {
			d.SegmentsMissing = append(d.SegmentsMissing, rel)
			missingDoc(&d, want.Docs[rel])
		}
	}
	got := map[string]*Index{}
	lateIn := map[string]bool{}
	for _, rel := range compare {
		doc, _, err := w.readDoc(rel)
		if err != nil {
			d.SegmentsMissing = append(d.SegmentsMissing, rel)
			missingDoc(&d, want.Docs[rel])
			continue
		}
		got[rel] = doc
		committed = append(committed, doc)
		if rel != LegacyIndex {
			have := map[string]bool{}
			for _, e := range doc.Entries {
				have[e.ID] = true
			}
			for _, id := range files[rel] {
				if !have[id] {
					d.LateMerge = append(d.LateMerge, id)
					lateIn[rel] = true
				}
			}
		}
	}
	want.preserve(append([]*Index{root}, committed...))

	compareDocs(&d, want.Root, root)
	wantRoot := *want.Root
	maskLike(&wantRoot, root)
	// A late-merged segment is never rewritten, so its committed count is the
	// right one for the root to carry; the difference is the late merge, not
	// a stale root.
	wantRoot.Segments = slices.Clone(wantRoot.Segments)
	for i, seg := range wantRoot.Segments {
		if lateIn[seg.Path] {
			wantRoot.Segments[i].Count = committedRefs[seg.Path]
		}
	}
	wantBytes, err := EncodeIndex(&wantRoot)
	if err != nil {
		return IndexDiff{}, err
	}
	d.BytesDiffer = !bytes.Equal(wantBytes, rootBytes)
	for rel, doc := range got {
		compareDocs(&d, want.Docs[rel], doc)
	}
	for _, l := range [][]string{d.Missing, d.Extra, d.Changed, d.LateMerge, d.SegmentsMissing, d.SegmentsExtra, d.RetractedMissing, d.RetractedExtra} {
		sort.Strings(l)
	}
	return d, nil
}

// missingDoc records everything an absent document should hold, so a
// missing segment names its objects rather than only its path.
func missingDoc(d *IndexDiff, want *Index) {
	for _, e := range want.Entries {
		d.Missing = append(d.Missing, e.ID)
	}
	d.RetractedMissing = append(d.RetractedMissing, want.Retracted...)
}

// compareDocs records how a committed document differs from its rebuild:
// entries by the dkf/0.1 rules, and retracted as a set of ids.
func compareDocs(d *IndexDiff, want, got *Index) {
	gotBy := map[string]Entry{}
	for _, e := range got.Entries {
		gotBy[e.ID] = e
	}
	wantBy := map[string]bool{}
	for _, e := range want.Entries {
		wantBy[e.ID] = true
		ge, ok := gotBy[e.ID]
		if !ok {
			d.Missing = append(d.Missing, e.ID)
			continue
		}
		masked := maskEntry(e, ge)
		if !entriesEqual(ge, masked) {
			d.Changed = append(d.Changed, e.ID)
		}
	}
	for _, e := range got.Entries {
		if !wantBy[e.ID] {
			d.Extra = append(d.Extra, e.ID)
		}
	}
	for _, id := range want.Retracted {
		if !slices.Contains(got.Retracted, id) {
			d.RetractedMissing = append(d.RetractedMissing, id)
		}
	}
	for _, id := range got.Retracted {
		if !slices.Contains(want.Retracted, id) {
			d.RetractedExtra = append(d.RetractedExtra, id)
		}
	}
}

// maskLike applies maskEntry to every entry of want that got also carries.
func maskLike(want, got *Index) {
	gotBy := map[string]Entry{}
	for _, e := range got.Entries {
		gotBy[e.ID] = e
	}
	entries := make([]Entry, len(want.Entries))
	for i, e := range want.Entries {
		if ge, ok := gotBy[e.ID]; ok {
			e = maskEntry(e, ge)
		}
		entries[i] = e
	}
	want.Entries = entries
}

// SealedRetractionMismatch is a retraction a sealed segment and the files
// disagree on: an id in one month's regenerated list or committed list and
// not the other.
type SealedRetractionMismatch struct {
	ID      string
	Segment string
	// InFiles is true when the files date the retraction into the sealed
	// month but the committed segment lacks it.
	InFiles bool
}

// SealedRetractions regenerates every sealed month's retracted list from g
// and compares it with the committed segment's. This needs every segment
// parsed, which is validation's cost, not the drift check's.
func (w *Workspace) SealedRetractions(g *Graph) ([]SealedRetractionMismatch, error) {
	if !w.Sharded() {
		return nil, nil
	}
	want := BuildSharded(g, w.Config.Format)
	var out []SealedRetractionMismatch
	for rel, wd := range want.Docs {
		if rel == LegacyIndex {
			continue
		}
		cd, _, err := w.readDoc(rel)
		if err != nil {
			continue // a missing segment is the drift check's finding
		}
		for _, id := range wd.Retracted {
			if !slices.Contains(cd.Retracted, id) {
				out = append(out, SealedRetractionMismatch{ID: id, Segment: rel, InFiles: true})
			}
		}
		for _, id := range cd.Retracted {
			if !slices.Contains(wd.Retracted, id) {
				out = append(out, SealedRetractionMismatch{ID: id, Segment: rel})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Segment != out[j].Segment {
			return out[i].Segment < out[j].Segment
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
