package query

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Verification outcomes for a claim's document.
const (
	// CodeContextDrift: the quote is still there, but the document around it
	// changed — "In staging, …" becoming "In production, …" falsifies a claim
	// without touching a character of the quote.
	CodeContextDrift = "context_drift"
	// CodeQuoteDrift: the cited text is gone or altered.
	CodeQuoteDrift = "quote_drift"
	// CodeUnverifiedDocument: the provenance was not machine-checked. This
	// says nothing is known, not that anything is wrong: a conversation, a
	// page behind a login, and a remote URI are all legitimate sources.
	CodeUnverifiedDocument = "unverified_document"
)

// DocumentState is the result of checking one document.
type DocumentState struct {
	Code    string // "" when verified
	Message string
}

// Verified reports whether the document was checked and nothing had moved.
func (d DocumentState) Verified() bool { return d.Code == "" }

// Drifted reports whether the document moved under the claim.
func (d DocumentState) Drifted() bool {
	return d.Code == CodeContextDrift || d.Code == CodeQuoteDrift
}

// VerifyDocument checks a claim's document against the workspace, offline.
//
// A document is checkable only when it resolves to a file inside the
// workspace — a relative path, or a file: URI. Nothing here reaches the
// network, now or ever: validate must stay deterministic, credential-free,
// and runnable on a laptop with no connection.
func VerifyDocument(ws *store.Workspace, d dkf.Document) DocumentState {
	if d.IsZero() {
		return DocumentState{}
	}
	if d.Hash == "" && d.Quote == "" {
		return DocumentState{CodeUnverifiedDocument, "document carries no hash or quote, so nothing can be checked"}
	}
	if algo := dkf.HashAlgorithm(d.Hash); d.Hash != "" && algo != dkf.AlgorithmSHA256 {
		// Another implementation's algorithm is a legitimate hash we cannot
		// compute. Unverified, never invalid — refusing it would mean two
		// conformant implementations unable to check each other.
		return DocumentState{CodeUnverifiedDocument, "hash algorithm " + algo + " is not implemented here, so the document is not checked"}
	}
	path, ok := LocalDocumentPath(ws, d.Ref)
	if !ok {
		return DocumentState{CodeUnverifiedDocument, "document is not a file in this workspace; verification would need a fetch, which validate does not do"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return DocumentState{CodeUnverifiedDocument, "document does not resolve to a readable file: " + ws.Rel(path)}
	}
	content := string(data)
	quotePresent := d.Quote == "" || dkf.QuoteMatches(content, d.Quote)
	hashMatches := d.Hash == "" || d.Hash == dkf.HashDocumentBytes(data)

	switch {
	case d.Hash == "":
		// Quote only: the quote is the whole signal.
		if quotePresent {
			return DocumentState{}
		}
		return DocumentState{CodeQuoteDrift, "the quoted text no longer appears in " + ws.Rel(path)}
	case hashMatches:
		if quotePresent {
			return DocumentState{}
		}
		// The document is byte-identical yet the quote is absent: the quote
		// was never in this document, which is worth saying differently.
		return DocumentState{CodeQuoteDrift, "the quoted text does not appear in " + ws.Rel(path) + ", though the document is unchanged since the claim was written"}
	case quotePresent:
		return DocumentState{CodeContextDrift, ws.Rel(path) + " changed since the claim was written; the quoted text is still present, but its surroundings are not"}
	default:
		return DocumentState{CodeQuoteDrift, ws.Rel(path) + " changed and the quoted text is gone"}
	}
}

// LocalDocumentPath resolves a document reference to a file inside the
// workspace, or reports that it is not one. Remote URIs are never resolved:
// nothing here fetches.
func LocalDocumentPath(ws *store.Workspace, ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", false
	}
	if u, err := url.Parse(ref); err == nil && u.Scheme != "" {
		if u.Scheme != "file" {
			return "", false // http, https, urn, anything remote
		}
		ref = u.Path
	}
	if filepath.IsAbs(ref) {
		if within(ws.Root, ref) {
			return ref, true
		}
		return "", false
	}
	abs := filepath.Join(ws.Root, filepath.FromSlash(ref))
	if !within(ws.Root, abs) {
		return "", false // ../ escaping the workspace
	}
	if fi, err := os.Stat(abs); err != nil || fi.IsDir() {
		return abs, err == nil && !fi.IsDir()
	}
	return abs, true
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
