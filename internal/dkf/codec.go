package dkf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// TimeLayout is the on-disk timestamp format: RFC 3339, UTC, seconds precision.
const TimeLayout = "2006-01-02T15:04:05Z"

// FormatTime renders t in the canonical on-disk form.
func FormatTime(t time.Time) string { return t.UTC().Format(TimeLayout) }

// ParseTime accepts RFC 3339 with or without fractional seconds or offsets.
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: must be RFC 3339", s)
	}
	return t.UTC(), nil
}

// Encode serialises any DKF object deterministically: 2-space indent, spec key
// order, literal block scalars for multi-line strings, RFC 3339 Z timestamps,
// optional fields omitted when unset, no document markers.
func Encode(obj Object) ([]byte, error) {
	var root *yaml.Node
	switch o := obj.(type) {
	case *Particular:
		root = particularNode(o)
	case *Claim:
		root = claimNode(o)
	case *Synthesis:
		root = synthesisNode(o)
	case *Merge:
		root = mergeNode(o)
	case *Promotion:
		root = promotionNode(o)
	default:
		return nil, fmt.Errorf("dkf: cannot encode %T", obj)
	}
	return marshalNode(root)
}

// EncodeRetracted serialises a retraction block as a top-level `retracted:`
// mapping, suitable for appending to an existing object file.
func EncodeRetracted(r *Retracted) ([]byte, error) {
	root := mapping()
	addKV(root, "retracted", retractedNode(r))
	return marshalNode(root)
}

func marshalNode(root *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// --- node builders -------------------------------------------------------

func mapping() *yaml.Node { return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"} }

func sequence() *yaml.Node { return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"} }

func scalar(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	if strings.Contains(s, "\n") {
		n.Style = yaml.LiteralStyle
	}
	return n
}

func plain(tag, s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: s}
}

func addKV(m *yaml.Node, key string, v *yaml.Node) {
	if v == nil {
		return
	}
	m.Content = append(m.Content, plain("!!str", key), v)
}

func addStr(m *yaml.Node, key, v string) {
	if v == "" {
		return
	}
	addKV(m, key, scalar(v))
}

// addStrAlways writes the key even when empty, for required fields, so that a
// malformed object round-trips visibly rather than silently dropping the key.
func addStrAlways(m *yaml.Node, key, v string) { addKV(m, key, scalar(v)) }

func addStrList(m *yaml.Node, key string, vs []string) {
	if len(vs) == 0 {
		return
	}
	seq := sequence()
	for _, v := range vs {
		seq.Content = append(seq.Content, scalar(v))
	}
	addKV(m, key, seq)
}

func addTime(m *yaml.Node, key string, t time.Time) {
	if t.IsZero() {
		return
	}
	addKV(m, key, plain("!!timestamp", FormatTime(t)))
}

func addFloat(m *yaml.Node, key string, f *float64) {
	if f == nil {
		return
	}
	addKV(m, key, plain("!!float", strconv.FormatFloat(*f, 'g', -1, 64)))
}

func sourceNode(s Source) *yaml.Node {
	m := mapping()
	addStr(m, "author", s.Author)
	addStr(m, "harness", s.Harness)
	addStr(m, "model", s.Model)
	addStr(m, "document", s.Document)
	return m
}

func contextNode(c Context) *yaml.Node {
	m := mapping()
	addStr(m, "scope", string(c.Scope))
	addStrList(m, "topics", c.Topics)
	return m
}

func retractedNode(r *Retracted) *yaml.Node {
	m := mapping()
	addTime(m, "timestamp", r.Timestamp)
	addStrAlways(m, "reason", r.Reason)
	addKV(m, "source", sourceNode(r.Source))
	addStr(m, "superseded-by", r.SupersededBy)
	return m
}

func particularNode(p *Particular) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", p.ID)
	addStrAlways(m, "type", string(TypeParticular))
	addStrAlways(m, "uri", p.URI)
	addStrAlways(m, "label", p.Label)
	addStrList(m, "aliases", p.Aliases)
	return m
}

func claimNode(c *Claim) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", c.ID)
	addStrAlways(m, "type", string(TypeClaim))
	addStrAlways(m, "subject", c.Subject)
	addStrAlways(m, "content", c.Content)
	addKV(m, "source", sourceNode(c.Source))
	addKV(m, "context", contextNode(c.Context))
	addTime(m, "timestamp", c.Timestamp)
	addFloat(m, "confidence", c.Confidence)
	if c.Retracted != nil {
		addKV(m, "retracted", retractedNode(c.Retracted))
	}
	return m
}

func synthesisNode(s *Synthesis) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", s.ID)
	addStrAlways(m, "type", string(TypeSynthesis))
	addStrAlways(m, "subject", s.Subject)
	addStrAlways(m, "content", s.Content)
	inputs := sequence()
	for _, in := range s.Inputs {
		im := mapping()
		addStrAlways(im, "id", in.ID)
		addStrAlways(im, "role", string(in.Role))
		addStr(im, "weight", string(in.Weight))
		inputs.Content = append(inputs.Content, im)
	}
	addKV(m, "inputs", inputs)
	addStrAlways(m, "unresolved", s.Unresolved)
	addKV(m, "source", sourceNode(s.Source))
	addStr(m, "method", s.Method)
	addTime(m, "timestamp", s.Timestamp)
	addKV(m, "context", contextNode(s.Context))
	addFloat(m, "confidence", s.Confidence)
	if s.Retracted != nil {
		addKV(m, "retracted", retractedNode(s.Retracted))
	}
	return m
}

func promotionNode(pr *Promotion) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", pr.ID)
	addStrAlways(m, "type", string(TypePublish))
	claims := sequence()
	for _, c := range pr.Claims {
		claims.Content = append(claims.Content, scalar(c))
	}
	addKV(m, "claims", claims)
	addStrAlways(m, "scope", string(pr.Scope))
	addStr(m, "reason", pr.Reason)
	addKV(m, "source", sourceNode(pr.Source))
	addTime(m, "timestamp", pr.Timestamp)
	if pr.Retracted != nil {
		addKV(m, "retracted", retractedNode(pr.Retracted))
	}
	return m
}

func mergeNode(mg *Merge) *yaml.Node {
	m := mapping()
	addStrAlways(m, "id", mg.ID)
	addStrAlways(m, "type", string(TypeMerge))
	uris := sequence()
	for _, u := range mg.URIs {
		uris.Content = append(uris.Content, scalar(u))
	}
	addKV(m, "uris", uris)
	addStr(m, "reason", mg.Reason)
	addKV(m, "source", sourceNode(mg.Source))
	addTime(m, "timestamp", mg.Timestamp)
	if mg.Retracted != nil {
		addKV(m, "retracted", retractedNode(mg.Retracted))
	}
	return m
}

// --- decoding ------------------------------------------------------------

// synthesisFile mirrors Synthesis but keeps provenance blocks as pointers so
// the decoder can tell which of source / produced-by were present.
type synthesisFile struct {
	ID         string      `yaml:"id"`
	Subject    string      `yaml:"subject"`
	Content    string      `yaml:"content"`
	Inputs     []Input     `yaml:"inputs"`
	Unresolved string      `yaml:"unresolved"`
	Source     *Source     `yaml:"source"`
	ProducedBy *ProducedBy `yaml:"produced-by"`
	Method     string      `yaml:"method"`
	Timestamp  time.Time   `yaml:"timestamp"`
	Context    Context     `yaml:"context"`
	Confidence *float64    `yaml:"confidence"`
	Retracted  *Retracted  `yaml:"retracted"`
}

func (f *synthesisFile) toSynthesis() *Synthesis {
	s := &Synthesis{
		ID: f.ID, Subject: f.Subject, Content: f.Content, Inputs: f.Inputs, Unresolved: f.Unresolved,
		Method: f.Method, Timestamp: f.Timestamp, Context: f.Context, Confidence: f.Confidence, Retracted: f.Retracted,
	}
	switch {
	case f.Source != nil && f.ProducedBy != nil:
		s.Source = *f.Source
		s.ConflictingProvenance = true
	case f.Source != nil:
		s.Source = *f.Source
	case f.ProducedBy != nil:
		s.Source = Source{Harness: f.ProducedBy.Harness, Model: f.ProducedBy.Model}
		s.LegacyProducedBy = true
	}
	return s
}

// typeProbe reads only the discriminator.
type typeProbe struct {
	Type Type `yaml:"type"`
}

// Decode parses a single DKF object. Unknown keys are ignored; identifiers are
// accepted under the lenient grammar. The `type` field selects the Go type.
func Decode(data []byte) (Object, error) {
	var probe typeProbe
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	switch probe.Type {
	case TypeParticular:
		var p Particular
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parse particular: %w", err)
		}
		return &p, nil
	case TypeClaim:
		var c Claim
		if err := yaml.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("parse claim: %w", err)
		}
		return &c, nil
	case TypeSynthesis:
		var f synthesisFile
		if err := yaml.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("parse synthesis: %w", err)
		}
		return f.toSynthesis(), nil
	case TypeMerge:
		var m Merge
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parse merge: %w", err)
		}
		return &m, nil
	case TypePublish:
		var pr Promotion
		if err := yaml.Unmarshal(data, &pr); err != nil {
			return nil, fmt.Errorf("parse promotion: %w", err)
		}
		return &pr, nil
	case "":
		return nil, fmt.Errorf("missing type field")
	default:
		return nil, fmt.Errorf("unknown type %q", probe.Type)
	}
}

// IsCanonical reports whether data is byte-identical to re-encoding the object
// it parses to. Callers use it for the `non_canonical` validation warning.
func IsCanonical(data []byte) (bool, error) {
	obj, err := Decode(data)
	if err != nil {
		return false, err
	}
	out, err := Encode(obj)
	if err != nil {
		return false, err
	}
	return bytes.Equal(out, data), nil
}
