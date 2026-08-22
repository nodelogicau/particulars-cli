package prov

import (
	"testing"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

func TestUnsubstitutedPlaceholdersAreAbsent(t *testing.T) {
	for _, v := range []string{"${user_config.author}", "  ${user_config.author}  ", "${USER}", "${}"} {
		if !IsPlaceholder(v) {
			t.Errorf("%q should be a placeholder", v)
		}
	}
	for _, v := range []string{"ben", "$USER", "${a} and ${b}", "a${b}", "${b}c", "", "$"} {
		if IsPlaceholder(v) {
			t.Errorf("%q should not be a placeholder", v)
		}
	}

	// A placeholder flag falls through to the next layer rather than being recorded.
	defaults := dkf.Source{Author: "configured", Harness: "claude"}
	got := Resolve(defaults, Explicit{Author: "${user_config.author}"}, "")
	if got.Author != "configured" {
		t.Errorf("placeholder should fall through to dkf.yaml: %q", got.Author)
	}
	// With nothing behind it, the author is simply absent — harness alone is a valid source.
	got = Resolve(dkf.Source{Harness: "claude"}, Explicit{Author: "${user_config.author}"}, "")
	if got.Author != "" {
		t.Errorf("placeholder with no fallback should be empty: %q", got.Author)
	}
	if err := Require(got, false); err != nil {
		t.Errorf("harness alone must satisfy the minimum: %v", err)
	}
	// The env layer is filtered too.
	t.Setenv(EnvAuthor, "${user_config.author}")
	if got := Resolve(dkf.Source{Author: "configured"}, Explicit{}, ""); got.Author != "configured" {
		t.Errorf("placeholder in env should fall through: %q", got.Author)
	}
	t.Setenv(EnvAuthor, "")
	// Neither author nor harness: still the ordinary error, not a recorded placeholder.
	if err := Require(Resolve(dkf.Source{}, Explicit{Author: "${user_config.author}"}, ""), false); err == nil {
		t.Error("a placeholder must not satisfy the author-or-harness minimum")
	}
}
