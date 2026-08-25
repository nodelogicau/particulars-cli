// Package dkf implements the Dialectical Knowledge Format (DKF) v0.1 object
// model: identifiers, YAML codec, and field-level validation. It has no
// filesystem access; callers (internal/store) own IO.
//
// Field order in every struct below is the spec's serialisation order and is
// what the encoder emits. Keep the two in sync.
package dkf

import "time"

// FormatVersion is the DKF format identifier written to dkf.yaml and index.yaml.
const FormatVersion = "dkf/0.1"

// Type is a DKF object type.
type Type string

const (
	TypeParticular Type = "particular"
	TypeClaim      Type = "claim"
	TypeSynthesis  Type = "synthesis"
	TypeMerge      Type = "merge"
	TypePublish    Type = "publish"
)

// Scope is a claim's visibility scope.
type Scope string

const (
	ScopePersonal     Scope = "personal"
	ScopeOrganisation Scope = "organisation"
	ScopePublic       Scope = "public"
)

// Role is the dialectical role a synthesis input plays.
type Role string

const (
	RoleThesis     Role = "thesis"
	RoleAntithesis Role = "antithesis"
)

// Weight is the influence of a synthesis input.
type Weight string

const (
	WeightPrimary    Weight = "primary"
	WeightQualifying Weight = "qualifying"
)

// Source records who or what asserted a claim.
// Spec order: author, harness, model, document.
type Source struct {
	Author   string   `yaml:"author,omitempty" json:"author,omitempty"`
	Harness  string   `yaml:"harness,omitempty" json:"harness,omitempty"`
	Model    string   `yaml:"model,omitempty" json:"model,omitempty"`
	Document Document `yaml:"document,omitempty" json:"document,omitempty"`
}

// IsZero reports whether no source field is set.
func (s Source) IsZero() bool {
	return s.Author == "" && s.Harness == "" && s.Model == "" && s.Document.IsZero()
}

// Context carries scope and topics.
// Spec order: scope, topics.
type Context struct {
	Scope  Scope    `yaml:"scope,omitempty" json:"scope,omitempty"`
	Topics []string `yaml:"topics,omitempty" json:"topics,omitempty"`
}

// Input is one thesis/antithesis input to a synthesis.
// Spec order: id, role, weight.
type Input struct {
	ID     string `yaml:"id" json:"id"`
	Role   Role   `yaml:"role" json:"role"`
	Weight Weight `yaml:"weight,omitempty" json:"weight,omitempty"`
}

// ProducedBy is the pre-resolution provenance block that v0.1.x wrote on
// syntheses. It is read (and mapped to Source) but never written.
type ProducedBy struct {
	Harness string `yaml:"harness,omitempty"`
	Model   string `yaml:"model,omitempty"`
}

// Retracted is the append-only retraction block on a claim or synthesis.
// Order: timestamp, reason, source, superseded-by.
type Retracted struct {
	Timestamp time.Time `yaml:"timestamp" json:"timestamp"`
	Reason    string    `yaml:"reason" json:"reason"`
	Source    Source    `yaml:"source" json:"source"`
	// Kind records why the claim died, at one of the three joints in
	// claim → source → world. Declared, never inferred from SupersededBy:
	// that field answers whether anything replaced the claim, which is a
	// different question — a typo-grade defect routinely carries a
	// replacement, and a claim retracted because its subject was
	// decommissioned is a supersession with nothing to point at.
	Kind         RetractionKind `yaml:"kind,omitempty" json:"kind,omitempty"`
	SupersededBy string         `yaml:"superseded-by,omitempty" json:"superseded-by,omitempty"`
}

// RetractionKind is why a claim was retracted.
type RetractionKind string

const (
	// KindDefect: the claim misread what its source said.
	KindDefect RetractionKind = "defect"
	// KindSupersession: the source was right then, and the world moved on.
	KindSupersession RetractionKind = "supersession"
	// KindProvenanceFailure: the source itself was wrong.
	KindProvenanceFailure RetractionKind = "provenance-failure"
)

// ValidRetractionKind reports whether k is one of the three joints.
func ValidRetractionKind(k RetractionKind) bool {
	return k == KindDefect || k == KindSupersession || k == KindProvenanceFailure
}

// Particular is a DPARTICULAR: a specific, identifiable thing claims are about.
// Spec order: id, type, uri, label, aliases.
type Particular struct {
	ID      string   `yaml:"id" json:"id"`
	URI     string   `yaml:"uri" json:"uri"`
	Label   string   `yaml:"label" json:"label"`
	Aliases []string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
}

// Claim is a DCLAIM: an assertion about a particular with provenance.
// Spec order: id, type, subject, content, source, context, timestamp,
// confidence, retracted.
type Claim struct {
	ID         string     `yaml:"id" json:"id"`
	Subject    string     `yaml:"subject" json:"subject"`
	Content    string     `yaml:"content" json:"content"`
	Source     Source     `yaml:"source" json:"source"`
	Context    Context    `yaml:"context" json:"context"`
	Timestamp  time.Time  `yaml:"timestamp" json:"timestamp"`
	Confidence *float64   `yaml:"confidence,omitempty" json:"confidence,omitempty"`
	Retracted  *Retracted `yaml:"retracted,omitempty" json:"retracted,omitempty"`
}

// Synthesis is a DSYNTHESIS: a claim derived from thesis/antithesis inputs.
// Spec order: id, type, subject, content, inputs, unresolved, source,
// method, timestamp, context, confidence, retracted.
type Synthesis struct {
	ID         string     `yaml:"id" json:"id"`
	Subject    string     `yaml:"subject" json:"subject"`
	Content    string     `yaml:"content" json:"content"`
	Inputs     []Input    `yaml:"inputs" json:"inputs"`
	Unresolved string     `yaml:"unresolved" json:"unresolved"`
	Source     Source     `yaml:"source" json:"source"`
	Method     string     `yaml:"method,omitempty" json:"method,omitempty"`
	Timestamp  time.Time  `yaml:"timestamp" json:"timestamp"`
	Context    Context    `yaml:"context" json:"context"`
	Confidence *float64   `yaml:"confidence,omitempty" json:"confidence,omitempty"`
	Retracted  *Retracted `yaml:"retracted,omitempty" json:"retracted,omitempty"`

	// LegacyProducedBy is set by the decoder when Source was populated from a
	// v0.1.x `produced-by` block. Never serialised.
	LegacyProducedBy bool `yaml:"-" json:"-"`
	// ConflictingProvenance is set when a file carried both `source` and
	// `produced-by`; `source` wins and validate reports it. Never serialised.
	ConflictingProvenance bool `yaml:"-" json:"-"`
}

// Merge is a merge record: a declaration that two URIs denote the same
// particular. It is a record about objects, not a knowledge object.
// Order: id, type, uris, reason, source, timestamp, retracted.
type Merge struct {
	ID        string     `yaml:"id" json:"id"`
	URIs      []string   `yaml:"uris" json:"uris"`
	Reason    string     `yaml:"reason,omitempty" json:"reason,omitempty"`
	Source    Source     `yaml:"source" json:"source"`
	Timestamp time.Time  `yaml:"timestamp" json:"timestamp"`
	Retracted *Retracted `yaml:"retracted,omitempty" json:"retracted,omitempty"`
}

// ScopeRank orders visibility from narrowest to widest: personal <
// organisation < public. An unrecognised scope ranks narrowest, so a value we
// do not understand is never treated as more widely shareable than it is.
func ScopeRank(s Scope) int {
	switch s {
	case ScopeOrganisation:
		return 1
	case ScopePublic:
		return 2
	}
	return 0
}

// ScopeAtLeast reports whether s is at least as wide as min.
func ScopeAtLeast(s, min Scope) bool { return ScopeRank(s) >= ScopeRank(min) }

// Promotion records a decision to share the named claims and syntheses more
// widely. Claims are immutable, so scope cannot be rewritten in place; an
// object's effective scope is its asserted scope widened by the promotions
// covering it. Promotion may only widen, and never cascades to a synthesis's
// inputs.
type Promotion struct {
	ID        string     `yaml:"id" json:"id"`
	Claims    []string   `yaml:"claims" json:"claims"`
	Scope     Scope      `yaml:"scope" json:"scope"`
	Reason    string     `yaml:"reason,omitempty" json:"reason,omitempty"`
	Source    Source     `yaml:"source" json:"source"`
	Timestamp time.Time  `yaml:"timestamp" json:"timestamp"`
	Retracted *Retracted `yaml:"retracted,omitempty" json:"retracted,omitempty"`
}

// DefaultMethod is the synthesis method recorded when none is given.
const DefaultMethod = "reconciliation"

// Object is any DKF object or record.
type Object interface {
	ObjectID() string
	ObjectType() Type
}

// Retractable is anything that can carry a retracted block: claims,
// syntheses, and merge records.
type Retractable interface {
	Object
	GetRetracted() *Retracted
	// SetRetracted attaches a retraction block (in memory only).
	SetRetracted(r *Retracted)
}

// Assertion is the behaviour shared by claims and syntheses: both are claims
// in the spec's sense and both can be retracted, recalled, and cited.
type Assertion interface {
	Retractable
	SubjectID() string
	GetContent() string
	GetContext() Context
	GetSource() Source
	GetTimestamp() time.Time
	GetConfidence() *float64
	// InputIDs returns the ids of cited inputs; empty for plain claims.
	InputIDs() []string
}

func (p *Particular) ObjectID() string { return p.ID }
func (p *Particular) ObjectType() Type { return TypeParticular }

func (c *Claim) ObjectID() string          { return c.ID }
func (c *Claim) ObjectType() Type          { return TypeClaim }
func (c *Claim) SubjectID() string         { return c.Subject }
func (c *Claim) GetContent() string        { return c.Content }
func (c *Claim) GetContext() Context       { return c.Context }
func (c *Claim) GetSource() Source         { return c.Source }
func (c *Claim) GetTimestamp() time.Time   { return c.Timestamp }
func (c *Claim) GetConfidence() *float64   { return c.Confidence }
func (c *Claim) GetRetracted() *Retracted  { return c.Retracted }
func (c *Claim) InputIDs() []string        { return nil }
func (c *Claim) SetRetracted(r *Retracted) { c.Retracted = r }

func (s *Synthesis) ObjectID() string          { return s.ID }
func (s *Synthesis) ObjectType() Type          { return TypeSynthesis }
func (s *Synthesis) SubjectID() string         { return s.Subject }
func (s *Synthesis) GetContent() string        { return s.Content }
func (s *Synthesis) GetContext() Context       { return s.Context }
func (s *Synthesis) GetSource() Source         { return s.Source }
func (s *Synthesis) GetTimestamp() time.Time   { return s.Timestamp }
func (s *Synthesis) GetConfidence() *float64   { return s.Confidence }
func (s *Synthesis) GetRetracted() *Retracted  { return s.Retracted }
func (s *Synthesis) SetRetracted(r *Retracted) { s.Retracted = r }

func (m *Merge) ObjectID() string               { return m.ID }
func (pr *Promotion) ObjectID() string          { return pr.ID }
func (pr *Promotion) ObjectType() Type          { return TypePublish }
func (pr *Promotion) GetRetracted() *Retracted  { return pr.Retracted }
func (pr *Promotion) SetRetracted(r *Retracted) { pr.Retracted = r }

func (m *Merge) ObjectType() Type          { return TypeMerge }
func (m *Merge) GetRetracted() *Retracted  { return m.Retracted }
func (m *Merge) SetRetracted(r *Retracted) { m.Retracted = r }

// InputIDs returns the ids of all inputs in declaration order.
func (s *Synthesis) InputIDs() []string {
	ids := make([]string, 0, len(s.Inputs))
	for _, in := range s.Inputs {
		ids = append(ids, in.ID)
	}
	return ids
}

// ValidScope reports whether s is one of the spec's scope values.
func ValidScope(s Scope) bool {
	switch s {
	case ScopePersonal, ScopeOrganisation, ScopePublic:
		return true
	}
	return false
}

// ValidRole reports whether r is a spec role.
func ValidRole(r Role) bool { return r == RoleThesis || r == RoleAntithesis }

// ValidWeight reports whether w is a spec weight.
func ValidWeight(w Weight) bool { return w == WeightPrimary || w == WeightQualifying }
