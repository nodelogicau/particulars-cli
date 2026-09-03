// Package store owns everything that touches the filesystem: workspace
// discovery, dkf.yaml, object files, retraction append, and index.yaml.
package store

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// ConfigFile is the workspace marker and configuration file name.
const ConfigFile = "dkf.yaml"

// Config is the parsed dkf.yaml. Unknown keys are ignored on read; the file
// is only ever written by Init, so nothing is lost.
type Config struct {
	Format    string          `yaml:"format"`
	Workspace WorkspaceConfig `yaml:"workspace"`
	Defaults  Defaults        `yaml:"defaults,omitempty"`
}

// WorkspaceConfig identifies the workspace and its optional base URI.
type WorkspaceConfig struct {
	ID      string `yaml:"id"`
	BaseURI string `yaml:"base-uri,omitempty"`
	// Conventions names the workspace's conventions document — prose for
	// agents (topic vocabulary, local naming) delivered to MCP clients with
	// the initialize instructions. Relative to the workspace root; when
	// empty, dkf.md at the root applies if present. See ConventionsPath.
	Conventions string `yaml:"conventions,omitempty"`
}

// Defaults supplies values omitted on the command line.
type Defaults struct {
	Scope  dkf.Scope  `yaml:"scope,omitempty"`
	Source dkf.Source `yaml:"source,omitempty"`
}

// NewConfig returns a config for a fresh workspace with a minted id.
func NewConfig() Config {
	return Config{
		Format:    dkf.FormatVersion,
		Workspace: WorkspaceConfig{ID: dkf.NewUUID()},
		Defaults:  Defaults{Scope: dkf.ScopePersonal},
	}
}

// Validate checks the config is usable.
func (c Config) Validate() error {
	if c.Format == "" {
		return fmt.Errorf("%s: missing format", ConfigFile)
	}
	if c.Format != dkf.FormatVersion {
		return fmt.Errorf("%s: unsupported format %q (want %s)", ConfigFile, c.Format, dkf.FormatVersion)
	}
	if c.Workspace.ID == "" {
		return fmt.Errorf("%s: missing workspace.id", ConfigFile)
	}
	if c.Defaults.Scope != "" && !dkf.ValidScope(c.Defaults.Scope) {
		return fmt.Errorf("%s: invalid defaults.scope %q", ConfigFile, c.Defaults.Scope)
	}
	if !dkf.ValidBaseURI(c.Workspace.BaseURI) {
		return fmt.Errorf("%w: %s: workspace.base-uri %q must end in \"/\"", ErrInvalidBaseURI, ConfigFile, c.Workspace.BaseURI)
	}
	return nil
}

// ConventionsPath returns workspace.conventions when it names a relative path
// inside the workspace, or "" with a warning when it does not. The check is
// lexical, on the cleaned slash-normalised path: an absolute path, a leading
// slash, or a first segment of ".." is invalid. An invalid value never makes
// the workspace invalid — readers that predate the key ignore it, and a
// strict rule would open the workspace under an older tool and refuse it
// under a newer one — so it is treated as absent and reported.
func (c Config) ConventionsPath() (rel, warning string) {
	v := c.Workspace.Conventions
	if v == "" {
		return "", ""
	}
	clean := path.Clean(filepath.ToSlash(v))
	// filepath.IsAbs alone is platform-dependent — on Windows "/x" has no
	// volume and reports false — so a leading slash is checked on the slashed
	// form too: the path must be relative on every platform the workspace is
	// cloned to.
	if filepath.IsAbs(v) || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Sprintf("%s: workspace.conventions %q is not a relative path inside the workspace; ignored", ConfigFile, v)
	}
	return v, ""
}

func encodeConfig(c Config) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", ConfigFile, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}
