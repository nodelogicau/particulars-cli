package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (a *app) synthesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "synthesis",
		Short: "Record syntheses that reconcile thesis and antithesis claims",
	}
	cmd.AddCommand(a.synthesisCreateCmd())
	return cmd
}

// parseInput parses id:role[:weight].
func parseInput(spec string) (dkf.Input, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return dkf.Input{}, usageErr("invalid --input %q: expected <id>:<thesis|antithesis>[:<primary|qualifying>]", spec)
	}
	in := dkf.Input{ID: strings.TrimSpace(parts[0]), Role: dkf.Role(strings.TrimSpace(parts[1])), Weight: dkf.WeightPrimary}
	if len(parts) == 3 {
		in.Weight = dkf.Weight(strings.TrimSpace(parts[2]))
	}
	if !dkf.IsValidID(in.ID) {
		return dkf.Input{}, usageErr("invalid --input %q: %q is not a valid id", spec, in.ID)
	}
	if !dkf.IsAssertionID(in.ID) {
		return dkf.Input{}, usageErr("invalid --input %q: inputs must be claims or syntheses, not particulars", spec)
	}
	if !dkf.ValidRole(in.Role) {
		return dkf.Input{}, usageErr("invalid --input %q: role must be thesis or antithesis", spec)
	}
	if !dkf.ValidWeight(in.Weight) {
		return dkf.Input{}, usageErr("invalid --input %q: weight must be primary or qualifying", spec)
	}
	return in, nil
}

func (a *app) synthesisCreateCmd() *cobra.Command {
	var subject, content, contentFile, unresolved, method, scope, confidence, timestamp string
	var inputs, topics []string
	var prov provenanceFlags
	cmd := &cobra.Command{
		Use:   "create --subject <particular> (--content <text> | --content-file <path|->) --input <id>:<role>[:<weight>]... --unresolved <text> [flags]",
		Short: "Record a synthesis the calling agent has already reasoned",
		Long: `A synthesis is itself a claim: it can be recalled, retracted, and cited as an
input to later syntheses. --unresolved is mandatory; state what could not be
reconciled, or say so explicitly (e.g. "None identified").`,
		Args: cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(subject) == "" {
				return usageErr("--subject is required")
			}
			if len(inputs) == 0 {
				return usageErr("at least one --input <id>:<role>[:<weight>] is required")
			}
			if strings.TrimSpace(unresolved) == "" {
				return usageErr("--unresolved is required; state what remains unreconciled, or \"None identified\"")
			}
			text, err := a.readContent(content, contentFile)
			if err != nil {
				return err
			}
			parsed := make([]dkf.Input, 0, len(inputs))
			for _, spec := range inputs {
				in, err := parseInput(spec)
				if err != nil {
					return err
				}
				parsed = append(parsed, in)
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			p, err := resolveSubject(g, subject)
			if err != nil {
				return err
			}
			var warnings []string
			for _, in := range parsed {
				child := g.Assertion(in.ID)
				if child == nil {
					return notFoundErr("input %s does not exist", in.ID)
				}
				if child.GetRetracted() != nil {
					warnings = append(warnings, fmt.Sprintf("input %s is retracted", in.ID))
				}
			}
			sc, err := resolveScope(ws, scope)
			if err != nil {
				return err
			}
			conf, err := parseConfidence(confidence)
			if err != nil {
				return err
			}
			ts, err := parseTimestamp(timestamp)
			if err != nil {
				return err
			}
			src := resolveSource(ws, prov)
			if src.Harness == "" {
				return usageErr("produced-by.harness is required: pass --harness, set %s, or configure defaults.source.harness in dkf.yaml", EnvHarness)
			}
			if method == "" {
				method = dkf.DefaultMethod
			}
			s := &dkf.Synthesis{
				ID: dkf.NewID(dkf.TypeSynthesis), Subject: p.ID, Content: text, Inputs: parsed, Unresolved: unresolved,
				ProducedBy: dkf.ProducedBy{Harness: src.Harness, Model: src.Model}, Method: method, Timestamp: ts,
				Context: dkf.Context{Scope: sc, Topics: topics}, Confidence: conf,
			}
			if err := ws.Create(s); err != nil {
				return err
			}
			if err := ws.UpsertIndex(s); err != nil {
				return err
			}
			if warnings == nil {
				warnings = []string{}
			}
			path, _ := ws.Path(s.ID)
			return a.emit(map[string]any{"synthesis": s, "path": ws.Rel(path), "warnings": warnings}, func(w io.Writer) {
				fmt.Fprintf(w, "Synthesised %s about %s (%s) from %s\n  %s\n", s.ID, p.Label, p.ID, plural(len(parsed), "input"), ws.Rel(path))
				for _, wmsg := range warnings {
					fmt.Fprintf(w, "  warning: %s\n", wmsg)
				}
			})
		}),
	}
	cmd.Flags().StringVar(&subject, "subject", "", "particular id, uri, label, or alias (required)")
	cmd.Flags().StringVar(&content, "content", "", "synthesis text")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "read synthesis text from a file, or '-' for piped stdin")
	cmd.Flags().StringArrayVar(&inputs, "input", nil, "<id>:<thesis|antithesis>[:<primary|qualifying>] (repeatable, required)")
	cmd.Flags().StringVar(&unresolved, "unresolved", "", "what could not be reconciled (required)")
	cmd.Flags().StringVar(&method, "method", "", "synthesis method (default "+dkf.DefaultMethod+")")
	cmd.Flags().StringVar(&prov.harness, "harness", "", "produced-by.harness (default: $DKF_HARNESS, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.model, "model", "", "produced-by.model (default: $DKF_MODEL, then dkf.yaml)")
	cmd.Flags().StringVar(&scope, "scope", "", "personal|organisation|public (default: dkf.yaml defaults.scope)")
	cmd.Flags().StringArrayVar(&topics, "topic", nil, "topic tag (repeatable)")
	cmd.Flags().StringVar(&confidence, "confidence", "", "confidence in [0, 1]")
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "RFC 3339 time (default: now)")
	_ = store.ErrNotFound
	return cmd
}
