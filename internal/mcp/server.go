// Package mcp is the Model Context Protocol front-end: one stdio server bound
// to one workspace, exposing the DKF specification's tools (spec names, spec
// parameters) plus two clearly labelled extensions. Results are the same values
// the CLI emits for --json, so there is a single documented contract.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
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
	s.srv.AddReceivingMiddleware(withDataText)
	s.registerTools()
	s.registerConventionsResource()
	s.srv.AddPrompt(&sdk.Prompt{Name: PromptName, Description: "The particulars discipline: recall before you assert, evidence on every claim, honest unresolved on every synthesis."}, func(ctx context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
		return &sdk.GetPromptResult{Description: "particulars discipline", Messages: []*sdk.PromptMessage{{Role: "user", Content: &sdk.TextContent{Text: s.instructions()}}}}, nil
	})
	return s
}

// registerConventionsResource lists the workspace's conventions document as
// a resource when one is readable at startup — the channel for clients that
// never surface instructions. The content is read at request time, so a
// client attaching it after an edit sees the edit; instructions, like the
// workspace binding, are a startup snapshot.
func (s *Server) registerConventionsResource() {
	rel, _, err := s.ws.Conventions()
	if rel == "" || err != nil {
		return
	}
	abs := filepath.Join(s.ws.Root, filepath.FromSlash(rel))
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // a Windows volume path still needs the file:/// form
	}
	uri := (&url.URL{Scheme: "file", Path: p}).String()
	s.srv.AddResource(&sdk.Resource{
		URI: uri, Name: rel, Title: "Workspace conventions", MIMEType: "text/markdown",
		Description: "This workspace's own conventions for agents — topic vocabulary, ingestion rules, scopes — delivered with the server instructions and readable here in full.",
	}, func(ctx context.Context, req *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{URI: uri, MIMEType: "text/markdown", Text: string(data)}}}, nil
	})
}

// MCP exposes the underlying SDK server (for transports other than Run).
func (s *Server) MCP() *sdk.Server { return s.srv }

// Run serves a single session over t until the client disconnects.
func (s *Server) Run(ctx context.Context, t sdk.Transport) error { return s.srv.Run(ctx, t) }

// cliTools maps each command the skill names to the tool that serves it, or
// to none. The skill is written for the command line and shared with every
// harness, so over MCP it is translated rather than rewritten: without this a
// model calls the commands as tools — `particulars`, `particulars_register` —
// and gets an unknown-tool error for each (#16). A test holds the tools named
// here equal to the tools registered.
var cliTools = []struct {
	CLI        []string
	Tool, Note string
}{
	{[]string{"particular resolve"}, "particular_resolve", ""},
	{[]string{"particular define"}, "particular_define", ""},
	{[]string{"particular merge"}, "particular_merge", ""},
	{[]string{"claim assert"}, "claim_assert", ""},
	{[]string{"synthesis create"}, "synthesis_create", ""},
	{[]string{"recall"}, "knowledge_recall", "by particular, query, author, or topic"},
	{[]string{"conflicts"}, "conflict_detect", ""},
	{[]string{"unresolved"}, "unresolved_list", ""},
	{[]string{"lineage"}, "lineage_trace", ""},
	{[]string{"publish"}, "knowledge_publish", ""},
	{[]string{"retract", "claim retract"}, "claim_retract", ""},
	{[]string{"topics"}, "topics_list", ""},
	{[]string{"workspace", "validate"}, "workspace_status", "carries the validate summary"},
	{[]string{"register"}, "", "no call needed: the register is included below"},
	{[]string{"export", "index", "init", "skill install"}, "", "no MCP equivalent: the human runs these"},
}

// mcpPreamble tells a model reading the skill over MCP how its commands map
// to tools. It sits before the skill so the translation is read first.
func mcpPreamble() string {
	var b strings.Builder
	b.WriteString("## Using this skill over MCP\n\n")
	b.WriteString("The skill below is written for the `particulars` command line. Over MCP there is no shell and no `particulars` tool: each command it names is one of this server's tools, listed here. Ignore `--json` — every tool result is already structured. Flags become parameters; for `claim assert`:\n\n")
	b.WriteString("- `--subject` → `particular_id`\n")
	b.WriteString("- `--document` → `source.document` (a string, or an object with `ref`)\n")
	b.WriteString("- `--document-author`, `--quote`, `--document-hash` → `source.document.author`, `.quote`, `.hash`\n")
	b.WriteString("- `--topic` → `context.topics`\n")
	b.WriteString("- `--hash-document` has no MCP equivalent: pass `hash` yourself, or omit it\n\n")
	b.WriteString("| Command in the skill | Tool |\n|---|---|\n")
	for _, m := range cliTools {
		tool := "`" + m.Tool + "`"
		if m.Tool == "" {
			tool = "—"
		}
		if m.Note != "" {
			tool += " (" + m.Note + ")"
		}
		cmds := make([]string, len(m.CLI))
		for i, c := range m.CLI {
			cmds[i] = "`particulars " + c + "`"
		}
		fmt.Fprintf(&b, "| %s | %s |\n", strings.Join(cmds, ", "), tool)
	}
	b.WriteString("\nTool names follow the DKF specification; topics_list, unresolved_list, and workspace_status are particulars extensions.\n\n")
	return b.String()
}

// Instructions is the text sent at initialize and via the prompt.
func (s *Server) Instructions() string { return s.instructions() }

func (s *Server) instructions() string {
	var b strings.Builder
	fmt.Fprintf(&b, "This particulars server is bound to the DKF workspace at %s (id %s). Everything you write lands as YAML files there for a human to review — typically through a git pull request; nothing is committed for you.\n\n", s.ws.Root, s.ws.Config.Workspace.ID)
	b.WriteString(mcpPreamble())
	b.Write(skill.Body())
	// What this workspace already uses, generated from the files. It sits
	// between the skill and the conventions because the skill states the
	// general discipline, this states the facts, and the human's prose
	// comments on the facts — so the prose is read last and wins.
	if g, err := s.ws.Load(); err == nil {
		b.WriteString("\n\n## This workspace's particulars and topics\n\n")
		b.WriteString(query.BuildRegister(g).Render(query.RegisterCap))
	}
	// The workspace's own conventions — topic vocabulary, local naming —
	// travel with the instructions, because an MCP-only client never reads
	// the repository. Read once, at startup, like the workspace binding.
	if rel, content, err := s.ws.Conventions(); err == nil && len(content) > 0 {
		b.WriteString("\n\n## Workspace conventions (" + rel + ")\n\n")
		// The spec's 16 KiB is a floor, and the cut must land on a
		// character boundary; both hold only if a character straddling the
		// mark is included, so the cut advances to the next rune start.
		const maxConventions = 16 * 1024
		if len(content) > maxConventions {
			cut := maxConventions
			for cut < len(content) && !utf8.RuneStart(content[cut]) {
				cut++
			}
			b.Write(content[:cut])
			b.WriteString("\n\n[truncated — read " + rel + " in the workspace for the rest]")
		} else {
			b.Write(content)
		}
	}
	return b.String()
}

// --- shared plumbing ----------------------------------------------------

// sourceIn is the spec's source block as tool input.
type sourceIn struct {
	Author  string `json:"author,omitempty" jsonschema:"a person — a particular id, URI, or name: you or the human you work for; pass it only when it differs from the workspace default"`
	Harness string `json:"harness,omitempty" jsonschema:"the AI harness, if one was involved (defaults to the connected client's name)"`
	Model   string `json:"model,omitempty" jsonschema:"the model, if known"`
	// Document is a union: a bare reference, or a mapping of uri/hash/quote.
	// It is an empty interface because the schema the SDK infers from a struct
	// would reject the string form, and a bare reference must stay valid — it
	// is not inferior provenance. Its schema is documentSchema, so the shape
	// is stated where a model reads it rather than only in this description.
	Document documentIn `json:"document,omitempty" jsonschema:"what was read to make the assertion: a path, URL, or command as a string — or an object with ref (uri is the legacy alias), optional author (who produced what was read: a particular id, URI, or name — for testimony, who told you), optional hash (sha256:…), and optional quote (the sentence that supports the claim, verbatim)"`
	// Quote and Hash are shorthand for the document's fields, as scope is
	// for context.scope: the flat shape is the one a model reaches for first,
	// because the CLI spells them as flags beside --document.
	Quote string `json:"quote,omitempty" jsonschema:"shorthand for source.document's quote; needs source.document"`
	Hash  string `json:"hash,omitempty" jsonschema:"shorthand for source.document's hash; needs source.document"`
	// DocumentAuthor is shorthand for the document's author, spelled as the
	// CLI's --document-author; plain author already names the asserter. The
	// hyphenated key is accepted because models send it as often as not.
	DocumentAuthor       string `json:"document_author,omitempty" jsonschema:"shorthand for source.document's author (who produced what was read; for testimony, who told you); needs source.document"`
	DocumentAuthorHyphen string `json:"document-author,omitempty" jsonschema:"the same as document_author"`
}

// documentIn is source.document as decoded: a string or a mapping. It is
// named so that inferring a tool's schema can give it documentSchema.
type documentIn interface{}

// documentSchema states both forms of source.document. The object form
// leaves additional properties open on purpose: an unknown key then reaches
// documentFrom, whose refusal names the key and the fields allowed, rather
// than failing the SDK's anyOf with a message that names neither.
func documentSchema() *jsonschema.Schema {
	str := func(d string) *jsonschema.Schema { return &jsonschema.Schema{Type: "string", Description: d} }
	return &jsonschema.Schema{AnyOf: []*jsonschema.Schema{
		{Type: "string"},
		{Type: "object", Properties: map[string]*jsonschema.Schema{
			"ref":    str("a path, URL, or command"),
			"uri":    str("legacy alias for ref"),
			"author": str("who produced what was read: a particular id, URI, or name"),
			"hash":   str("sha256:… of the document as read"),
			"quote":  str("the sentence that supports the claim, verbatim"),
		}},
	}}
}

// withSource infers the input schema of a tool whose input carries a source
// block, giving source.document its stated shape.
func withSource[In any]() *jsonschema.Schema {
	s, err := jsonschema.For[In](&jsonschema.ForOptions{TypeSchemas: map[reflect.Type]*jsonschema.Schema{reflect.TypeFor[documentIn](): documentSchema()}})
	if err != nil {
		panic(err)
	}
	return s
}

// withShorthand folds source.quote, source.hash and source.document_author
// into doc. Setting a field both ways with different values is refused rather
// than one silently winning, since a quote is evidence and the losing one
// would be gone; without a per-call document there is nothing for the quote
// to be from.
func withShorthand(in *sourceIn, doc dkf.Document) (dkf.Document, error) {
	author := in.DocumentAuthor
	if in.DocumentAuthorHyphen != "" {
		if author != "" && author != in.DocumentAuthorHyphen {
			return doc, apperr.Usage("pass document author once: as source.document_author or source.document-author, not both")
		}
		author = in.DocumentAuthorHyphen
	}
	for _, f := range []struct {
		name, key, field, short string
		into                    *string
	}{
		{"quote", "quote", "quote", in.Quote, &doc.Quote},
		{"hash", "hash", "hash", in.Hash, &doc.Hash},
		{"document author", "document_author", "author", author, &doc.Author},
	} {
		if f.short == "" {
			continue
		}
		if in.Document == nil {
			return doc, apperr.Usage("source.%s needs source.document: pass {\"document\": {\"ref\": …, \"%s\": …}}", f.key, f.field)
		}
		// The same value given both ways loses nothing; only a second,
		// different value would be dropped, so only that is refused.
		if *f.into != "" && *f.into != f.short {
			return doc, apperr.Usage("pass %s once: in source.document or as source.%s, not both", f.name, f.key)
		}
		*f.into = f.short
	}
	if doc.Quote != "" && !doc.HasQuote() {
		return doc, apperr.Usage("source.quote is blank; omit it rather than recording nothing")
	}
	return doc, nil
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
		if d.Quote != "" && !d.HasQuote() {
			// Refused on write, as a confidence outside [0, 1] is: a reader
			// would have to treat it as no quote and warn forever.
			return d, apperr.Usage("source.document.quote is blank; omit it rather than recording nothing")
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
		if derr == nil {
			doc, derr = withShorthand(in, doc)
		}
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

// withDataText appends a successful tool result's structured content to its
// text as JSON. Clients such as Claude Desktop hand the model only the text,
// so without it the model sees the summary ("3 topics") and none of the data.
func withDataText(next sdk.MethodHandler) sdk.MethodHandler {
	return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
		res, err := next(ctx, method, req)
		if r, ok := res.(*sdk.CallToolResult); ok && err == nil && !r.IsError && r.StructuredContent != nil {
			var b bytes.Buffer
			enc := json.NewEncoder(&b)
			enc.SetEscapeHTML(false)
			if enc.Encode(r.StructuredContent) == nil {
				r.Content = append(r.Content, &sdk.TextContent{Text: strings.TrimSpace(b.String())})
			}
		}
		return res, err
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
