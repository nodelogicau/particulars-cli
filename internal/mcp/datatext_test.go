package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestEveryToolRepeatsItsDataAsText calls every registered tool successfully
// and checks that the last text block decodes to exactly the structured
// content, so a client that hands the model only text loses nothing (#15).
func TestEveryToolRepeatsItsDataAsText(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	called := map[string]bool{}
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		res, err := h.cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if res.IsError {
			t.Fatalf("%s failed: %v", name, res.Content)
		}
		called[name] = true
		if len(res.Content) < 2 {
			t.Fatalf("%s: want a summary and a data block, got %d block(s)", name, len(res.Content))
		}
		text, ok := res.Content[len(res.Content)-1].(*sdk.TextContent)
		if !ok {
			t.Fatalf("%s: last block is %T, not text", name, res.Content[len(res.Content)-1])
		}
		var fromText, structured map[string]any
		if err := json.Unmarshal([]byte(text.Text), &fromText); err != nil {
			t.Fatalf("%s: data block is not JSON: %v\n%s", name, err, text.Text)
		}
		b, _ := json.Marshal(res.StructuredContent)
		_ = json.Unmarshal(b, &structured)
		if !reflect.DeepEqual(fromText, structured) {
			t.Errorf("%s: text does not round-trip to structured content\ntext: %s\nstructured: %s", name, text.Text, b)
		}
		return structured
	}

	pid := id(call("particular_define", map[string]any{"label": "Project X"}), "particular")
	call("particular_define", map[string]any{"label": "Project Y"})
	call("particular_resolve", map[string]any{"query": "Project X"})
	a := id(call("claim_assert", map[string]any{"evidential": "observed", "particular_id": pid, "content": "Uses Postgres", "context": map[string]any{"topics": []string{"db"}}}), "claim")
	b := id(call("claim_assert", map[string]any{"evidential": "observed", "particular_id": pid, "content": "Uses MySQL"}), "claim")
	s := id(call("synthesis_create", map[string]any{"particular_id": pid, "content": "Postgres, MySQL retired", "unresolved": "None identified",
		"inputs": []map[string]any{{"id": a, "role": "thesis"}, {"id": b, "role": "antithesis"}}}), "synthesis")
	call("knowledge_recall", map[string]any{"particular_id": pid})
	call("knowledge_recall", map[string]any{"query": "Postgres"})
	call("conflict_detect", map[string]any{"particular_id": pid})
	call("lineage_trace", map[string]any{"claim_id": s})
	call("topics_list", map[string]any{})
	call("unresolved_list", map[string]any{"include_none": true})
	call("workspace_status", map[string]any{})
	call("knowledge_publish", map[string]any{"claim_ids": []string{a}, "scope": "organisation"})
	call("particular_merge", map[string]any{"uri_a": "Project X", "uri_b": "Project Y"})
	call("claim_retract", map[string]any{"claim_id": b, "reason": "MySQL was never used"})

	tools, err := h.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if !called[tool.Name] {
			t.Errorf("tool %s is not exercised; add it so its text content is checked", tool.Name)
		}
	}
}
