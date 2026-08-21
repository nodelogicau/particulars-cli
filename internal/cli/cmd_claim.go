package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

func (a *app) claimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim",
		Short: "Assert and retract claims",
	}
	cmd.AddCommand(a.claimAssertCmd(), a.buildRetractCmd(true))
	return cmd
}

func (a *app) claimAssertCmd() *cobra.Command {
	var subject, content, contentFile, scope, confidence, timestamp string
	var topics []string
	var prov provenanceFlags
	cmd := &cobra.Command{
		Use:   "assert --subject <particular> (--content <text> | --content-file <path|->) [flags]",
		Short: "Record a new claim about a particular",
		Args:  cobra.NoArgs,
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(subject) == "" {
				return usageErr("--subject is required")
			}
			text, err := a.readContent(content, contentFile)
			if err != nil {
				return err
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
			if err := requireProvenance(src, false); err != nil {
				return err
			}
			for i, t := range topics {
				topics[i] = strings.TrimSpace(t)
				if topics[i] == "" {
					return usageErr("--topic values must be non-empty")
				}
			}
			c := &dkf.Claim{
				ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: text, Source: src,
				Context: dkf.Context{Scope: sc, Topics: topics}, Timestamp: ts, Confidence: conf,
			}
			if err := ws.Create(c); err != nil {
				return err
			}
			if err := ws.UpsertIndex(c); err != nil {
				return err
			}
			path, _ := ws.Path(c.ID)
			return a.emit(map[string]any{"claim": c, "path": ws.Rel(path)}, func(w io.Writer) {
				fmt.Fprintf(w, "Asserted %s about %s (%s)\n  %s\n", c.ID, p.Label, p.ID, ws.Rel(path))
			})
		}),
	}
	cmd.Flags().StringVar(&subject, "subject", "", "particular id, uri, label, or alias (required)")
	cmd.Flags().StringVar(&content, "content", "", "claim text")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "read claim text from a file, or '-' for piped stdin")
	cmd.Flags().StringVar(&prov.author, "author", "", "source.author (default: $DKF_AUTHOR, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.harness, "harness", "", "source.harness (default: $DKF_HARNESS, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.model, "model", "", "source.model (default: $DKF_MODEL, then dkf.yaml)")
	cmd.Flags().StringVar(&prov.document, "document", "", "source.document: URI or path of the evidence")
	cmd.Flags().StringVar(&scope, "scope", "", "personal|organisation|public (default: dkf.yaml defaults.scope)")
	cmd.Flags().StringArrayVar(&topics, "topic", nil, "topic tag (repeatable)")
	cmd.Flags().StringVar(&confidence, "confidence", "", "confidence in [0, 1]")
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "assertion time, RFC 3339 (default: now)")
	return cmd
}
