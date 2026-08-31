// Package mcp is the Model Context Protocol front-end: one stdio server bound
// to one workspace, exposing the DKF specification's tools (spec names, spec
// parameters) plus two clearly labelled extensions. Results are the same values
// the CLI emits for --json, so there is a single documented contract.
package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/particulars-cli/internal/apperr"
	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/prov"
	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
	skill "github.com/nodelogicau/particulars-cli/skills/particulars"
)

// Options configure a server.
type Options struct {
	Workspace *store.Workspace
	Version   string
	// Explicit provenance defaults from server flags (optional).
	Author, Harness, Model string
}

// Server wraps an SDK server bound to a workspace.
type Server struct {
	ws   *store.Workspace
	opts Options
	mu   sync.Mutex // serialises every mutating tool: load → write → index
	srv  *sdk.Server
}

// PromptName is the prompt that carries the same text as instructions.
const PromptName = "particulars-discipline"

// New builds a server for ws. Tools, prompt, and instructions are registered;
// call Run to serve.
func New(o Options) *Server {
	s := &Server{ws: o.Workspace, opts: o}
	s.srv = sdk.NewServer(&sdk.Implementation{Name: "particulars", Version: skill.NormaliseVersion(o.Version)}, &sdk.ServerOptions{Instructions: s.instructions()})
	s.registerTools()
	s.srv.AddPrompt(&sdk.Prompt{Name: PromptName, Description: "The particulars discipline: recall before you assert, evidence on every claim, honest unresolved on every synthesis."}, func(ctx context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
		return &sdk.GetPromptResult{Description: "particulars discipline", Messages: []*sdk.PromptMessage{{Role: "user", Content: &sdk.TextContent{Text: s.instructions()}}}}, nil
	})
	return s
}

// MCP exposes the underlying SDK server (for transports other than Run).
func (s *Server) MCP() *sdk.Server { return s.srv }

// Run serves a single session over t until the client disconnects.
func (s *Server) Run(ctx context.Context, t sdk.Transport) error { return s.srv.Run(ctx, t) }

// Instructions is the text sent at initialize and via the prompt.
func (s *Server) Instructions() string { return s.instructions() }

func (s *Server) instructions() string {
	var b strings.Builder
	fmt.Fprintf(&b, "This particulars server is bound to the DKF workspace at %s (id %s). Everything you write lands as YAML files there for a human to review — typically through a git pull request; nothing is committed for you.\n\n", s.ws.Root, s.ws.Config.Workspace.ID)
	b.WriteString("Tool names follow the DKF specification (particular_*, claim_*, synthesis_create, knowledge_recall, conflict_detect, lineage_trace, knowledge_publish); topics_list and workspace_status are particulars extensions.\n")
	b.Write(skill.Body())
	return b.String()
}

// --- shared plumbing ----------------------------------------------------

// sourceIn is the spec's source block as tool input.
type sourceIn struct {
	Author  string `json:"author,omitempty" jsonschema:"a person — a particular id, URI, or name: you or the human you work for; pass it only when it differs from the workspace default"`
	Harness string `json:"harness,omitempty" jsonschema:"the AI harness, if one was involved (defaults to the connected client's name)"`
	Model   string `json:"model,omitempty" jsonschema:"the model, if known"`
	// Document is a union: a bare reference, or a mapping of uri/hash/quote.
	// It is typed as any because the schema the SDK infers from a struct would
	// reject the string form, and a bare reference must stay valid — it is not
	// inferior provenance.
	Document any `json:"document,omitempty" jsonschema:"what was read to make the assertion: a path, URL, or command as a string — or an object with ref (uri is the legacy alias), optional author (who produced what was read: a particular id, URI, or name — for testimony, who told you), optional hash (sha256:…), and optional quote (the sentence that supports the claim, verbatim)"`
}

// documentFrom accepts the string or the mapping form of a document. The
// mapping is keyed by ref, with uri accepted as the legacy alias; an unknown
// key is refused rather than silently dropped, since a caller who sent it
// meant something by it.
func documentFrom(v any) (dkf.Document, error) {
	switch t := v.(type) {
	case nil:
		return dkf.Document{}, nil
	case string:
		return dkf.Document{Ref: t}, nil
	case map[string]any:
		d := dkf.Document{}
		var ref, uri string
		targets := map[string]*string{"ref": &ref, "uri": &uri, "author": &d.Author, "hash": &d.Hash, "quote": &d.Quote}
		for key, raw := range t {
			target, ok := targets[key]
			if !ok {
				return d, apperr.Usage("source.document has no field %q: the mapping takes ref, author, hash, and quote", key)
			}
			sv, ok := raw.(string)
			if !ok {
				return d, apperr.Usage("source.document.%s must be a string", key)
			}
			*target = sv
		}
		if ref != "" && uri != "" {
			return d, apperr.Usage("source.document: pass ref, not both ref and its legacy alias uri")
		}
		d.Ref = ref
		if d.Ref == "" {
			d.Ref = uri
		}
		return d, nil
	}
	return dkf.Document{}, apperr.Usage("source.document must be a string or an object with ref, author, hash, and quote")
}

// contextIn is the spec's context block as tool input.
type contextIn struct {
	Scope  string   `json:"scope,omitempty" jsonschema:"personal | organisation | public (default: workspace default)"`
	Topics []string `json:"topics,omitempty" jsonschema:"stable lowercase tags used by knowledge_recall"`
}

// source resolves provenance for a call — explicit → env → dkf.yaml → client
// name — then resolves author references for writing: a defined particular is
// written as its uri. A per-call author is explicit (an ambiguous name is
// refused with candidates); the server flag and workspace default fall
// through unchanged, noted on stderr, because failing them would block every
// write of the session.
func (s *Server) source(g *store.Graph, req clientInfoer, in *sourceIn, needHarness bool) (dkf.Source, error) {
	e := prov.Explicit{Author: s.opts.Author, Harness: s.opts.Harness, Model: s.opts.Model}
	if in != nil {
		if in.Author != "" {
			e.Author = in.Author
		}
		if in.Harness != "" {
			e.Harness = in.Harness
		}
		if in.Model != "" {
			e.Model = in.Model
		}
		doc, derr := documentFrom(in.Document)
		if derr != nil {
			return dkf.Source{}, derr
		}
		e.Document, e.DocumentAuthor, e.DocumentHash, e.Quote = doc.Ref, doc.Author, doc.Hash, doc.Quote
	}
	fallback := ""
	if ci := req.ClientInfo(); ci != nil {
		fallback = ci.Name
	}
	src := prov.Resolve(s.ws.Config.Defaults.Source, e, fallback)
	if err := prov.Require(src, needHarness); err != nil {
		return dkf.Source{}, err
	}
	explicit := in != nil && strings.TrimSpace(in.Author) != ""
	written, ambiguous, err := query.ResolveAuthorForWrite(g, src.Author, explicit, "source.author")
	if err != nil {
		return dkf.Source{}, err
	}
	if len(ambiguous) > 0 {
		fmt.Fprintf(os.Stderr, "note: author %q matches %s and is written unchanged; add an alias or merge, or set the default to a URI\n", src.Author, strings.Join(ambiguous, ", "))
	}
	src.Author = written
	if src.Document.Author != "" {
		written, _, err := query.ResolveAuthorForWrite(g, src.Document.Author, true, "source.document.author")
		if err != nil {
			return dkf.Source{}, err
		}
		src.Document.Author = written
	}
	return src, nil
}

type clientInfoer interface{ ClientInfo() *sdk.Implementation }

// errResult turns a domain error into an isError tool result carrying the
// CLI's error code, so clients see the same vocabulary as --json users.
func errResult(err error) *sdk.CallToolResult {
	ae := apperr.Classify(err)
	return &sdk.CallToolResult{
		IsError:           true,
		Content:           []sdk.Content{&sdk.TextContent{Text: ae.ErrCode + ": " + ae.Err.Error()}},
		StructuredContent: map[string]any{"error": map[string]any{"code": ae.ErrCode, "message": ae.Err.Error()}},
	}
}

// okResult pairs a one-line text summary with the structured value.
func okResult(summary string) *sdk.CallToolResult {
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: summary}}}
}

func boolp(b bool) *bool { return &b }

var (
	readOnly   = &sdk.ToolAnnotations{ReadOnlyHint: true}
	additive   = &sdk.ToolAnnotations{DestructiveHint: boolp(false)}
	idempotent = &sdk.ToolAnnotations{DestructiveHint: boolp(false), IdempotentHint: true}
)
