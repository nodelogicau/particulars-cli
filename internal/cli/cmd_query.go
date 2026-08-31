package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
)

func (a *app) recallCmd() *cobra.Command {
	var includeRetracted bool
	var limit int
	var scope, author string
	var topics []string
	cmd := &cobra.Command{
		Use:   "recall [<particular>] [--author <who>] [--topic <t>]... [--scope <s>] [--include-retracted] [--limit <n>]",
		Short: "Retrieve claims and syntheses in lineage order",
		Long: `Returns claims and syntheses about a particular (by id, uri, label, or alias)
and/or carrying every given --topic, oldest first so that inputs precede the
syntheses that cite them. The most recent non-retracted synthesis about a
particular is marked current. --limit keeps the most recent N.`,
		Args: cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && len(topics) == 0 && author == "" {
				return usageErr("pass a particular, one or more --topic, --author, or a combination")
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			opts := query.RecallOptions{Topics: topics, IncludeRetracted: includeRetracted, Limit: limit}
			var subjectLabel string
			if len(args) == 1 {
				p, err := resolveSubject(g, args[0])
				if err != nil {
					return err
				}
				opts.Subject = p.ID
				subjectLabel = p.Label
			}
			if scope != "" {
				if !dkf.ValidScope(dkf.Scope(scope)) {
					return usageErr("invalid --scope %q", scope)
				}
				opts.Scope = dkf.Scope(scope)
			}
			if author != "" {
				p, err := resolveSubject(g, author)
				if err != nil {
					return err
				}
				opts.Author = p.ID
			}
			entries := query.Recall(g, opts)
			out := map[string]any{"entries": entries, "count": len(entries)}
			if opts.Author != "" {
				out["author"] = opts.Author
			}
			if opts.Subject != "" {
				out["subject"] = opts.Subject
				if class := g.ClassOf(opts.Subject); len(class) > 1 {
					out["class"] = class
				}
			}
			return a.emit(out, func(w io.Writer) {
				if len(entries) == 0 {
					fmt.Fprintln(w, "(nothing recalled)")
					return
				}
				if subjectLabel != "" {
					fmt.Fprintf(w, "%s (%s): %s\n", subjectLabel, opts.Subject, plural(len(entries), "entry"))
				}
				for _, e := range entries {
					flags := ""
					if e.Current {
						flags += " [current]"
					}
					if e.Unsynthesised {
						flags += " [unsynthesised]"
					}
					if e.Retracted {
						flags += " [retracted]"
					}
					for _, r := range e.Relations {
						flags += " [" + r + "]"
					}
					if opts.Subject != "" && e.Subject != opts.Subject {
						flags += " [subject " + e.Subject + "]"
					}
					fmt.Fprintf(w, "%s  %-9s %s  conf=%s%s\n    %s\n", e.ID, e.Type, e.Timestamp, fmtConfidence(e.Confidence), flags, oneLine(e.Content, 160))
					if e.Type == dkf.TypeSynthesis {
						ids := make([]string, len(e.Inputs))
						for i, in := range e.Inputs {
							ids[i] = fmt.Sprintf("%s(%s)", in.ID, in.Role)
						}
						fmt.Fprintf(w, "    inputs: %s\n    unresolved: %s\n", strings.Join(ids, ", "), oneLine(e.Unresolved, 120))
					}
				}
			})
		}),
	}
	cmd.Flags().BoolVar(&includeRetracted, "include-retracted", false, "include retracted claims and syntheses")
	cmd.Flags().IntVar(&limit, "limit", 0, "keep only the most recent N entries")
	cmd.Flags().StringVar(&scope, "scope", "", "only entries with this scope")
	cmd.Flags().StringVar(&author, "author", "", "only entries asserted by or reported from this particular (id, uri, label, or alias)")
	cmd.Flags().StringArrayVar(&topics, "topic", nil, "only entries carrying this topic (repeatable; all must match)")
	return cmd
}

func (a *app) lineageCmd() *cobra.Command {
	var depth int
	cmd := &cobra.Command{
		Use:   "lineage <id> [--depth <n>]",
		Short: "Show the provenance tree of a claim or synthesis",
		Args:  cobra.ExactArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			if !dkf.IsValidID(args[0]) {
				return usageErr("%q is not a valid id", args[0])
			}
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			tree, err := query.Lineage(g, args[0], depth)
			if err != nil {
				return err
			}
			return a.emit(tree, func(w io.Writer) { printTree(w, tree, "", true) })
		}),
	}
	cmd.Flags().IntVar(&depth, "depth", 0, "maximum levels to expand (0 = unlimited)")
	return cmd
}

func printTree(w io.Writer, n *query.Node, prefix string, root bool) {
	label := n.ID
	if n.Role != "" {
		label = fmt.Sprintf("%s (%s", n.ID, n.Role)
		if n.Weight != "" && n.Weight != dkf.WeightPrimary {
			label += ", " + string(n.Weight)
		}
		label += ")"
	}
	var marks []string
	if n.Retracted {
		if n.SupersededBy != "" {
			marks = append(marks, "retracted → "+n.SupersededBy)
		} else {
			marks = append(marks, "retracted")
		}
	}
	if n.Missing {
		marks = append(marks, "missing")
	}
	if n.Cycle {
		marks = append(marks, "cycle")
	}
	if n.Truncated {
		marks = append(marks, "…")
	}
	if len(marks) > 0 {
		label += " [" + strings.Join(marks, ", ") + "]"
	}
	if !n.Missing && !n.Cycle {
		label += "  " + oneLine(n.Content, 100)
	}
	fmt.Fprintf(w, "%s%s\n", prefix, label)
	childPrefix := prefix
	if !root {
		childPrefix = prefix
	}
	for i, c := range n.Inputs {
		connector := "├── "
		next := childPrefix + "│   "
		if i == len(n.Inputs)-1 {
			connector = "└── "
			next = childPrefix + "    "
		}
		printTreeChild(w, c, childPrefix+connector, next)
	}
}

func printTreeChild(w io.Writer, n *query.Node, linePrefix, childPrefix string) {
	var sb strings.Builder
	printTree(&sb, n, "", true)
	lines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	fmt.Fprintf(w, "%s%s\n", linePrefix, lines[0])
	for _, l := range lines[1:] {
		fmt.Fprintf(w, "%s%s\n", childPrefix, l)
	}
}

func (a *app) conflictsCmd() *cobra.Command {
	var failOn bool
	cmd := &cobra.Command{
		Use:   "conflicts [<particular>] [--fail-on-conflicts]",
		Short: "Report unsynthesised claims and stale syntheses (structural, not semantic)",
		Long: `For each particular: current is the most recent non-retracted synthesis;
unsynthesised are non-retracted claims and syntheses not reconciled into it;
stale are syntheses citing a retracted input. The tool does not judge whether
contents contradict — that is the agent's job.`,
		Args: cobra.MaximumNArgs(1),
		RunE: a.run(func(cmd *cobra.Command, args []string) error {
			ws, err := a.openWorkspace()
			if err != nil {
				return err
			}
			g, err := loadGraph(ws)
			if err != nil {
				return err
			}
			subject := ""
			if len(args) == 1 {
				p, err := resolveSubject(g, args[0])
				if err != nil {
					return err
				}
				subject = p.ID
			}
			reports := query.Conflicts(g, subject)
			if err := a.emit(map[string]any{"reports": reports, "count": len(reports)}, func(w io.Writer) {
				if len(reports) == 0 {
					fmt.Fprintln(w, "No conflicts.")
					return
				}
				for _, r := range reports {
					fmt.Fprintf(w, "%s  %s  priority=%d\n", r.Particular, r.Label, r.Priority)
					if len(r.Members) > 1 {
						fmt.Fprintf(w, "  members:       %s\n", strings.Join(r.Members, ", "))
					}
					if r.Current != "" {
						fmt.Fprintf(w, "  current:       %s\n", r.Current)
					} else {
						fmt.Fprintf(w, "  current:       (none)\n")
					}
					if len(r.Unsynthesised) > 0 {
						fmt.Fprintf(w, "  unsynthesised: %s\n", strings.Join(r.Unsynthesised, ", "))
					}
					if len(r.Stale) > 0 {
						fmt.Fprintf(w, "  stale:         %s\n", strings.Join(r.Stale, ", "))
					}
				}
			}); err != nil {
				return err
			}
			if failOn && len(reports) > 0 {
				return checkFailedErr("%s reported", plural(len(reports), "conflict"))
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&failOn, "fail-on-conflicts", false, "exit 4 if any particular is reported (for CI)")
	return cmd
}
