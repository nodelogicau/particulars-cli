package mcp

import (
	"strings"
	"testing"
)

func TestClaimAssertBasisOverMCP(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	h.ok("particular_define", map[string]any{"label": "Certs"})
	obs := func(content string) string {
		return id(h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Certs", "content": content}), "claim")
	}
	a, b := obs("A went wrong"), obs("B went wrong")
	rule := func(ev string, basis ...string) map[string]any {
		args := map[string]any{"evidential": ev, "particular_id": "Certs", "content": "renewals go wrong when the TTL is long"}
		if basis != nil {
			args["basis"] = basis
		}
		return args
	}

	r := h.ok("claim_assert", rule("inferred", a, b))
	got, _ := r["claim"].(map[string]any)["basis"].([]any)
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Errorf("basis = %v", got)
	}
	if r["warnings"] != nil {
		t.Errorf("no warnings expected: %v", r["warnings"])
	}

	if text := h.fail("claim_assert", rule("held", a), "usage"); !strings.Contains(text, "basis") {
		t.Errorf("refusal should name basis: %q", text)
	}
	h.fail("claim_assert", rule("inferred", a, a), "usage")
	h.fail("claim_assert", rule("inferred", "clm_019196a5-8b4c-7def-8abc-0123456789ff"), "not_found")

	r = h.ok("claim_assert", rule("inferred"))
	if ws, _ := r["warnings"].([]any); len(ws) != 1 || !strings.Contains(ws[0].(string), "inferred_without_basis") {
		t.Errorf("want an inferred_without_basis warning, got %v", r["warnings"])
	}
}
