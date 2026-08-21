package dkf

import (
	"fmt"
	"strings"
)

// Problem codes shared by write paths and `validate`.
const (
	CodeMissingField     = "missing_field"
	CodeInvalidEnum      = "invalid_enum"
	CodeOutOfRange       = "out_of_range"
	CodeInvalidTimestamp = "invalid_timestamp"
	CodeInvalidID        = "invalid_id"
	// CodeConflictingProvenance: a synthesis file carried both source and produced-by.
	CodeConflictingProvenance = "conflicting_provenance"
	// CodeInvalidMerge: a merge record's uris are malformed.
	CodeInvalidMerge = "invalid_merge"
)

// Problem is one field-level defect in an object.
type Problem struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func (p Problem) Error() string {
	if p.Field != "" {
		return fmt.Sprintf("%s: %s", p.Field, p.Message)
	}
	return p.Message
}

// Problems is a list of problems that also satisfies error.
type Problems []Problem

func (ps Problems) Error() string {
	msgs := make([]string, len(ps))
	for i, p := range ps {
		msgs[i] = p.Error()
	}
	return strings.Join(msgs, "; ")
}

// ErrOrNil returns ps as an error when non-empty, else nil.
func (ps Problems) ErrOrNil() error {
	if len(ps) == 0 {
		return nil
	}
	return ps
}

func missing(field string) Problem {
	return Problem{Code: CodeMissingField, Field: field, Message: "required field is missing or empty"}
}

// ValidateObject dispatches to the type-specific validator.
func ValidateObject(obj Object) Problems {
	switch o := obj.(type) {
	case *Particular:
		return ValidateParticular(o)
	case *Claim:
		return ValidateClaim(o)
	case *Synthesis:
		return ValidateSynthesis(o)
	case *Merge:
		return ValidateMerge(o)
	}
	return Problems{{Code: CodeInvalidEnum, Field: "type", Message: fmt.Sprintf("unsupported object %T", obj)}}
}

// ValidateParticular checks field-level constraints of a particular.
func ValidateParticular(p *Particular) Problems {
	var ps Problems
	ps = append(ps, checkID(p.ID, TypeParticular)...)
	if strings.TrimSpace(p.URI) == "" {
		ps = append(ps, missing("uri"))
	}
	if strings.TrimSpace(p.Label) == "" {
		ps = append(ps, missing("label"))
	}
	for i, a := range p.Aliases {
		if strings.TrimSpace(a) == "" {
			ps = append(ps, Problem{Code: CodeMissingField, Field: fmt.Sprintf("aliases[%d]", i), Message: "alias must be non-empty"})
		}
	}
	return ps
}

// ValidateClaim checks field-level constraints of a claim.
func ValidateClaim(c *Claim) Problems {
	var ps Problems
	ps = append(ps, checkID(c.ID, TypeClaim)...)
	ps = append(ps, checkSubject(c.Subject)...)
	if strings.TrimSpace(c.Content) == "" {
		ps = append(ps, missing("content"))
	}
	ps = append(ps, checkSource("source", c.Source)...)
	ps = append(ps, checkContext(c.Context)...)
	if c.Timestamp.IsZero() {
		ps = append(ps, missing("timestamp"))
	}
	ps = append(ps, checkConfidence(c.Confidence)...)
	if c.Retracted != nil {
		ps = append(ps, ValidateRetracted(c.Retracted)...)
	}
	return ps
}

// ValidateSynthesis checks field-level constraints of a synthesis.
func ValidateSynthesis(s *Synthesis) Problems {
	var ps Problems
	ps = append(ps, checkID(s.ID, TypeSynthesis)...)
	ps = append(ps, checkSubject(s.Subject)...)
	if strings.TrimSpace(s.Content) == "" {
		ps = append(ps, missing("content"))
	}
	if len(s.Inputs) == 0 {
		ps = append(ps, Problem{Code: CodeMissingField, Field: "inputs", Message: "at least one input is required"})
	}
	for i, in := range s.Inputs {
		f := fmt.Sprintf("inputs[%d]", i)
		if !IsValidID(in.ID) {
			ps = append(ps, Problem{Code: CodeInvalidID, Field: f + ".id", Message: fmt.Sprintf("invalid identifier %q", in.ID)})
		} else if !IsAssertionID(in.ID) {
			ps = append(ps, Problem{Code: CodeInvalidID, Field: f + ".id", Message: "input must reference a claim or synthesis, not a particular"})
		}
		if !ValidRole(in.Role) {
			ps = append(ps, Problem{Code: CodeInvalidEnum, Field: f + ".role", Message: fmt.Sprintf("role must be thesis or antithesis, got %q", in.Role)})
		}
		if in.Weight != "" && !ValidWeight(in.Weight) {
			ps = append(ps, Problem{Code: CodeInvalidEnum, Field: f + ".weight", Message: fmt.Sprintf("weight must be primary or qualifying, got %q", in.Weight)})
		}
	}
	if strings.TrimSpace(s.Unresolved) == "" {
		ps = append(ps, Problem{Code: CodeMissingField, Field: "unresolved", Message: "unresolved is required; state what could not be reconciled, or that nothing was identified"})
	}
	ps = append(ps, checkSource("source", s.Source)...)
	if strings.TrimSpace(s.Source.Harness) == "" {
		ps = append(ps, Problem{Code: CodeMissingField, Field: "source.harness", Message: "a synthesis must record the harness that produced it"})
	}
	if s.ConflictingProvenance {
		ps = append(ps, Problem{Code: CodeConflictingProvenance, Field: "produced-by", Message: "file carries both source and the legacy produced-by block; remove one"})
	}
	if s.Timestamp.IsZero() {
		ps = append(ps, missing("timestamp"))
	}
	ps = append(ps, checkContext(s.Context)...)
	ps = append(ps, checkConfidence(s.Confidence)...)
	if s.Retracted != nil {
		ps = append(ps, ValidateRetracted(s.Retracted)...)
	}
	return ps
}

// ValidateRetracted checks a retraction block.
func ValidateRetracted(r *Retracted) Problems {
	var ps Problems
	if r.Timestamp.IsZero() {
		ps = append(ps, missing("retracted.timestamp"))
	}
	if strings.TrimSpace(r.Reason) == "" {
		ps = append(ps, missing("retracted.reason"))
	}
	ps = append(ps, checkSource("retracted.source", r.Source)...)
	if r.SupersededBy != "" && !IsAssertionID(r.SupersededBy) {
		ps = append(ps, Problem{Code: CodeInvalidID, Field: "retracted.superseded-by", Message: fmt.Sprintf("must reference a claim or synthesis, got %q", r.SupersededBy)})
	}
	return ps
}

func checkID(id string, want Type) Problems {
	if id == "" {
		return Problems{missing("id")}
	}
	got, err := TypeOfID(id)
	if err != nil {
		return Problems{{Code: CodeInvalidID, Field: "id", Message: err.Error()}}
	}
	if got != want {
		return Problems{{Code: CodeInvalidID, Field: "id", Message: fmt.Sprintf("prefix implies %s but type is %s", got, want)}}
	}
	return nil
}

func checkSubject(subject string) Problems {
	if subject == "" {
		return Problems{missing("subject")}
	}
	t, err := TypeOfID(subject)
	if err != nil {
		return Problems{{Code: CodeInvalidID, Field: "subject", Message: err.Error()}}
	}
	if t != TypeParticular {
		return Problems{{Code: CodeInvalidID, Field: "subject", Message: "subject must reference a particular"}}
	}
	return nil
}

// checkSource enforces the format's minimum: at least one of author or harness.
func checkSource(field string, s Source) Problems {
	if strings.TrimSpace(s.Author) == "" && strings.TrimSpace(s.Harness) == "" {
		return Problems{{Code: CodeMissingField, Field: field, Message: "source must contain at least one of author or harness"}}
	}
	return nil
}

// ValidateMerge checks field-level constraints of a merge record.
func ValidateMerge(m *Merge) Problems {
	var ps Problems
	ps = append(ps, checkID(m.ID, TypeMerge)...)
	switch {
	case len(m.URIs) != 2:
		ps = append(ps, Problem{Code: CodeInvalidMerge, Field: "uris", Message: fmt.Sprintf("a merge must list exactly two uris, got %d", len(m.URIs))})
	case strings.TrimSpace(m.URIs[0]) == "" || strings.TrimSpace(m.URIs[1]) == "":
		ps = append(ps, Problem{Code: CodeInvalidMerge, Field: "uris", Message: "uris must be non-empty"})
	case m.URIs[0] == m.URIs[1]:
		ps = append(ps, Problem{Code: CodeInvalidMerge, Field: "uris", Message: "a merge cannot join a uri to itself"})
	}
	ps = append(ps, checkSource("source", m.Source)...)
	if m.Timestamp.IsZero() {
		ps = append(ps, missing("timestamp"))
	}
	if m.Retracted != nil {
		ps = append(ps, ValidateRetracted(m.Retracted)...)
		if m.Retracted.SupersededBy != "" {
			ps = append(ps, Problem{Code: CodeInvalidID, Field: "retracted.superseded-by", Message: "a merge is undone, not superseded; superseded-by is not allowed"})
		}
	}
	return ps
}

func checkContext(c Context) Problems {
	var ps Problems
	if c.Scope == "" {
		ps = append(ps, missing("context.scope"))
	} else if !ValidScope(c.Scope) {
		ps = append(ps, Problem{Code: CodeInvalidEnum, Field: "context.scope", Message: fmt.Sprintf("scope must be personal, organisation, or public, got %q", c.Scope)})
	}
	for i, t := range c.Topics {
		if strings.TrimSpace(t) == "" {
			ps = append(ps, Problem{Code: CodeMissingField, Field: fmt.Sprintf("context.topics[%d]", i), Message: "topic must be non-empty"})
		}
	}
	return ps
}

func checkConfidence(f *float64) Problems {
	if f == nil {
		return nil
	}
	if *f < 0 || *f > 1 || *f != *f {
		return Problems{{Code: CodeOutOfRange, Field: "confidence", Message: fmt.Sprintf("confidence must be in [0, 1], got %v", *f)}}
	}
	return nil
}
