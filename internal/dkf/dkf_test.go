package dkf

import (
	"bytes"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

func f(v float64) *float64 { return &v }

var ts = time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)

func sampleParticular() *Particular {
	return &Particular{ID: "par_019196a5-8b4c-7def-8abc-0123456789ab", URI: "https://example.com/particulars/project-x", Label: "Project X", Aliases: []string{"ProjectX", "project_x"}}
}

func sampleClaim() *Claim {
	return &Claim{
		ID: "clm_019196a5-8b4c-7def-8abc-0123456789ac", Subject: "par_019196a5-8b4c-7def-8abc-0123456789ab",
		Content:   "Project X uses a microservices architecture, with separate\nservices for auth, billing, and core API.\n",
		Source:    Source{Author: "ben", Harness: "claude", Model: "claude-sonnet-4-6", Document: "https://example.com/docs/architecture.md"},
		Context:   Context{Scope: ScopeOrganisation, Topics: []string{"architecture", "distributed-systems"}},
		Timestamp: ts, Confidence: f(0.9),
	}
}

func sampleSynthesis() *Synthesis {
	return &Synthesis{
		ID: "syn_019196a5-8b4c-7def-8abc-0123456789ad", Subject: "par_019196a5-8b4c-7def-8abc-0123456789ab",
		Content: "Consolidated into a monolith in November 2024.",
		Inputs: []Input{
			{ID: "clm_019196a5-8b4c-7def-8abc-0123456789ac", Role: RoleThesis, Weight: WeightPrimary},
			{ID: "clm_019196a5-8b4c-7def-8abc-0123456789ae", Role: RoleAntithesis, Weight: WeightQualifying},
		},
		Unresolved: "Compliance basis unsourced.",
		Source:     Source{Harness: "claude", Model: "claude-sonnet-4-6"},
		Method:     DefaultMethod, Timestamp: ts,
		Context:    Context{Scope: ScopeOrganisation},
		Confidence: f(0.85),
	}
}

func sampleMerge() *Merge {
	return &Merge{ID: "mrg_019196a5-8b4c-7def-8abc-0123456789b0", URIs: []string{"https://example.com/particulars/project-x", "urn:dkf:w:projectx"}, Reason: "Same project", Source: Source{Author: "ben", Harness: "claude"}, Timestamp: ts}
}

func TestRoundTripStable(t *testing.T) {
	for _, obj := range []Object{sampleParticular(), sampleClaim(), sampleSynthesis(), sampleMerge()} {
		first, err := Encode(obj)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := Decode(first)
		if err != nil {
			t.Fatalf("%s: %v\n%s", obj.ObjectType(), err, first)
		}
		second, err := Encode(decoded)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			t.Errorf("%s not stable:\n--- first\n%s\n--- second\n%s", obj.ObjectType(), first, second)
		}
		if ok, _ := IsCanonical(first); !ok {
			t.Errorf("%s: encoded output not reported canonical", obj.ObjectType())
		}
		if strings.HasPrefix(string(first), "---") {
			t.Errorf("%s: document marker present", obj.ObjectType())
		}
	}
}

func topLevelKeys(doc []byte) []string {
	var keys []string
	for _, line := range strings.Split(string(doc), "\n") {
		if line == "" || line[0] == ' ' || line[0] == '-' {
			continue
		}
		keys = append(keys, strings.SplitN(line, ":", 2)[0])
	}
	return keys
}

func TestSpecFieldOrder(t *testing.T) {
	cases := []struct {
		obj  Object
		want []string
	}{
		{sampleParticular(), []string{"id", "type", "uri", "label", "aliases"}},
		{sampleClaim(), []string{"id", "type", "subject", "content", "source", "context", "timestamp", "confidence"}},
		{sampleSynthesis(), []string{"id", "type", "subject", "content", "inputs", "unresolved", "source", "method", "timestamp", "context", "confidence"}},
		{sampleMerge(), []string{"id", "type", "uris", "reason", "source", "timestamp"}},
	}
	for _, c := range cases {
		out, err := Encode(c.obj)
		if err != nil {
			t.Fatal(err)
		}
		got := topLevelKeys(out)
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("%s order = %v, want %v", c.obj.ObjectType(), got, c.want)
		}
	}
}

func TestEncodeDetails(t *testing.T) {
	out, _ := Encode(sampleClaim())
	s := string(out)
	for _, want := range []string{"content: |\n  Project X uses", "timestamp: 2026-08-20T09:00:00Z", "confidence: 0.9", "  scope: organisation\n  topics:\n    - architecture"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	c := sampleClaim()
	c.Confidence = nil
	c.Context.Topics = nil
	out, _ = Encode(c)
	if strings.Contains(string(out), "confidence") || strings.Contains(string(out), "topics") {
		t.Errorf("unset optionals should be omitted:\n%s", out)
	}
}

func TestRetractedEncoding(t *testing.T) {
	c := sampleClaim()
	base, _ := Encode(c)
	r := &Retracted{Timestamp: ts.Add(time.Hour), Reason: "Port was 8443", Source: Source{Author: "ben"}, SupersededBy: "clm_019196a5-8b4c-7def-8abc-0123456789af"}
	block, err := EncodeRetracted(r)
	if err != nil {
		t.Fatal(err)
	}
	appended := append(append([]byte{}, base...), block...)
	obj, err := Decode(appended)
	if err != nil {
		t.Fatalf("appended file does not parse: %v\n%s", err, appended)
	}
	got := obj.(*Claim)
	if got.Retracted == nil || got.Retracted.Reason != "Port was 8443" || got.Retracted.SupersededBy != r.SupersededBy {
		t.Errorf("retracted block not decoded: %+v", got.Retracted)
	}
	// Appended form must equal the canonical encoding of the retracted object.
	c.Retracted = r
	canon, _ := Encode(c)
	if !bytes.Equal(canon, appended) {
		t.Errorf("append != canonical:\n--- appended\n%s\n--- canonical\n%s", appended, canon)
	}
}

func TestDecodeTolerant(t *testing.T) {
	doc := "id: clm_01j9xk2p3q4r5s6t\ntype: claim\nsubject: par_01j9xk2p3q4r5s6t\ncontent: hello\nsource:\n  author: ben\n  extra: ignored\ncontext:\n  scope: personal\ntimestamp: 2024-08-20T09:00:00Z\nconfidence: 1\nfuture-field: whatever\n"
	obj, err := Decode([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	c := obj.(*Claim)
	if c.ID != "clm_01j9xk2p3q4r5s6t" || c.Confidence == nil || *c.Confidence != 1 {
		t.Errorf("unexpected decode: %+v", c)
	}
	if ps := ValidateClaim(c); len(ps) != 0 {
		t.Errorf("foreign id should validate: %v", ps)
	}
	if _, err := Decode([]byte("id: x\n")); err == nil {
		t.Error("missing type should fail")
	}
	if _, err := Decode([]byte("type: thing\n")); err == nil {
		t.Error("unknown type should fail")
	}
}

func TestIDs(t *testing.T) {
	id := NewID(TypeClaim)
	if !IsCanonicalID(id) {
		t.Errorf("minted id not canonical: %s", id)
	}
	if id != strings.ToLower(id) {
		t.Errorf("id not lowercase: %s", id)
	}
	typ, err := TypeOfID(id)
	if err != nil || typ != TypeClaim {
		t.Errorf("TypeOfID(%s) = %v, %v", id, typ, err)
	}
	if _, err := TypeOfID("foo_123"); err == nil {
		t.Error("unknown prefix accepted")
	}
	if !IsValidID("clm_01j9xk2p3q4r5s6t") || IsCanonicalID("clm_01j9xk2p3q4r5s6t") {
		t.Error("lenient/canonical mismatch for foreign id")
	}
	if !regexp.MustCompile(`^[0-9a-f-]{36}$`).MatchString(NewUUID()) {
		t.Error("workspace uuid not canonical")
	}
}

func TestBurstMintOrdered(t *testing.T) {
	const n = 500
	ids := make([]string, n)
	for i := range ids {
		ids[i] = NewID(TypeClaim)
	}
	sorted := append([]string{}, ids...)
	sort.Strings(sorted)
	for i := range ids {
		if ids[i] != sorted[i] {
			t.Fatalf("ids not in creation order at %d: %s vs %s", i, ids[i], sorted[i])
		}
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = true
	}
}

func TestValidation(t *testing.T) {
	if ps := ValidateClaim(sampleClaim()); len(ps) != 0 {
		t.Errorf("sample claim invalid: %v", ps)
	}
	if ps := ValidateSynthesis(sampleSynthesis()); len(ps) != 0 {
		t.Errorf("sample synthesis invalid: %v", ps)
	}
	c := sampleClaim()
	c.Confidence = f(1.5)
	c.Context.Scope = "team"
	c.Source = Source{Document: "x"}
	ps := ValidateClaim(c)
	codes := map[string]bool{}
	for _, p := range ps {
		codes[p.Code+":"+p.Field] = true
	}
	for _, want := range []string{"out_of_range:confidence", "invalid_enum:context.scope", "missing_field:source"} {
		if !codes[want] {
			t.Errorf("missing problem %s in %v", want, ps)
		}
	}
	s := sampleSynthesis()
	s.Unresolved = " "
	s.Inputs[0].Role = "support"
	s.Inputs = append(s.Inputs, Input{ID: "par_x", Role: RoleThesis})
	ps = ValidateSynthesis(s)
	codes = map[string]bool{}
	for _, p := range ps {
		codes[p.Code+":"+p.Field] = true
	}
	for _, want := range []string{"missing_field:unresolved", "invalid_enum:inputs[0].role", "invalid_id:inputs[2].id"} {
		if !codes[want] {
			t.Errorf("missing problem %s in %v", want, ps)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Project X":       "project-x",
		"Billing Service": "billing-service",
		"Auth & Sessions": "auth-sessions",
		"  Café--Zürich ": "cafe-zurich",
		"!!!":             "",
		"snake_case_name": "snake-case-name",
		"v2.0 API":        "v2-0-api",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMintURI(t *testing.T) {
	cases := []struct{ base, ws, slug, want string }{
		{"https://example.com/particulars/", "W", "billing-service", "https://example.com/particulars/billing-service"},
		{"", "0191-ws", "auth-sessions", "urn:dkf:0191-ws:auth-sessions"},
	}
	for _, c := range cases {
		if got := MintURI(c.base, c.ws, c.slug); got != c.want {
			t.Errorf("MintURI(%q,%q,%q) = %q, want %q", c.base, c.ws, c.slug, got, c.want)
		}
	}
}

func TestParseTime(t *testing.T) {
	got, err := ParseTime("2024-08-20T19:00:00+10:00")
	if err != nil || FormatTime(got) != "2024-08-20T09:00:00Z" {
		t.Errorf("ParseTime offset: %v %v", got, err)
	}
	if _, err := ParseTime("yesterday"); err == nil {
		t.Error("bad timestamp accepted")
	}
}

func TestLegacyProducedBy(t *testing.T) {
	legacy := "id: syn_01a0-legacy\ntype: synthesis\nsubject: par_x\ncontent: c\ninputs:\n  - id: clm_a\n    role: thesis\nunresolved: n\nproduced-by:\n  harness: claude\n  model: m\ntimestamp: 2026-08-20T09:00:00Z\ncontext:\n  scope: personal\n"
	obj, err := Decode([]byte(legacy))
	if err != nil {
		t.Fatal(err)
	}
	s := obj.(*Synthesis)
	if !s.LegacyProducedBy || s.Source.Harness != "claude" || s.Source.Model != "m" || s.ConflictingProvenance {
		t.Errorf("legacy mapping: %+v", s)
	}
	if ps := ValidateSynthesis(s); len(ps) != 0 {
		t.Errorf("legacy synthesis should validate field-wise: %v", ps)
	}
	out, _ := Encode(s)
	if strings.Contains(string(out), "produced-by") || !strings.Contains(string(out), "source:\n  harness: claude") {
		t.Errorf("re-encode must use source:\n%s", out)
	}
	both := strings.Replace(legacy, "produced-by:", "source:\n  harness: x\nproduced-by:", 1)
	obj, _ = Decode([]byte(both))
	s = obj.(*Synthesis)
	if !s.ConflictingProvenance || s.Source.Harness != "x" {
		t.Errorf("both blocks: %+v", s)
	}
	found := false
	for _, p := range ValidateSynthesis(s) {
		if p.Code == CodeConflictingProvenance {
			found = true
		}
	}
	if !found {
		t.Error("conflicting_provenance not reported")
	}
	neither := strings.Replace(legacy, "produced-by:\n  harness: claude\n  model: m\n", "", 1)
	obj, _ = Decode([]byte(neither))
	if ps := ValidateSynthesis(obj.(*Synthesis)); !strings.Contains(ps.Error(), "source") {
		t.Errorf("missing provenance should fail: %v", ps)
	}
}

func TestSourceMinimum(t *testing.T) {
	c := sampleClaim()
	c.Source = Source{Harness: "claude", Model: "m"}
	if ps := ValidateClaim(c); len(ps) != 0 {
		t.Errorf("agent-only source should be valid: %v", ps)
	}
	c.Source = Source{Author: "ben"}
	if ps := ValidateClaim(c); len(ps) != 0 {
		t.Errorf("human-only source should be valid: %v", ps)
	}
	c.Source = Source{Document: "https://x"}
	if ps := ValidateClaim(c); len(ps) != 1 || ps[0].Field != "source" {
		t.Errorf("document-only source should fail on source: %v", ps)
	}
	s := sampleSynthesis()
	s.Source = Source{Author: "ben"}
	if ps := ValidateSynthesis(s); len(ps) != 1 || ps[0].Field != "source.harness" {
		t.Errorf("synthesis without harness: %v", ps)
	}
	r := &Retracted{Timestamp: ts, Reason: "r", Source: Source{Harness: "claude"}}
	if ps := ValidateRetracted(r); len(ps) != 0 {
		t.Errorf("agent retraction should be valid: %v", ps)
	}
}

func TestMergeValidation(t *testing.T) {
	m := sampleMerge()
	if ps := ValidateMerge(m); len(ps) != 0 {
		t.Errorf("sample merge invalid: %v", ps)
	}
	m.URIs = append(m.URIs, "urn:x")
	if ps := ValidateMerge(m); len(ps) != 1 || ps[0].Code != CodeInvalidMerge {
		t.Errorf("three uris: %v", ps)
	}
	m.URIs = []string{"urn:a", "urn:a"}
	if ps := ValidateMerge(m); len(ps) != 1 || ps[0].Code != CodeInvalidMerge {
		t.Errorf("self merge: %v", ps)
	}
	m = sampleMerge()
	m.Retracted = &Retracted{Timestamp: ts, Reason: "r", Source: Source{Author: "ben"}, SupersededBy: "clm_019196a5-8b4c-7def-8abc-0123456789ac"}
	if ps := ValidateMerge(m); len(ps) != 1 || ps[0].Field != "retracted.superseded-by" {
		t.Errorf("superseded-by on merge: %v", ps)
	}
	if id := NewID(TypeMerge); !strings.HasPrefix(id, "mrg_") || !IsCanonicalID(id) || !IsRetractableID(id) || IsAssertionID(id) {
		t.Errorf("merge id: %s", id)
	}
}

func TestBaseURI(t *testing.T) {
	if NormaliseBaseURI("https://e.com/p") != "https://e.com/p/" || NormaliseBaseURI("https://e.com/p/") != "https://e.com/p/" || NormaliseBaseURI("") != "" {
		t.Error("normalise")
	}
	if !ValidBaseURI("") || !ValidBaseURI("https://e.com/p/") || ValidBaseURI("https://e.com/p") {
		t.Error("valid")
	}
	if MintURI("https://e.com/p/", "w", "x") != "https://e.com/p/x" {
		t.Error("mint")
	}
}
