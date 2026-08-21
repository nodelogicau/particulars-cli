package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// FileProblem records a file that could not be loaded as a valid object.
type FileProblem struct {
	Path    string // relative to workspace root
	Code    string // parse_error | id_mismatch | type_mismatch
	Message string
}

func (p FileProblem) Error() string { return fmt.Sprintf("%s: %s", p.Path, p.Message) }

// Graph is the in-memory view of a workspace.
type Graph struct {
	Particulars map[string]*dkf.Particular // by id
	Assertions  map[string]dkf.Assertion   // claims and syntheses by id
	Merges      map[string]*dkf.Merge      // merge records by id
	BySubject   map[string][]dkf.Assertion // subject id -> assertions, sorted by id
	Files       map[string]string          // object id -> relative path
	Raw         map[string][]byte          // object id -> file bytes (for canonical checks)
	Problems    []FileProblem              // files skipped while loading

	byURI   map[string]*dkf.Particular
	classOf map[string][]string // particular id -> sorted member ids (incl. self)
}

func newGraph() *Graph {
	return &Graph{
		Particulars: map[string]*dkf.Particular{},
		Assertions:  map[string]dkf.Assertion{},
		Merges:      map[string]*dkf.Merge{},
		BySubject:   map[string][]dkf.Assertion{},
		Files:       map[string]string{},
		Raw:         map[string][]byte{},
		byURI:       map[string]*dkf.Particular{},
	}
}

// Load reads every object file. It returns an error only for IO failures; files
// that fail to parse or are misplaced are recorded in Graph.Problems so that
// `validate` can report them. Callers that need a clean graph should check
// Problems (see Graph.Err).
func (w *Workspace) Load() (*Graph, error) {
	g := newGraph()
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
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			rel := w.Rel(path)
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			obj, err := dkf.Decode(data)
			if err != nil {
				g.Problems = append(g.Problems, FileProblem{Path: rel, Code: "parse_error", Message: err.Error()})
				continue
			}
			fileID := strings.TrimSuffix(e.Name(), ".yaml")
			if obj.ObjectID() != fileID {
				g.Problems = append(g.Problems, FileProblem{Path: rel, Code: "id_mismatch", Message: fmt.Sprintf("file name implies %s but id field is %q", fileID, obj.ObjectID())})
				continue
			}
			if obj.ObjectType() != t {
				g.Problems = append(g.Problems, FileProblem{Path: rel, Code: "type_mismatch", Message: fmt.Sprintf("file is in %s/ but type is %s", dirFor(t), obj.ObjectType())})
				continue
			}
			if idType, err := dkf.TypeOfID(obj.ObjectID()); err != nil || idType != t {
				g.Problems = append(g.Problems, FileProblem{Path: rel, Code: "type_mismatch", Message: fmt.Sprintf("id prefix does not match type %s", t)})
				continue
			}
			g.Files[obj.ObjectID()] = rel
			g.Raw[obj.ObjectID()] = data
			g.add(obj)
		}
	}
	for subj := range g.BySubject {
		sortAssertions(g.BySubject[subj])
	}
	g.buildClasses()
	return g, nil
}

// buildClasses unions the URIs of every non-retracted merge and derives, for
// each particular, the sorted ids of all particulars in its class. URIs with
// no local particular still act as bridges.
func (g *Graph) buildClasses() {
	parent := map[string]string{}
	var find func(string) string
	find = func(u string) string {
		p, ok := parent[u]
		if !ok {
			parent[u] = u
			return u
		}
		if p != u {
			parent[u] = find(p)
		}
		return parent[u]
	}
	for _, m := range g.Merges {
		if m.Retracted != nil || len(m.URIs) != 2 {
			continue
		}
		a, b := find(m.URIs[0]), find(m.URIs[1])
		if a != b {
			parent[a] = b
		}
	}
	members := map[string][]string{} // root uri -> particular ids
	for _, p := range g.Particulars {
		root := find(p.URI)
		members[root] = append(members[root], p.ID)
	}
	g.classOf = map[string][]string{}
	for _, ids := range members {
		sort.Strings(ids)
		for _, id := range ids {
			g.classOf[id] = ids
		}
	}
}

// ClassOf returns the sorted ids of every particular in id's merge
// equivalence class, including id itself. Unknown ids yield a singleton.
func (g *Graph) ClassOf(id string) []string {
	if c, ok := g.classOf[id]; ok {
		return c
	}
	return []string{id}
}

// ClassAssertions returns the assertions whose subject is any member of the
// class, sorted by id.
func (g *Graph) ClassAssertions(members []string) []dkf.Assertion {
	var out []dkf.Assertion
	for _, m := range members {
		out = append(out, g.BySubject[m]...)
	}
	sortAssertions(out)
	return out
}

// MergeBetween returns the non-retracted merge joining exactly these two
// URIs (in either order), or nil.
func (g *Graph) MergeBetween(a, b string) *dkf.Merge {
	if a > b {
		a, b = b, a
	}
	var found *dkf.Merge
	for _, m := range g.Merges {
		if m.Retracted != nil || len(m.URIs) != 2 {
			continue
		}
		x, y := m.URIs[0], m.URIs[1]
		if x > y {
			x, y = y, x
		}
		if x == a && y == b && (found == nil || m.ID < found.ID) {
			found = m
		}
	}
	return found
}

// SortedMerges returns all merge records ordered by id.
func (g *Graph) SortedMerges() []*dkf.Merge {
	out := make([]*dkf.Merge, 0, len(g.Merges))
	for _, m := range g.Merges {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (g *Graph) add(obj dkf.Object) {
	switch o := obj.(type) {
	case *dkf.Particular:
		g.Particulars[o.ID] = o
		if _, dup := g.byURI[o.URI]; !dup {
			g.byURI[o.URI] = o
		}
	case dkf.Assertion:
		g.Assertions[o.ObjectID()] = o
		g.BySubject[o.SubjectID()] = append(g.BySubject[o.SubjectID()], o)
	case *dkf.Merge:
		g.Merges[o.ID] = o
	}
}

// Err returns a single error summarising load problems, or nil.
func (g *Graph) Err() error {
	if len(g.Problems) == 0 {
		return nil
	}
	msgs := make([]string, 0, len(g.Problems))
	for _, p := range g.Problems {
		msgs = append(msgs, p.Error())
	}
	return fmt.Errorf("%d unreadable object file(s); run `particulars validate`: %s", len(g.Problems), strings.Join(msgs, "; "))
}

// ParticularByURI returns the particular with the given URI, or nil.
func (g *Graph) ParticularByURI(uri string) *dkf.Particular { return g.byURI[uri] }

// Particular returns a particular by id, or nil.
func (g *Graph) Particular(id string) *dkf.Particular { return g.Particulars[id] }

// Assertion returns a claim or synthesis by id, or nil.
func (g *Graph) Assertion(id string) dkf.Assertion { return g.Assertions[id] }

// SortedParticulars returns all particulars ordered by id.
func (g *Graph) SortedParticulars() []*dkf.Particular {
	out := make([]*dkf.Particular, 0, len(g.Particulars))
	for _, p := range g.Particulars {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// SortedAssertions returns all claims and syntheses ordered by id.
func (g *Graph) SortedAssertions() []dkf.Assertion {
	out := make([]dkf.Assertion, 0, len(g.Assertions))
	for _, a := range g.Assertions {
		out = append(out, a)
	}
	sortAssertions(out)
	return out
}

// Objects returns every object and record ordered by id.
func (g *Graph) Objects() []dkf.Object {
	out := make([]dkf.Object, 0, len(g.Particulars)+len(g.Assertions)+len(g.Merges))
	for _, p := range g.SortedParticulars() {
		out = append(out, p)
	}
	for _, a := range g.SortedAssertions() {
		out = append(out, a)
	}
	for _, m := range g.SortedMerges() {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ObjectID() < out[j].ObjectID() })
	return out
}

func sortAssertions(as []dkf.Assertion) {
	sort.Slice(as, func(i, j int) bool { return as[i].ObjectID() < as[j].ObjectID() })
}
