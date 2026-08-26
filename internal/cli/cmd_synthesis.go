package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
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
input to later syntheses. Its source must name the harness that produced it.

--unresolved is mandatory. State what could not be reconciled; when nothing is
outstanding use the exact conventional value "None identified", so tooling can
tell "considered and empty" from "forgotten".

The current synthesis for a particular is the most recent by --timestamp (then
id), so a backdated synthesis does not displace a newer one.`,
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
			if err := requireProvenance(src, true); err != nil {
				return err
			}
			if method == "" {
				method = dkf.DefaultMethod
			}
			if !dkf.ValidMethod(method) {
				return usageErr("invalid --method %q: reconciliation (the inputs disagreed about a fact), qualification (each true in a different context), or positions (no evidence settles this)", method)
			}
			s := &dkf.Synthesis{
				ID: dkf.NewID(dkf.TypeSynthesis), Subject: p.ID, Content: text, Inputs: parsed, Unresolved: unresolved,
				Source: src, Method: method, Timestamp: ts,
				Context: dkf.Context{Scope: sc, Topics: topics}, Confidence: conf,
			}
			if err := ws.Create(s); err != nil {
				return err
			}
			if err := ws.UpsertIndex(s); err != nil {
				return err
			}
			// The spec evaluates scope_wider_than_inputs at create time as well
			// as during validate — reported, never blocking: a wider synthesis
			// is permitted, and refusing it would push authors toward writing
			// standalone claims with no inputs, losing the lineage.
			if g2, err := loadGraph(ws); err == nil {
				if msg := query.ScopeWiderThanInputs(g2, s); msg != "" {
					warnings = append(warnings, msg)
				}
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
	cmd.Flags().StringVar(&unresolved, "unresolved", "", "what could not be reconciled (required; \"None identified\" when nothing)")
	cmd.Flags().StringVar(&method, "method", "", "reconciliation|qualification|positions (default "+dkf.DefaultMethod+")")
	cmd.Flags().StringVar(&prov.author, "author", "", "source.author (default: $DKF_AUTHOR, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.harness, "harness", "", "source.harness, required (default: $DKF_HARNESS, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.model, "model", "", "source.model (default: $DKF_MODEL, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.document, "document", "", "source.document: URI or path of the evidence")
	cmd.Flags().StringVar(&scope, "scope", "", "personal|organisation|public (default: dkf.yaml defaults.scope)")
	cmd.Flags().StringArrayVar(&topics, "topic", nil, "topic tag (repeatable)")
	cmd.Flags().StringVar(&confidence, "confidence", "", "confidence in [0, 1]")
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "RFC 3339 time (default: now)")
	return cmd
}
