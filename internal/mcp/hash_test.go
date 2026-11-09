package mcp

import (
	"strings"
	"testing"
)

func TestClaimAssertRefusesNoncanonicalHash(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	h.ok("particular_define", map[string]any{"label": "Billing"})
	hex := strings.Repeat("ab", 32)
	assert := func(hash string) map[string]any {
		return map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443",
			"source": map[string]any{"document": map[string]any{"ref": "https://example.com/a", "hash": hash}}}
	}
	for hash, names := range map[string]string{
		"sha256:" + strings.ToUpper(hex): "sha256:" + hex,
		"SHA-256:" + hex:                 "sha256:" + hex,
		"sha256:9f2a":                    "64 lowercase hex digits",
	} {
		if text := h.fail("claim_assert", assert(hash), "invalid"); !strings.Contains(text, names) {
			t.Errorf("%s: refusal should name %q: %q", hash, names, text)
		}
	}
	h.ok("claim_assert", assert("sha256:"+hex))
	h.ok("claim_assert", assert("blake3:"+strings.Repeat("c", 32)))
}
