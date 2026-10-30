package dkf

import (
	"strings"
	"testing"
)

const (
	basisA = "clm_019196a5-8b4c-7def-8abc-0123456789b1"
	basisB = "syn_019196a5-8b4c-7def-8abc-0123456789b2"
)

func inferredClaim(basis ...string) *Claim {
	c := sampleClaim()
	c.Evidential = EvidentialInferred
	c.Basis = basis
	return c
}

func hasCode(ps Problems, code string) bool {
	for _, p := range ps {
		if p.Code == code {
			return true
		}
	}
	return false
}

func TestBasisRoundTripAndOrder(t *testing.T) {
	c := inferredClaim(basisA, basisB)
	out, err := Encode(c)
	if err != nil {
		t.Fatal(err)
	}
	keys := strings.Join(topLevelKeys(out), ",")
	if !strings.Contains(keys, "timestamp,evidential,basis,confidence") {
		t.Fatalf("basis must sit between evidential and confidence, got %s", keys)
	}
	back, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	got := back.(*Claim).Basis
	if len(got) != 2 || got[0] != basisA || got[1] != basisB {
		t.Fatalf("basis did not round-trip: %v", got)
	}
	if ok, _ := IsCanonical(out); !ok {
		t.Error("encoded claim with a basis is not canonical")
	}
	if len(back.(*Claim).InputIDs()) != 0 {
		t.Error("a claim's basis must never be reported as inputs")
	}
	if ids := Cited(back.(*Claim)); len(ids) != 2 {
		t.Errorf("Cited = %v, want the basis", ids)
	}
	if ps := ValidateClaim(c); len(ps) != 0 {
		t.Errorf("valid inferred claim reported %v", ps)
	}
}

func TestBasisOutOfOrderIsNonCanonical(t *testing.T) {
	out, _ := Encode(inferredClaim(basisA))
	moved := strings.Replace(string(out), "evidential: inferred\nbasis:\n  - "+basisA+"\n", "basis:\n  - "+basisA+"\nevidential: inferred\n", 1)
	if moved == string(out) {
		t.Fatal("fixture did not move the basis")
	}
	if ok, _ := IsCanonical([]byte(moved)); ok {
		t.Error("basis before evidential must not be canonical")
	}
}

func TestEmptyBasisSurvivesAndFails(t *testing.T) {
	out, _ := Encode(inferredClaim())
	withEmpty := strings.Replace(string(out), "evidential: inferred\n", "evidential: inferred\nbasis: []\n", 1)
	obj, err := Decode([]byte(withEmpty))
	if err != nil {
		t.Fatal(err)
	}
	c := obj.(*Claim)
	if c.Basis == nil {
		t.Fatal("`basis: []` must decode as present and empty")
	}
	if !hasCode(ValidateClaim(c), CodeEmptyBasis) {
		t.Error("an empty basis must be reported")
	}
	if ok, _ := IsCanonical([]byte(withEmpty)); !ok {
		t.Error("`basis: []` must re-encode as written, so the defect stays visible")
	}
}

func TestBasisNotInferred(t *testing.T) {
	for _, e := range []Evidential{EvidentialObserved, EvidentialHeld, ""} {
		c := inferredClaim(basisA)
		c.Evidential = e
		c.Confidence = nil
		if !hasCode(ValidateClaim(c), CodeBasisNotInferred) {
			t.Errorf("basis on a %q claim must be basis_not_inferred", e)
		}
	}
	s := sampleSynthesis()
	out, _ := Encode(s)
	obj, err := Decode([]byte(string(out) + "basis:\n  - " + basisA + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	syn := obj.(*Synthesis)
	if !hasCode(ValidateSynthesis(syn), CodeBasisNotInferred) {
		t.Error("basis on a synthesis must be basis_not_inferred")
	}
	if len(syn.BasisIDs()) != 0 || len(Cited(syn)) != len(syn.Inputs) {
		t.Error("a synthesis's stray basis must never be traversed")
	}
	if enc, _ := Encode(syn); strings.Contains(string(enc), "\nbasis:") {
		t.Error("a stray basis must never be serialised")
	}
}

func TestBasisIDs(t *testing.T) {
	c := inferredClaim(sampleParticular().ID, "not-an-id")
	ps := ValidateClaim(c)
	if n := countCode(ps, CodeInvalidID); n != 2 {
		t.Errorf("want two invalid_id problems, got %v", ps)
	}
	self := inferredClaim(sampleClaim().ID)
	if !hasCode(ValidateClaim(self), CodeSelfBasis) {
		t.Error("a self-citing basis must be self_basis")
	}
}

func countCode(ps Problems, code string) int {
	n := 0
	for _, p := range ps {
		if p.Code == code {
			n++
		}
	}
	return n
}
