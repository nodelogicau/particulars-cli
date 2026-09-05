package dkf

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
		Source:    Source{Author: "ben", Harness: "claude", Model: "claude-sonnet-4-6", Document: Document{Ref: "https://example.com/docs/architecture.md"}},
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
	c.Source = Source{Document: Document{Ref: "x"}}
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
	c.Source = Source{Document: Document{Ref: "https://x"}}
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

func TestPromotionRoundTrip(t *testing.T) {
	ts := time.Date(2026, 8, 24, 9, 30, 0, 0, time.UTC)
	pr := &Promotion{
		ID:        NewID(TypePublish),
		Claims:    []string{"clm_01916f03-b680-71a3-974f-9401ba374e1f", "syn_01933034-b1a0-705f-b788-2c7c58c46e29"},
		Scope:     ScopePublic,
		Reason:    "Architecture history cleared for the public docs site.",
		Source:    Source{Author: "ben", Harness: "claude"},
		Timestamp: ts,
	}
	if ps := ValidatePromotion(pr); len(ps) != 0 {
		t.Fatalf("valid promotion rejected: %v", ps)
	}
	data, err := Encode(pr)
	if err != nil {
		t.Fatal(err)
	}
	// Field order is normative: id, type, claims, scope, reason, source, timestamp.
	var keys []string
	for _, line := range strings.Split(string(data), "\n") {
		if len(line) > 0 && line[0] != ' ' && line[0] != '-' && strings.Contains(line, ":") {
			keys = append(keys, strings.SplitN(line, ":", 2)[0])
		}
	}
	want := []string{"id", "type", "claims", "scope", "reason", "source", "timestamp"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Errorf("field order:\n got %v\nwant %v\n%s", keys, want, data)
	}
	back, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := back.(*Promotion)
	if !ok {
		t.Fatalf("decoded as %T", back)
	}
	if got.ID != pr.ID || got.Scope != pr.Scope || len(got.Claims) != 2 || got.Claims[1] != pr.Claims[1] {
		t.Errorf("round trip lost data: %+v", got)
	}
	again, err := Encode(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Errorf("not byte-stable:\n%s\n---\n%s", data, again)
	}
	// A retraction is appended without disturbing the rest.
	got.SetRetracted(&Retracted{Timestamp: ts, Reason: "no longer cleared", Source: Source{Author: "ben"}})
	withR, err := Encode(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(withR, data) {
		t.Errorf("retraction should append, not rewrite:\n%s", withR)
	}
}

func TestPromotionValidation(t *testing.T) {
	base := func() *Promotion {
		return &Promotion{ID: NewID(TypePublish), Claims: []string{"clm_01916f03-b680-71a3-974f-9401ba374e1f"},
			Scope: ScopeOrganisation, Source: Source{Harness: "claude"}, Timestamp: time.Now()}
	}
	for name, mutate := range map[string]func(*Promotion){
		"empty claims":     func(p *Promotion) { p.Claims = nil },
		"a particular":     func(p *Promotion) { p.Claims = []string{"par_01916f03-b680-71a3-974f-9401ba374e1f"} },
		"a merge":          func(p *Promotion) { p.Claims = []string{"mrg_01916f03-b680-71a3-974f-9401ba374e1f"} },
		"a duplicate id":   func(p *Promotion) { p.Claims = append(p.Claims, p.Claims[0]) },
		"a bad id":         func(p *Promotion) { p.Claims = []string{"nope"} },
		"an invalid scope": func(p *Promotion) { p.Scope = "everyone" },
		"no source":        func(p *Promotion) { p.Source = Source{} },
		"superseded-by set": func(p *Promotion) {
			p.Retracted = &Retracted{Timestamp: time.Now(), Reason: "x", Source: Source{Author: "b"}, SupersededBy: "pub_01916f03-b680-71a3-974f-9401ba374e1f"}
		},
	} {
		p := base()
		mutate(p)
		if ps := ValidatePromotion(p); len(ps) == 0 {
			t.Errorf("%s should be rejected", name)
		}
	}
	// pub ids are canonical, so they never raise legacy_id.
	if !IsCanonicalID(NewID(TypePublish)) {
		t.Error("a minted promotion id must match the canonical pattern")
	}
	if !IsRetractableID(NewID(TypePublish)) {
		t.Error("promotions must be retractable")
	}
}

// TestExistingFilesAreByteStable is the guard for the document union: every
// shape written before it existed must decode and re-encode to identical
// bytes. Fixtures cover the shapes; DKF_ROUNDTRIP_DIR points it at a real
// workspace, which is where an unanticipated shape would actually live.
func TestExistingFilesAreByteStable(t *testing.T) {
	fixtures := []string{
		// claim with a bare-string document, the pre-union shape
		"id: clm_01a021ab-18ae-7910-883f-f8b8be27edb0\ntype: claim\nsubject: par_01a021ab-16e6-753f-8e6e-e8b69b25aeb5\ncontent: Uses Postgres 16.\nsource:\n  author: ben\n  harness: claude\n  document: docs/architecture.md\ncontext:\n  scope: personal\ntimestamp: 2026-08-20T09:00:00Z\nconfidence: 0.9\n",
		// document with a URL, and no other source fields
		"id: clm_01a021ab-18c7-7d36-bfbf-a04da42cc81f\ntype: claim\nsubject: par_01a021ab-16e6-753f-8e6e-e8b69b25aeb5\ncontent: A claim with a URL.\nsource:\n  harness: claude\n  document: https://example.com/a/b?c=d#e\ncontext:\n  scope: organisation\ntimestamp: 2026-08-20T09:00:00Z\n",
		// no document at all
		"id: clm_01a021ab-18d0-7d36-bfbf-a04da42cc820\ntype: claim\nsubject: par_01a021ab-16e6-753f-8e6e-e8b69b25aeb5\ncontent: No document.\nsource:\n  author: ben\ncontext:\n  scope: personal\ntimestamp: 2026-08-20T09:00:00Z\n",
		// a document that looks like a path with a line suffix
		"id: clm_01a021ab-18d9-7d36-bfbf-a04da42cc821\ntype: claim\nsubject: par_01a021ab-16e6-753f-8e6e-e8b69b25aeb5\ncontent: Line-suffixed path.\nsource:\n  author: ben\n  document: src/billing/cron.go:14\ncontext:\n  scope: personal\ntimestamp: 2026-08-20T09:00:00Z\n",
		// retracted claim, so the retracted block's key order is covered too
		"id: clm_01a021ab-18e2-7d36-bfbf-a04da42cc822\ntype: claim\nsubject: par_01a021ab-16e6-753f-8e6e-e8b69b25aeb5\ncontent: Withdrawn.\nsource:\n  author: ben\n  document: notes.md\ncontext:\n  scope: personal\ntimestamp: 2026-08-20T09:00:00Z\nretracted:\n  timestamp: 2026-08-21T09:00:00Z\n  reason: wrong\n  source:\n    author: ben\n  superseded-by: clm_01a021ab-18ae-7910-883f-f8b8be27edb0\n",
	}
	for i, in := range fixtures {
		obj, err := Decode([]byte(in))
		if err != nil {
			t.Fatalf("fixture %d: decode: %v", i, err)
		}
		out, err := Encode(obj)
		if err != nil {
			t.Fatalf("fixture %d: encode: %v", i, err)
		}
		if string(out) != in {
			t.Errorf("fixture %d is not byte-stable:\n--- in\n%s\n--- out\n%s", i, in, out)
		}
	}

	dir := os.Getenv("DKF_ROUNDTRIP_DIR")
	if dir == "" {
		t.Skip("set DKF_ROUNDTRIP_DIR to a workspace to check real files")
	}
	var checked, legacy int
	for _, sub := range []string{"claims", "syntheses", "particulars", "merges", "publishes"} {
		entries, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			path := filepath.Join(dir, sub, e.Name())
			in, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			obj, err := Decode(in)
			if err != nil {
				t.Errorf("%s: decode: %v", path, err)
				continue
			}
			out, err := Encode(obj)
			if err != nil {
				t.Errorf("%s: encode: %v", path, err)
				continue
			}
			// A legacy produced-by synthesis is deliberately not byte-stable:
			// the reader maps it to source and the writer emits source, which
			// is what legacy_produced_by warns about. Everything else must be.
			if bytes.Contains(in, []byte("produced-by:")) {
				if bytes.Contains(out, []byte("produced-by:")) {
					t.Errorf("%s: the encoder must never emit produced-by", path)
				}
				legacy++
				continue
			}
			if !bytes.Equal(in, out) {
				t.Errorf("%s is not byte-stable", path)
			}
			checked++
		}
	}
	t.Logf("round-tripped %d real files byte-identically (%d legacy produced-by, rewritten by design) from %s", checked, legacy, dir)
}

func TestDocumentUnion(t *testing.T) {
	// Scalar in, scalar out — the shape every existing claim has.
	scalarYAML := "id: clm_01a021ab-18ae-7910-883f-f8b8be27edb0\ntype: claim\nsubject: par_01a021ab-16e6-753f-8e6e-e8b69b25aeb5\ncontent: x\nsource:\n  author: ben\n  document: docs/a.md\ncontext:\n  scope: personal\ntimestamp: 2026-08-20T09:00:00Z\n"
	obj, err := Decode([]byte(scalarYAML))
	if err != nil {
		t.Fatal(err)
	}
	c := obj.(*Claim)
	if c.Source.Document.Ref != "docs/a.md" || c.Source.Document.structured() {
		t.Errorf("scalar decode: %+v", c.Source.Document)
	}
	if out, _ := Encode(c); string(out) != scalarYAML {
		t.Errorf("scalar re-encode differs:\n%s", out)
	}

	// Mapping round-trips, keys in the order uri, hash, quote.
	c.Source.Document = Document{Ref: "docs/a.md", Hash: "sha256:" + strings.Repeat("a", 64), Quote: "the billing service listens on 8443"}
	out, err := Encode(c)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "    ") && strings.Contains(line, ":") {
			keys = append(keys, strings.TrimSpace(strings.SplitN(line, ":", 2)[0]))
		}
	}
	if strings.Join(keys, ",") != "ref,hash,quote" {
		t.Errorf("document key order: %v\n%s", keys, out)
	}
	back, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := back.(*Claim).Source.Document; got != c.Source.Document {
		t.Errorf("mapping round trip: %+v", got)
	}
	if again, _ := Encode(back); !bytes.Equal(out, again) {
		t.Error("mapping form is not byte-stable")
	}

	// JSON mirrors YAML, so an existing consumer still sees a string.
	if b, _ := json.Marshal(Document{Ref: "docs/a.md"}); string(b) != `"docs/a.md"` {
		t.Errorf("scalar JSON: %s", b)
	}
	if b, _ := json.Marshal(Document{Ref: "u", Quote: "q"}); !strings.HasPrefix(string(b), `{"ref"`) {
		t.Errorf("mapping JSON: %s", b)
	}
	var d Document
	if err := json.Unmarshal([]byte(`"docs/a.md"`), &d); err != nil || d.Ref != "docs/a.md" {
		t.Errorf("scalar JSON decode: %v %+v", err, d)
	}
	if err := json.Unmarshal([]byte(`{"uri":"u","hash":"sha256:`+strings.Repeat("b", 64)+`"}`), &d); err != nil || d.Hash == "" {
		t.Errorf("mapping JSON decode: %v %+v", err, d)
	}

	// Validation.
	for name, doc := range map[string]Document{
		"hash without uri":  {Hash: "sha256:" + strings.Repeat("a", 64)},
		"quote without uri": {Quote: "something"},
		"short sha256":      {Ref: "u", Hash: "sha256:abc"},
		"no algorithm":      {Ref: "u", Hash: strings.Repeat("a", 64)},
		"empty algorithm":   {Ref: "u", Hash: ":" + strings.Repeat("a", 64)},
		"uppercase hash":    {Ref: "u", Hash: "sha256:" + strings.Repeat("A", 64)},
		"unprefixed hash":   {Ref: "u", Hash: strings.Repeat("a", 64)},
		"blank quote":       {Ref: "u", Quote: "   "},
	} {
		if ps := doc.Validate("source.document"); len(ps) == 0 {
			t.Errorf("%s should be rejected", name)
		}
	}
	for name, doc := range map[string]Document{
		"bare reference": {Ref: "chat session 2026-08-22"},
		"nothing at all": {},
		"full mapping":   {Ref: "u", Hash: "sha256:" + strings.Repeat("f", 64), Quote: "q"},
		// Another implementation's algorithm is accepted, not rejected: a
		// reader that refused it could not check anything it wrote.
		"another algorithm": {Ref: "u", Hash: "blake3:" + strings.Repeat("c", 32)},
	} {
		if ps := doc.Validate("source.document"); len(ps) != 0 {
			t.Errorf("%s should be valid: %v", name, ps)
		}
	}
}

func TestHashAndQuoteNormalisation(t *testing.T) {
	// The same content from a CRLF and an LF checkout hashes identically.
	lf := "line one\nline two\nline three\n"
	crlf := "line one\r\nline two\r\nline three\r\n"
	if HashDocumentBytes([]byte(lf)) != HashDocumentBytes([]byte(crlf)) {
		t.Error("CRLF and LF forms of the same content must hash the same")
	}
	// Everything else is an edit and must change the hash.
	for name, variant := range map[string]string{
		"trailing whitespace": "line one \nline two\nline three\n",
		"missing final line":  "line one\nline two\n",
		"changed word":        "line one\nline TWO\nline three\n",
	} {
		if HashDocumentBytes([]byte(variant)) == HashDocumentBytes([]byte(lf)) {
			t.Errorf("%s should change the hash", name)
		}
	}
	if h := HashDocumentBytes([]byte(lf)); !hashPattern.MatchString(h) {
		t.Errorf("hash shape: %q", h)
	}
	if got, err := HashDocument(strings.NewReader(crlf)); err != nil || got != HashDocumentBytes([]byte(lf)) {
		t.Errorf("HashDocument: %v %q", err, got)
	}

	doc := "Preamble.\n\nIn staging, the billing service listens on 443.\n\n```go\nfunc main() {\n\tx := 1\n}\n```\n"
	// A block-scalar quote carries a trailing newline that is an artefact of
	// YAML, not of the source.
	if !QuoteMatches(doc, "In staging, the billing service listens on 443.\n") {
		t.Error("a block-scalar quote should match")
	}
	if !QuoteMatches(doc, "  In staging, the billing service listens on 443.  ") {
		t.Error("surrounding whitespace on the quote should be trimmed")
	}
	// Whitespace folds on both sides: the words are what was quoted.
	if !QuoteMatches(doc, "func main() {\n\tx := 1\n}") {
		t.Error("an indented code quote should match with its indentation")
	}
	if !QuoteMatches(doc, "func main() {\nx := 1\n}") {
		t.Error("a re-indented code quote should still match")
	}
	if !QuoteMatches(strings.ReplaceAll(doc, "\t", "    "), "func main() {\n\tx := 1\n}") {
		t.Error("a tab-indented quote should match a space-indented document")
	}
	// A quote spanning a hard line wrap — the case from issue #9 — matches,
	// and keeps matching when the document is wrapped at another column.
	wrapped := "Preamble.\n\nIn staging, the billing service\nlistens on 443.\n"
	for _, variant := range []string{wrapped, "In staging,\nthe billing service listens\non 443.", strings.ReplaceAll(wrapped, "\n", "\r\n")} {
		if !QuoteMatches(variant, "In staging, the billing service listens on 443.") {
			t.Errorf("a quote across a line wrap should match: %q", variant)
		}
	}
	if !QuoteMatches(doc, "Preamble. In staging, the billing service listens on 443.") {
		t.Error("a quote spanning a blank line should match")
	}
	// CRLF on either side still matches.
	if !QuoteMatches(strings.ReplaceAll(doc, "\n", "\r\n"), "In staging, the billing service listens on 443.") {
		t.Error("a CRLF document should still match an LF quote")
	}
	for _, absent := range []string{"In production, the billing service listens on 443.", "In staging, the billing service listens on 8443.", "in staging, the billing service listens on 443.", "", "   "} {
		if QuoteMatches(doc, absent) {
			t.Errorf("%q should not match", absent)
		}
	}
}

func TestRetractionKind(t *testing.T) {
	base := sampleClaim()
	ts := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	// Key order: timestamp, reason, source, kind, superseded-by.
	base.Retracted = &Retracted{Timestamp: ts, Reason: "misread the port", Source: Source{Author: "ben"},
		Kind: KindDefect, SupersededBy: "clm_01a021ab-18ae-7910-883f-f8b8be27edb0"}
	out, err := Encode(base)
	if err != nil {
		t.Fatal(err)
	}
	block := string(out[strings.Index(string(out), "retracted:"):])
	var keys []string
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.Contains(line, ":") {
			keys = append(keys, strings.TrimSpace(strings.SplitN(line, ":", 2)[0]))
		}
	}
	if strings.Join(keys, ",") != "timestamp,reason,source,kind,superseded-by" {
		t.Errorf("retracted key order: %v\n%s", keys, block)
	}
	back, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	if back.(*Claim).Retracted.Kind != KindDefect {
		t.Error("kind lost in round trip")
	}
	if again, _ := Encode(back); !bytes.Equal(out, again) {
		t.Error("a retraction with a kind is not byte-stable")
	}

	// Optional, and absent stays absent.
	base.Retracted = &Retracted{Timestamp: ts, Reason: "x", Source: Source{Author: "ben"},
		SupersededBy: "clm_01a021ab-18ae-7910-883f-f8b8be27edb0"}
	out, _ = Encode(base)
	if strings.Contains(string(out), "kind:") {
		t.Error("no kind key should be emitted when none was declared")
	}
	// A replacement must not imply a kind: that is the inference the spec forbids.
	if k := base.Retracted.Kind; k != "" {
		t.Errorf("kind must never be inferred from superseded-by, got %q", k)
	}
	if ps := ValidateRetracted(base.Retracted); len(ps) != 0 {
		t.Errorf("a kindless retraction is valid: %v", ps)
	}

	for _, k := range []RetractionKind{KindDefect, KindSupersession, KindProvenanceFailure} {
		if !ValidRetractionKind(k) {
			t.Errorf("%q should be valid", k)
		}
	}
	for _, k := range []RetractionKind{"wrong", "Defect", "supersede", "provenance failure"} {
		r := &Retracted{Timestamp: ts, Reason: "x", Source: Source{Author: "ben"}, Kind: k}
		if ps := ValidateRetracted(r); len(ps) == 0 {
			t.Errorf("kind %q should be rejected", k)
		}
	}
}

func TestLegacyDocumentURIAccepted(t *testing.T) {
	// A file written by v0.8.0, before the rename. It can never be rewritten —
	// appending a retraction is the only permitted modification — so the
	// reader accepts it in perpetuity and flags it instead.
	in := "id: clm_01a021ab-18ae-7910-883f-f8b8be27edb0\ntype: claim\nsubject: par_01a021ab-16e6-753f-8e6e-e8b69b25aeb5\ncontent: x\nsource:\n  author: ben\n  document:\n    uri: docs/a.md\n    quote: something quoted\ncontext:\n  scope: personal\ntimestamp: 2026-08-20T09:00:00Z\n"
	obj, err := Decode([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	doc := obj.(*Claim).Source.Document
	if doc.Ref != "docs/a.md" || doc.Quote != "something quoted" {
		t.Fatalf("legacy uri not read: %+v", doc)
	}
	if !doc.LegacyURI() {
		t.Error("reading a uri key should be recorded so validate can say so")
	}
	if ps := doc.Validate("source.document"); len(ps) != 0 {
		t.Errorf("a legacy document is valid, not invalid: %v", ps)
	}
	// Rewriting emits ref, which is why it is not byte-stable — the same
	// deliberate exception as produced-by.
	out, err := Encode(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "    ref: docs/a.md") || strings.Contains(string(out), "uri:") {
		t.Errorf("the encoder must write ref:\n%s", out)
	}
	// ref wins if somehow both are present.
	both, err := Decode([]byte(strings.Replace(in, "    uri: docs/a.md", "    ref: r.md\n    uri: u.md", 1)))
	if err != nil {
		t.Fatal(err)
	}
	if d := both.(*Claim).Source.Document; d.Ref != "r.md" || d.LegacyURI() {
		t.Errorf("ref should win over uri: %+v", d)
	}
}

func TestEvidentialField(t *testing.T) {
	c := sampleClaim()
	c.Evidential = EvidentialObserved
	out, err := Encode(c)
	if err != nil {
		t.Fatal(err)
	}
	// Between timestamp and confidence, per the spec's field order.
	text := string(out)
	ti, ei, ci := strings.Index(text, "\ntimestamp:"), strings.Index(text, "\nevidential:"), strings.Index(text, "\nconfidence:")
	if ti >= ei || ei >= ci || ei < 0 {
		t.Errorf("field order timestamp < evidential < confidence violated:\n%s", text)
	}
	back, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	if back.(*Claim).Evidential != EvidentialObserved {
		t.Error("evidential lost in round trip")
	}
	if again, _ := Encode(back); !bytes.Equal(out, again) {
		t.Error("not byte-stable with evidential")
	}
	// Absent stays absent: the pre-evidential shape is untouched.
	c.Evidential = ""
	out, _ = Encode(c)
	if strings.Contains(string(out), "evidential") {
		t.Errorf("no evidential key when none declared:\n%s", out)
	}
	// If present it must be one of the three; absence is not an error.
	if ps := ValidateClaim(c); len(ps) != 0 {
		t.Errorf("absence is lenient: %v", ps)
	}
	c.Evidential = "opinion"
	if ps := ValidateClaim(c); len(ps) == 0 {
		t.Error("an unknown evidential must be rejected")
	}
	c.Evidential = "undeclared"
	if ps := ValidateClaim(c); len(ps) == 0 {
		t.Error("undeclared is a reader's report, never a value")
	}
	// held + confidence is the one shared rule, wrong wherever it appears.
	c.Evidential = EvidentialHeld
	conf := 0.9
	c.Confidence = &conf
	found := false
	for _, p := range ValidateClaim(c) {
		if p.Code == CodeConfidenceOnHeld {
			found = true
		}
	}
	if !found {
		t.Error("held with confidence must be rejected as confidence_on_held")
	}
	c.Confidence = nil
	if ps := ValidateClaim(c); len(ps) != 0 {
		t.Errorf("held without confidence is valid: %v", ps)
	}

	for _, m := range []string{MethodReconciliation, MethodQualification, MethodPositions} {
		if !ValidMethod(m) {
			t.Errorf("%q should be valid", m)
		}
	}
	if ValidMethod("consensus") || ValidMethod("") {
		t.Error("the method vocabulary is closed")
	}
}
