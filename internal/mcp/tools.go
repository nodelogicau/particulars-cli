package mcp

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/particulars-cli/internal/apperr"
	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

func (s *Server) registerTools() {
	sdk.AddTool(s.srv, &sdk.Tool{Name: "particular_define", Annotations: idempotent,
		Description: "Create or update a particular (a specific, identifiable thing claims are about). A particular is a thing in the world — a person, project, place, system — never the document or feed being read; what you read belongs in claim_assert's source.document. Idempotent on URI: without uri one is minted from the label (<base-uri><slug> or urn:dkf:<workspace-id>:<slug>), so the same label always resolves to the same particular. Prefer an existing global URI (Wikidata, an ORCID, a person's or project's GitHub page) when the thing has one."},
		s.particularDefine)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "particular_resolve", Annotations: readOnly,
		Description: "Find a particular by id, URI, label, or alias (label/alias case-insensitive). Returns particular: null when nothing matches. Call this before defining or asserting to avoid duplicates."},
		s.particularResolve)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "particular_merge", Annotations: additive,
		Description: "Declare that two URIs denote the same particular. Writes a merge record; nothing is rewritten. Merged particulars form one class for knowledge_recall and conflict_detect. Undo with claim_retract on the merge id."},
		s.particularMerge)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "claim_assert", Annotations: additive,
		Description: "Record one falsifiable statement about a particular, with evidence in source.document and calibrated confidence. The statement is about the thing in the world; what you read goes in source.document (who produced it in document.author) — content that names an article or URL is a citation, not a claim. Recall first (knowledge_recall) so you extend or reconcile rather than duplicate. Claims are immutable; correct them with claim_retract + a new claim, or a synthesis."},
		s.claimAssert)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "claim_retract", Annotations: additive,
		Description: "Append a retracted block (reason, source, optional superseded_by) to a claim, synthesis, merge, or promotion record. Retracting a promotion returns the objects it covered to the scope they would have had without it. Never deletes; provenance is preserved. Syntheses that cite a retracted input become stale until re-synthesised."},
		s.claimRetract)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "synthesis_create", Annotations: additive,
		Description: "Record a synthesis you have already reasoned: inputs with roles (thesis = the belief challenged, antithesis = the challenger; weight primary|qualifying), content carrying the reasoning, and the mandatory unresolved — state what remains open, or exactly 'None identified'. A synthesis is itself a claim and becomes the current belief for its particular."},
		s.synthesisCreate)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "knowledge_recall", Annotations: readOnly,
		Description: "Retrieve claims and syntheses about a particular (by id, URI, label, or alias) and/or carrying every given topic, in lineage order (inputs before the syntheses citing them). Marks the current synthesis and each unsynthesised entry. Operates across merged particulars."},
		s.knowledgeRecall)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "conflict_detect", Annotations: readOnly,
		Description: "Structural conflict sets for a particular (or a given set of claim ids): current synthesis, unsynthesised assertions not reconciled into it, stale syntheses citing a retracted input, and a priority. The tool does not judge contradiction — you do."},
		s.conflictDetect)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "lineage_trace", Annotations: readOnly,
		Description: "Provenance tree of a claim or synthesis: inputs with roles, recursively, including retracted ancestors and their superseded_by successors."},
		s.lineageTrace)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "knowledge_publish", Annotations: additive,
		Description: "Share claims and syntheses more widely by writing a promotion record. Claims are immutable, so scope is never rewritten: effective scope is the asserted scope widened by the promotions covering an object. Promotion may only widen — reduce exposure by retracting the promotion, not by promoting downwards — and never cascades, so promote a synthesis's inputs too when the chain should be traversable. Names objects by id only. Explicit and deliberate; not a default."},
		s.knowledgePublish)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "topics_list", Annotations: readOnly,
		Description: "(particulars extension, not part of the DKF tool set) List topic tags in use with counts, so you reuse existing tags rather than inventing near-duplicates."},
		s.topicsList)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "unresolved_list", Annotations: readOnly,
		Description: "(particulars extension, not part of the DKF tool set) What each current synthesis admits it could not settle, oldest first — the open questions of the current belief, with the number of unsynthesised assertions in each class so you can see where a question may already have new evidence. Entries saying \"None identified\" are hidden unless include_none."},
		s.unresolvedList)
	sdk.AddTool(s.srv, &sdk.Tool{Name: "workspace_status", Annotations: readOnly,
		Description: "(particulars extension, not part of the DKF tool set) The bound workspace: root, id, base-uri, object counts, validate summary, open conflicts, and — when inside a git checkout — workspace files not yet committed. Read-only; never runs a git command that writes."},
		s.workspaceStatus)
}

// --- helpers --------------------------------------------------------------

func (s *Server) load() (*store.Graph, error) {
	g, err := s.ws.Load()
	if err != nil {
		return nil, err
	}
	if err := g.Err(); err != nil {
		return nil, err
	}
	return g, nil
}

func resolveOne(g *store.Graph, q string) (*dkf.Particular, error) {
	matches := query.Resolve(g, q)
	switch len(matches) {
	case 0:
		return nil, apperr.NotFound("no particular matches %q", q)
	case 1:
		return matches[0], nil
	}
	ids := make([]string, len(matches))
	for i, m := range matches {
		ids[i] = m.ID
	}
	return nil, apperr.Usage("%q is ambiguous; it matches %s — use an id or uri", q, strings.Join(ids, ", "))
}

func (s *Server) scope(in string) (dkf.Scope, error) {
	sc := dkf.Scope(in)
	if sc == "" {
		sc = s.ws.Config.Defaults.Scope
	}
	if sc == "" {
		sc = dkf.ScopePersonal
	}
	if !dkf.ValidScope(sc) {
		return "", apperr.Usage("invalid scope %q: must be personal, organisation, or public", sc)
	}
	return sc, nil
}

func parseTS(in string) (time.Time, error) {
	if in == "" {
		return time.Now().UTC().Truncate(time.Second), nil
	}
	t, err := dkf.ParseTime(in)
	if err != nil {
		return time.Time{}, apperr.Usage("%v", err)
	}
	return t, nil
}

func confidence(f *float64) (*float64, error) {
	if f != nil && (*f < 0 || *f > 1) {
		return nil, apperr.Usage("confidence must be in [0, 1], got %v", *f)
	}
	return f, nil
}

func (s *Server) rel(id string) string {
	p, _ := s.ws.Path(id)
	return s.ws.Rel(p)
}

// --- particulars --------------------------------------------------------

type defineIn struct {
	URI     string   `json:"uri,omitempty" jsonschema:"canonical URI; minted from the label when omitted"`
	Label   string   `json:"label" jsonschema:"human-readable label (required)"`
	Aliases []string `json:"aliases,omitempty" jsonschema:"alternative names"`
}

func (s *Server) particularDefine(ctx context.Context, req *sdk.CallToolRequest, in defineIn) (*sdk.CallToolResult, any, error) {
	label := strings.TrimSpace(in.Label)
	if label == "" {
		return errResult(apperr.Usage("label is required")), nil, nil
	}
	uri := strings.TrimSpace(in.URI)
	if uri == "" {
		slug := dkf.Slugify(label)
		if slug == "" {
			return errResult(apperr.Usage("label %q yields an empty slug; pass uri explicitly", label)), nil, nil
		}
		uri = dkf.MintURI(s.ws.Config.Workspace.BaseURI, s.ws.Config.Workspace.ID, slug)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, created, err := s.ws.UpsertParticular(uri, label, in.Aliases)
	if err != nil {
		return errResult(err), nil, nil
	}
	verb := "Updated"
	if created {
		verb = "Created"
	}
	return okResult(fmt.Sprintf("%s %s (%s) %s", verb, p.ID, p.Label, p.URI)), map[string]any{"particular": p, "created": created}, nil
}

type resolveIn struct {
	Query string `json:"query" jsonschema:"id, URI, label, or alias"`
}

func (s *Server) particularResolve(ctx context.Context, req *sdk.CallToolRequest, in resolveIn) (*sdk.CallToolResult, any, error) {
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	matches := query.Resolve(g, in.Query)
	switch len(matches) {
	case 0:
		return okResult("no particular matches " + in.Query), map[string]any{"particular": nil}, nil
	case 1:
		return okResult(matches[0].ID + " " + matches[0].Label), map[string]any{"particular": matches[0]}, nil
	}
	_, err = resolveOne(g, in.Query)
	return errResult(err), nil, nil
}

type mergeIn struct {
	URIA   string    `json:"uri_a" jsonschema:"a particular (id, URI, label, or alias) or a bare URI with no local particular"`
	URIB   string    `json:"uri_b" jsonschema:"the other side, same forms"`
	Reason string    `json:"reason,omitempty" jsonschema:"why the two are the same thing"`
	Source *sourceIn `json:"source,omitempty"`
}

func (s *Server) particularMerge(ctx context.Context, req *sdk.CallToolRequest, in mergeIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	side := func(arg string) (uri, id string, err error) {
		matches := query.Resolve(g, arg)
		switch len(matches) {
		case 1:
			return matches[0].URI, matches[0].ID, nil
		case 0:
			if u, perr := url.Parse(strings.TrimSpace(arg)); perr == nil && u.Scheme != "" && !strings.ContainsAny(arg, " \t") {
				return strings.TrimSpace(arg), "", nil
			}
			return "", "", apperr.NotFound("%q matches no particular and is not a URI", arg)
		}
		_, e := resolveOne(g, arg)
		return "", "", e
	}
	ua, ida, err := side(in.URIA)
	if err != nil {
		return errResult(err), nil, nil
	}
	ub, idb, err := side(in.URIB)
	if err != nil {
		return errResult(err), nil, nil
	}
	if ua == ub {
		return errResult(apperr.Usage("both arguments resolve to %s; nothing to merge", ua)), nil, nil
	}
	if m := g.MergeBetween(ua, ub); m != nil {
		return errResult(apperr.Usage("%s and %s are already joined by %s", ua, ub, m.ID)), nil, nil
	}
	src, err := s.source(g, req, in.Source, false)
	if err != nil {
		return errResult(err), nil, nil
	}
	m, err := s.ws.CreateMerge(ua, ub, in.Reason, src, time.Now().UTC().Truncate(time.Second))
	if err != nil {
		return errResult(err), nil, nil
	}
	sides := []map[string]any{{"uri": ua, "particular": ida}, {"uri": ub, "particular": idb}}
	return okResult("Merged " + m.ID + ": " + ua + " = " + ub), map[string]any{"merge": m, "sides": sides, "path": s.rel(m.ID)}, nil
}

// --- promotion ---------------------------------------------------------------

type publishIn struct {
	ClaimIDs []string  `json:"claim_ids" jsonschema:"ids of the claims and syntheses to share more widely; ids only, never labels"`
	Scope    string    `json:"scope" jsonschema:"the scope to widen to: personal, organisation, or public"`
	Reason   string    `json:"reason,omitempty" jsonschema:"why these may be shared more widely"`
	Source   *sourceIn `json:"source,omitempty"`
}

func (s *Server) knowledgePublish(ctx context.Context, req *sdk.CallToolRequest, in publishIn) (*sdk.CallToolResult, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(in.ClaimIDs) == 0 {
		return errResult(apperr.Usage("claim_ids: name at least one claim or synthesis to promote")), nil, nil
	}
	if !dkf.ValidScope(dkf.Scope(in.Scope)) {
		return errResult(apperr.Usage("invalid scope %q: must be personal, organisation, or public", in.Scope)), nil, nil
	}
	for _, id := range in.ClaimIDs {
		if !dkf.IsAssertionID(id) {
			return errResult(apperr.Usage("%q is not a claim or synthesis id; promotion names objects by id, never by label", id)), nil, nil
		}
	}
	gAuth, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	src, err := s.source(gAuth, req, in.Source, false)
	if err != nil {
		return errResult(err), nil, nil
	}
	pr, err := s.ws.CreatePromotion(in.ClaimIDs, dkf.Scope(in.Scope), in.Reason, src, time.Now().UTC().Truncate(time.Second))
	if err != nil {
		return errResult(err), nil, nil
	}
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	warnings := query.ScopeFindingsForPromotion(g, pr)
	warnings = append(warnings, query.QuoteDisclosuresForPromotion(g, pr)...)
	out := map[string]any{"promotion": pr, "path": s.rel(pr.ID)}
	if len(warnings) > 0 {
		out["warnings"] = warnings
	}
	return okResult(fmt.Sprintf("Promoted %d object(s) to %s", len(pr.Claims), pr.Scope)), out, nil
}

// --- claims -----------------------------------------------------------------

type assertIn struct {
	ParticularID string     `json:"particular_id" jsonschema:"the subject: id, URI, label, or alias of the thing in the world the fact is about — never the document or feed it was read in"`
	Content      string     `json:"content" jsonschema:"one falsifiable statement about the world, not about a document; if it would name an article or URL, put that in source.document and state the fact instead"`
	Evidential   string     `json:"evidential" jsonschema:"what backs the claim (required, no default): observed = someone or something looked; inferred = reasoning from other claims; held = nothing external backs it, it is a position"`
	Source       *sourceIn  `json:"source,omitempty"`
	Context      *contextIn `json:"context,omitempty"`
	Confidence   *float64   `json:"confidence,omitempty" jsonschema:"0..1; 0.9+ seen directly, 0.6-0.8 inferred"`
	Scope        string     `json:"scope,omitempty" jsonschema:"shorthand for context.scope"`
	Timestamp    string     `json:"timestamp,omitempty" jsonschema:"RFC 3339 assertion time; backdate when recording a dated document"`
}

func (s *Server) claimAssert(ctx context.Context, req *sdk.CallToolRequest, in assertIn) (*sdk.CallToolResult, any, error) {
	if !dkf.ValidEvidential(dkf.Evidential(in.Evidential)) {
		if in.Evidential == "" {
			return errResult(apperr.Usage("evidential is required and has no default: observed (someone or something looked), inferred (reasoning from other claims), or held (nothing external backs it; it is a position)")), nil, nil
		}
		return errResult(apperr.Usage("invalid evidential %q: must be observed, inferred, or held", in.Evidential)), nil, nil
	}
	if dkf.Evidential(in.Evidential) == dkf.EvidentialHeld && in.Confidence != nil {
		return errResult(apperr.Usage("a held claim carries no confidence: a position is not mistaken in the way a probability describes")), nil, nil
	}
	if strings.TrimSpace(in.Content) == "" {
		return errResult(apperr.Usage("content is required")), nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	p, err := resolveOne(g, in.ParticularID)
	if err != nil {
		return errResult(err), nil, nil
	}
	scopeIn := in.Scope
	var topics []string
	if in.Context != nil {
		if in.Context.Scope != "" {
			scopeIn = in.Context.Scope
		}
		topics = in.Context.Topics
	}
	sc, err := s.scope(scopeIn)
	if err != nil {
		return errResult(err), nil, nil
	}
	conf, err := confidence(in.Confidence)
	if err != nil {
		return errResult(err), nil, nil
	}
	ts, err := parseTS(in.Timestamp)
	if err != nil {
		return errResult(err), nil, nil
	}
	src, err := s.source(g, req, in.Source, false)
	if err != nil {
		return errResult(err), nil, nil
	}
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: in.Content, Source: src, Context: dkf.Context{Scope: sc, Topics: topics}, Timestamp: ts, Evidential: dkf.Evidential(in.Evidential), Confidence: conf}
	if err := s.ws.Create(c); err != nil {
		return errResult(err), nil, nil
	}
	if err := s.ws.UpsertIndex(c); err != nil {
		return errResult(err), nil, nil
	}
	out := map[string]any{"claim": c, "path": s.rel(c.ID)}
	if w := query.QuoteAbsentLocally(s.ws, src.Document); w != "" {
		out["warnings"] = []string{w}
	}
	return okResult("Asserted " + c.ID + " about " + p.Label), out, nil
}

type retractIn struct {
	ClaimID      string    `json:"claim_id" jsonschema:"a claim, synthesis, or merge record id"`
	Reason       string    `json:"reason" jsonschema:"why it is retracted (required)"`
	Source       *sourceIn `json:"source,omitempty"`
	Kind         string    `json:"kind,omitempty" jsonschema:"why it died: defect (the claim misread its source), supersession (the source was right then, the world moved on), or provenance-failure (the source itself was wrong); never inferred from superseded_by"`
	SupersededBy string    `json:"superseded_by,omitempty" jsonschema:"id of the claim or synthesis that replaces it (not for merges)"`
}

func (s *Server) claimRetract(ctx context.Context, req *sdk.CallToolRequest, in retractIn) (*sdk.CallToolResult, any, error) {
	if strings.TrimSpace(in.Reason) == "" {
		return errResult(apperr.Usage("reason is required")), nil, nil
	}
	if !dkf.IsRetractableID(in.ClaimID) {
		return errResult(apperr.Usage("%q is not a claim, synthesis, or merge id", in.ClaimID)), nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.SupersededBy != "" {
		if !dkf.IsAssertionID(in.SupersededBy) {
			return errResult(apperr.Usage("superseded_by %q is not a claim or synthesis id", in.SupersededBy)), nil, nil
		}
		if !s.ws.Exists(in.SupersededBy) {
			return errResult(apperr.NotFound("superseded_by %s does not exist", in.SupersededBy)), nil, nil
		}
	}
	gAuth, err := s.ws.Load()
	if err != nil {
		return errResult(err), nil, nil
	}
	src, err := s.source(gAuth, req, in.Source, false)
	if err != nil {
		return errResult(err), nil, nil
	}
	if in.Kind != "" && !dkf.ValidRetractionKind(dkf.RetractionKind(in.Kind)) {
		return errResult(apperr.Usage("invalid kind %q: must be defect, supersession, or provenance-failure", in.Kind)), nil, nil
	}
	r := &dkf.Retracted{Timestamp: time.Now().UTC().Truncate(time.Second), Reason: in.Reason, Source: src, Kind: dkf.RetractionKind(in.Kind), SupersededBy: in.SupersededBy}
	updated, err := s.ws.Retract(in.ClaimID, r)
	if err != nil {
		return errResult(err), nil, nil
	}
	if err := s.ws.UpsertIndex(updated); err != nil {
		return errResult(err), nil, nil
	}
	return okResult("Retracted " + in.ClaimID), map[string]any{"id": in.ClaimID, "type": updated.ObjectType(), "retracted": r}, nil
}

// --- syntheses --------------------------------------------------------------

type inputIn struct {
	ID     string `json:"id" jsonschema:"a claim or synthesis id"`
	Role   string `json:"role" jsonschema:"thesis | antithesis"`
	Weight string `json:"weight,omitempty" jsonschema:"primary (default) | qualifying"`
}

type synthesisIn struct {
	ParticularID string     `json:"particular_id" jsonschema:"the subject: id, URI, label, or alias"`
	Content      string     `json:"content" jsonschema:"the resolution, carrying the reasoning"`
	Inputs       []inputIn  `json:"inputs" jsonschema:"thesis/antithesis inputs (at least one)"`
	Unresolved   string     `json:"unresolved" jsonschema:"what could not be reconciled, or exactly 'None identified'"`
	Source       *sourceIn  `json:"source,omitempty" jsonschema:"harness is required (defaults to the connected client)"`
	Method       string     `json:"method,omitempty" jsonschema:"reconciliation (the inputs disagreed about a fact) | qualification (each true in a different context) | positions (no evidence settles this); default reconciliation"`
	Context      *contextIn `json:"context,omitempty"`
	Confidence   *float64   `json:"confidence,omitempty"`
	Timestamp    string     `json:"timestamp,omitempty" jsonschema:"RFC 3339; current is chosen by timestamp then id"`
}

func (s *Server) synthesisCreate(ctx context.Context, req *sdk.CallToolRequest, in synthesisIn) (*sdk.CallToolResult, any, error) {
	if strings.TrimSpace(in.Content) == "" {
		return errResult(apperr.Usage("content is required")), nil, nil
	}
	if strings.TrimSpace(in.Unresolved) == "" {
		return errResult(apperr.Usage("unresolved is required; state what remains, or \"None identified\"")), nil, nil
	}
	if len(in.Inputs) == 0 {
		return errResult(apperr.Usage("at least one input is required")), nil, nil
	}
	parsed := make([]dkf.Input, 0, len(in.Inputs))
	for _, i := range in.Inputs {
		w := dkf.Weight(i.Weight)
		if w == "" {
			w = dkf.WeightPrimary
		}
		input := dkf.Input{ID: strings.TrimSpace(i.ID), Role: dkf.Role(i.Role), Weight: w}
		switch {
		case !dkf.IsValidID(input.ID):
			return errResult(apperr.Usage("input %q is not a valid id", i.ID)), nil, nil
		case !dkf.IsAssertionID(input.ID):
			return errResult(apperr.Usage("input %s must be a claim or synthesis", i.ID)), nil, nil
		case !dkf.ValidRole(input.Role):
			return errResult(apperr.Usage("input %s: role must be thesis or antithesis", i.ID)), nil, nil
		case !dkf.ValidWeight(w):
			return errResult(apperr.Usage("input %s: weight must be primary or qualifying", i.ID)), nil, nil
		}
		parsed = append(parsed, input)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	p, err := resolveOne(g, in.ParticularID)
	if err != nil {
		return errResult(err), nil, nil
	}
	warnings := []string{}
	for _, i := range parsed {
		child := g.Assertion(i.ID)
		if child == nil {
			return errResult(apperr.NotFound("input %s does not exist", i.ID)), nil, nil
		}
		if child.GetRetracted() != nil {
			warnings = append(warnings, "input "+i.ID+" is retracted")
		}
	}
	scopeIn := ""
	var topics []string
	if in.Context != nil {
		scopeIn, topics = in.Context.Scope, in.Context.Topics
	}
	sc, err := s.scope(scopeIn)
	if err != nil {
		return errResult(err), nil, nil
	}
	conf, err := confidence(in.Confidence)
	if err != nil {
		return errResult(err), nil, nil
	}
	ts, err := parseTS(in.Timestamp)
	if err != nil {
		return errResult(err), nil, nil
	}
	src, err := s.source(g, req, in.Source, true)
	if err != nil {
		return errResult(err), nil, nil
	}
	method := in.Method
	if method == "" {
		method = dkf.DefaultMethod
	}
	if !dkf.ValidMethod(method) {
		return errResult(apperr.Usage("invalid method %q: must be reconciliation, qualification, or positions", method)), nil, nil
	}
	syn := &dkf.Synthesis{ID: dkf.NewID(dkf.TypeSynthesis), Subject: p.ID, Content: in.Content, Inputs: parsed, Unresolved: in.Unresolved, Source: src, Method: method, Timestamp: ts, Context: dkf.Context{Scope: sc, Topics: topics}, Confidence: conf}
	if err := s.ws.Create(syn); err != nil {
		return errResult(err), nil, nil
	}
	if err := s.ws.UpsertIndex(syn); err != nil {
		return errResult(err), nil, nil
	}
	return okResult(fmt.Sprintf("Synthesised %s about %s from %d inputs", syn.ID, p.Label, len(parsed))), map[string]any{"synthesis": syn, "path": s.rel(syn.ID), "warnings": warnings}, nil
}

// --- queries ----------------------------------------------------------------

type recallIn struct {
	ParticularID     string   `json:"particular_id,omitempty" jsonschema:"id, URI, label, or alias; omit to recall by topic only"`
	Query            string   `json:"query,omitempty" jsonschema:"alias of particular_id, as in the DKF tool table"`
	Scope            string   `json:"scope,omitempty" jsonschema:"only this scope"`
	Topics           []string `json:"topics,omitempty" jsonschema:"all must match"`
	IncludeRetracted bool     `json:"include_retracted,omitempty"`
	Limit            int      `json:"limit,omitempty" jsonschema:"keep the most recent N in lineage order"`
	Author           string   `json:"author,omitempty" jsonschema:"objects asserted by or reported from this particular (id, URI, label, or alias), each labelled with its relations"`
}

func (s *Server) knowledgeRecall(ctx context.Context, req *sdk.CallToolRequest, in recallIn) (*sdk.CallToolResult, any, error) {
	subject := in.ParticularID
	if subject == "" {
		subject = in.Query
	}
	if subject == "" && len(in.Topics) == 0 && in.Author == "" {
		return errResult(apperr.Usage("pass particular_id (or query), topics, author, or a combination")), nil, nil
	}
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	opts := query.RecallOptions{Topics: in.Topics, IncludeRetracted: in.IncludeRetracted, Limit: in.Limit}
	if in.Scope != "" {
		if !dkf.ValidScope(dkf.Scope(in.Scope)) {
			return errResult(apperr.Usage("invalid scope %q", in.Scope)), nil, nil
		}
		opts.Scope = dkf.Scope(in.Scope)
	}
	out := map[string]any{}
	if subject != "" {
		p, err := resolveOne(g, subject)
		if err != nil {
			return errResult(err), nil, nil
		}
		opts.Subject = p.ID
		out["subject"] = p.ID
		if class := g.ClassOf(p.ID); len(class) > 1 {
			out["class"] = class
		}
	}
	if in.Author != "" {
		p, err := resolveOne(g, in.Author)
		if err != nil {
			return errResult(err), nil, nil
		}
		opts.Author = p.ID
		out["author"] = p.ID
	}
	entries := query.Recall(g, opts)
	out["entries"], out["count"] = entries, len(entries)
	cur := ""
	for _, e := range entries {
		if e.Current {
			cur = e.ID
		}
	}
	return okResult(fmt.Sprintf("%d entries; current: %s", len(entries), orNone(cur))), out, nil
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

type conflictIn struct {
	ParticularID string   `json:"particular_id,omitempty" jsonschema:"id, URI, label, or alias"`
	ClaimIDs     []string `json:"claim_ids,omitempty" jsonschema:"alternatively, a set of claim/synthesis ids to analyse as a universe"`
}

func (s *Server) conflictDetect(ctx context.Context, req *sdk.CallToolRequest, in conflictIn) (*sdk.CallToolResult, any, error) {
	if (in.ParticularID == "") == (len(in.ClaimIDs) == 0) {
		return errResult(apperr.Usage("pass exactly one of particular_id or claim_ids")), nil, nil
	}
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	if len(in.ClaimIDs) > 0 {
		r, err := query.AnalyseSet(g, in.ClaimIDs)
		if err != nil {
			return errResult(err), nil, nil
		}
		return okResult(fmt.Sprintf("set of %d: current %s, %d unsynthesised, %d stale", len(r.Set), orNone(r.Current), len(r.Unsynthesised), len(r.Stale))), r, nil
	}
	p, err := resolveOne(g, in.ParticularID)
	if err != nil {
		return errResult(err), nil, nil
	}
	reports := query.Conflicts(g, p.ID)
	return okResult(fmt.Sprintf("%d report(s) for %s", len(reports), p.Label)), map[string]any{"reports": reports, "count": len(reports)}, nil
}

type lineageIn struct {
	ClaimID string `json:"claim_id" jsonschema:"a claim or synthesis id"`
	Depth   int    `json:"depth,omitempty" jsonschema:"levels to expand; 0 = unlimited"`
}

func (s *Server) lineageTrace(ctx context.Context, req *sdk.CallToolRequest, in lineageIn) (*sdk.CallToolResult, any, error) {
	if !dkf.IsValidID(in.ClaimID) {
		return errResult(apperr.Usage("%q is not a valid id", in.ClaimID)), nil, nil
	}
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	tree, err := query.Lineage(g, in.ClaimID, in.Depth)
	if err != nil {
		return errResult(err), nil, nil
	}
	return okResult("lineage of " + in.ClaimID), tree, nil
}

type unresolvedIn struct {
	ParticularID string `json:"particular_id,omitempty" jsonschema:"restrict to one particular's merge class (id, URI, label, or alias)"`
	Scope        string `json:"scope,omitempty" jsonschema:"only entries whose current synthesis has this effective scope"`
	IncludeNone  bool   `json:"include_none,omitempty" jsonschema:"include entries whose unresolved is exactly \"None identified\""`
}

func (s *Server) unresolvedList(ctx context.Context, req *sdk.CallToolRequest, in unresolvedIn) (*sdk.CallToolResult, any, error) {
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	opts := query.UnresolvedOptions{IncludeNone: in.IncludeNone, Scope: dkf.Scope(in.Scope)}
	if in.ParticularID != "" {
		p, err := resolveOne(g, in.ParticularID)
		if err != nil {
			return errResult(err), nil, nil
		}
		opts.Subject = p.ID
	}
	entries := query.Unresolved(g, opts)
	return okResult(fmt.Sprintf("%d unresolved", len(entries))), map[string]any{"entries": entries, "count": len(entries)}, nil
}

type topicsIn struct {
	ParticularID     string `json:"particular_id,omitempty"`
	Scope            string `json:"scope,omitempty"`
	IncludeRetracted bool   `json:"include_retracted,omitempty"`
}

func (s *Server) topicsList(ctx context.Context, req *sdk.CallToolRequest, in topicsIn) (*sdk.CallToolResult, any, error) {
	g, err := s.load()
	if err != nil {
		return errResult(err), nil, nil
	}
	opts := query.RecallOptions{IncludeRetracted: in.IncludeRetracted, Scope: dkf.Scope(in.Scope)}
	if in.ParticularID != "" {
		p, err := resolveOne(g, in.ParticularID)
		if err != nil {
			return errResult(err), nil, nil
		}
		opts.Subject = p.ID
	}
	topics := query.Topics(g, opts)
	return okResult(fmt.Sprintf("%d topics", len(topics))), map[string]any{"topics": topics, "count": len(topics)}, nil
}
