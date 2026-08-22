package graph

import (
	"bytes"
	"encoding/json"
)

// Properties marshals in a fixed order with the @odata.type specifiers
// Microsoft requires for collections, so exports are byte-deterministic.
type Properties struct {
	Title            string
	URL              string
	ParticularURI    string
	Scope            string
	Topics           []string
	Authors          []string
	LastModified     string
	ClaimCount       int
	OpenQuestions    int
	CurrentSynthesis string
}

// MarshalJSON writes the properties in declaration order, omitting empty
// optional values and adding collection type specifiers.
func (p Properties) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	first := true
	kv := func(key string, val any) error {
		if !first {
			b.WriteByte(',')
		}
		first = false
		k, err := json.Marshal(key)
		if err != nil {
			return err
		}
		v, err := json.Marshal(val)
		if err != nil {
			return err
		}
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
		return nil
	}
	if err := kv("title", p.Title); err != nil {
		return nil, err
	}
	if p.URL != "" {
		if err := kv("url", p.URL); err != nil {
			return nil, err
		}
	}
	if err := kv("particularUri", p.ParticularURI); err != nil {
		return nil, err
	}
	if err := kv("scope", p.Scope); err != nil {
		return nil, err
	}
	if len(p.Topics) > 0 {
		if err := kv("topics@odata.type", "Collection(String)"); err != nil {
			return nil, err
		}
		if err := kv("topics", p.Topics); err != nil {
			return nil, err
		}
	}
	if len(p.Authors) > 0 {
		if err := kv("authors@odata.type", "Collection(String)"); err != nil {
			return nil, err
		}
		if err := kv("authors", p.Authors); err != nil {
			return nil, err
		}
	}
	if p.LastModified != "" {
		if err := kv("lastModifiedDateTime", p.LastModified); err != nil {
			return nil, err
		}
	}
	if err := kv("claimCount", p.ClaimCount); err != nil {
		return nil, err
	}
	if err := kv("openQuestions", p.OpenQuestions); err != nil {
		return nil, err
	}
	if p.CurrentSynthesis != "" {
		if err := kv("currentSynthesis", p.CurrentSynthesis); err != nil {
			return nil, err
		}
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// SchemaProperty is one entry in the connection schema.
type SchemaProperty struct {
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	IsSearchable         bool     `json:"isSearchable,omitempty"`
	IsQueryable          bool     `json:"isQueryable,omitempty"`
	IsRetrievable        bool     `json:"isRetrievable,omitempty"`
	IsRefinable          bool     `json:"isRefinable,omitempty"`
	IsExactMatchRequired bool     `json:"isExactMatchRequired,omitempty"`
	Labels               []string `json:"labels,omitempty"`
	Aliases              []string `json:"aliases,omitempty"`
}

// Connection is the externalConnection creation body.
type Connection struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Schema is the schema registration body.
type Schema struct {
	BaseType   string           `json:"baseType"`
	Properties []SchemaProperty `json:"properties"`
}

// Registration is what `export --format graph --schema` emits.
type Registration struct {
	Connection Connection `json:"connection"`
	Schema     Schema     `json:"schema"`
}

// NewRegistration builds the connection and schema payloads.
//
// The flags satisfy Microsoft's constraints: a property is never both
// searchable and refinable; isExactMatchRequired appears only on properties
// that are not searchable; every labelled property is retrievable; and each
// label is used at most once.
func NewRegistration(id, name, description string) Registration {
	if name == "" {
		name = "particulars knowledge"
	}
	if description == "" {
		description = "Reviewed claims and syntheses from a Dialectical Knowledge Format workspace: what is currently believed about each particular, what it rests on, and what remains unreconciled."
	}
	return Registration{
		Connection: Connection{ID: id, Name: name, Description: description},
		Schema: Schema{
			BaseType: "microsoft.graph.externalItem",
			Properties: []SchemaProperty{
				{Name: "title", Type: "String", IsSearchable: true, IsQueryable: true, IsRetrievable: true, Labels: []string{"title"}, Aliases: []string{"subject", "particular"}},
				{Name: "url", Type: "String", IsRetrievable: true, Labels: []string{"url"}},
				{Name: "particularUri", Type: "String", IsQueryable: true, IsRetrievable: true, IsExactMatchRequired: true, Aliases: []string{"uri"}},
				{Name: "scope", Type: "String", IsQueryable: true, IsRetrievable: true, IsRefinable: true},
				{Name: "topics", Type: "StringCollection", IsQueryable: true, IsRetrievable: true, IsRefinable: true, IsExactMatchRequired: true, Aliases: []string{"tags", "categories"}},
				{Name: "authors", Type: "StringCollection", IsQueryable: true, IsRetrievable: true, Labels: []string{"authors"}},
				{Name: "lastModifiedDateTime", Type: "DateTime", IsQueryable: true, IsRetrievable: true, IsRefinable: true, Labels: []string{"lastModifiedDateTime"}},
				{Name: "claimCount", Type: "Int64", IsQueryable: true, IsRetrievable: true},
				{Name: "openQuestions", Type: "Int64", IsQueryable: true, IsRetrievable: true, Aliases: []string{"unresolved"}},
				{Name: "currentSynthesis", Type: "String", IsQueryable: true, IsRetrievable: true, IsExactMatchRequired: true},
			},
		},
	}
}
