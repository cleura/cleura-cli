// Package config handles the CLI configuration file and the resolution of
// effective settings from flags, environment variables and profiles.
package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/cleura/cleura-client-go/cleura"
	"gopkg.in/yaml.v3"
)

// Version is the current config file schema version, stamped on every Save.
// Bump it only for changes an older CLI could misread; readers reject files
// from the future instead of silently dropping fields they do not know.
const Version = 1

// Profile holds the stored settings for one named profile.
type Profile struct {
	// Cloud is the cloud name. It selects the predefined API URL for "public"
	// and "compliant", and is also used as a path parameter by parts of the
	// API (e.g. Gardener), so private clouds set both cloud and api_url.
	Cloud string `yaml:"cloud,omitempty"`
	// APIURL overrides the cloud's default API URL. Required for private clouds.
	APIURL    string `yaml:"api_url,omitempty"`
	Username  string `yaml:"username,omitempty"`
	Token     string `yaml:"token,omitempty"`
	Region    string `yaml:"region,omitempty"`
	ProjectID string `yaml:"project_id,omitempty"`
	// TokenStoredAt records when the token was obtained. Tokens are
	// short-lived and the API exposes no expiry, so the storage time is the
	// only basis for staleness diagnostics.
	TokenStoredAt time.Time `yaml:"token_stored_at,omitempty"`
}

// Config is the on-disk CLI configuration.
type Config struct {
	Version        int                 `yaml:"version,omitempty"`
	CurrentProfile string              `yaml:"current_profile,omitempty"`
	Profiles       map[string]*Profile `yaml:"profiles,omitempty"`

	path string
}

// Path returns the config file location: $CLEURA_CONFIG if set, otherwise
// $XDG_CONFIG_HOME/cleura/config.yaml, otherwise ~/.config/cleura/config.yaml.
func Path() (string, error) {
	path, _, err := PathWithSource()
	return path, err
}

// PathWithSource returns the config file location and which mechanism chose it
// ("$CLEURA_CONFIG", "$XDG_CONFIG_HOME", or "default").
func PathWithSource() (string, string, error) {
	if p := os.Getenv("CLEURA_CONFIG"); p != "" {
		return p, "$CLEURA_CONFIG", nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "cleura", "config.yaml"), "$XDG_CONFIG_HOME", nil
	}
	if runtime.GOOS == "windows" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", "", fmt.Errorf("cannot determine config directory: %w", err)
		}
		return filepath.Join(base, "cleura", "config.yaml"), "default", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "cleura", "config.yaml"), "default", nil
}

// Load reads the config file. A missing file yields an empty config.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	cfg := &Config{path: path}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w\nfix the file, or move it aside and run 'cleura login' to recreate it", path, err)
	}
	if cfg.Version > Version {
		return nil, fmt.Errorf("config %s was written by a newer cleura (config version %d, this cleura supports %d); upgrade cleura, or move the file aside and run 'cleura login' to recreate it", path, cfg.Version, Version)
	}
	return cfg, nil
}

// Save writes the config file with owner-only permissions (it contains
// tokens), atomically: a crash mid-write must not destroy the stored tokens.
func (c *Config) Save() error {
	c.Version = Version
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	data = append([]byte("# Managed by the cleura CLI; manual comments and formatting are not preserved.\n"), data...)

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), c.path)
}

// Profile returns the named profile, creating it if it does not exist yet.
func (c *Config) Profile(name string) *Profile {
	if c.Profiles == nil {
		c.Profiles = map[string]*Profile{}
	}
	if c.Profiles[name] == nil {
		c.Profiles[name] = &Profile{}
	}
	return c.Profiles[name]
}

// ProfileNames returns the configured profile names, sorted.
func (c *Config) ProfileNames() []string {
	return slices.Sorted(maps.Keys(c.Profiles))
}

// Settings are the effective values for one invocation after applying
// precedence: flag > environment variable > profile > default. Cloud is the
// logical cloud name and is always set; APIURL is only set when an explicit
// URL override is in effect for the endpoint.
type Settings struct {
	ProfileName string
	// ProfileExists reports whether the selected profile is present in the
	// config file, distinguishing "no such profile" from "not logged in".
	ProfileExists bool
	Cloud         string
	APIURL        string
	Username      string
	Token         string
	Region        string
	ProjectID     string
	// TokenStoredAt is when the effective token was obtained. Only known
	// when the token comes from the profile; zero for env-provided tokens.
	TokenStoredAt time.Time

	// Sources records where each value above was resolved from, for display
	// by "cleura config view".
	Sources Sources
}

// Sources names the origin of each resolved Settings value: a flag
// ("--cloud"), an environment variable ("$CLEURA_CLOUD"), the selected
// profile ("profile"), a built-in ("default"), or "" when the value is unset.
type Sources struct {
	Profile   string
	Cloud     string
	APIURL    string
	Username  string
	Token     string
	Region    string
	ProjectID string
	// Endpoint is the source that determined the API endpoint: the source of
	// the explicit URL when one is in effect, otherwise the source of the
	// named cloud.
	Endpoint string
}

// Flags carries command-line flag values into Resolve. Empty fields are unset.
type Flags struct {
	Profile   string
	Cloud     string
	APIURL    string
	Region    string
	ProjectID string
}

// ProfileName resolves which profile to use: --profile flag, CLEURA_PROFILE,
// the config's current_profile, or "default".
func (c *Config) ProfileName(flags Flags) string {
	name, _ := c.profileName(flags)
	return name
}

func (c *Config) profileName(flags Flags) (string, string) {
	return pick(
		candidate{flags.Profile, "--profile"},
		candidate{os.Getenv("CLEURA_PROFILE"), "$CLEURA_PROFILE"},
		candidate{c.CurrentProfile, "config"},
		candidate{"default", "default"},
	)
}

// Resolve computes the effective settings for one invocation.
func (c *Config) Resolve(flags Flags) Settings {
	var s Settings
	s.ProfileName, s.Sources.Profile = c.profileName(flags)
	profile := c.Profiles[s.ProfileName]
	s.ProfileExists = profile != nil
	if profile == nil {
		profile = &Profile{}
	}

	s.Username, s.Sources.Username = pick(
		candidate{os.Getenv("CLEURA_API_USERNAME"), "$CLEURA_API_USERNAME"},
		candidate{profile.Username, "profile"},
	)
	s.Token, s.Sources.Token = pick(
		candidate{os.Getenv("CLEURA_API_TOKEN"), "$CLEURA_API_TOKEN"},
		candidate{profile.Token, "profile"},
	)
	if s.Sources.Token == "profile" {
		s.TokenStoredAt = profile.TokenStoredAt
	}
	s.Region, s.Sources.Region = pick(
		candidate{flags.Region, "--region"},
		candidate{os.Getenv("CLEURA_REGION"), "$CLEURA_REGION"},
		candidate{profile.Region, "profile"},
	)
	s.ProjectID, s.Sources.ProjectID = pick(
		candidate{flags.ProjectID, "--project-id"},
		candidate{os.Getenv("CLEURA_PROJECT_ID"), "$CLEURA_PROJECT_ID"},
		candidate{profile.ProjectID, "profile"},
	)
	s.resolveEndpoint(flags, profile)
	return s
}

// resolveEndpoint resolves the logical cloud name and the endpoint override.
//
// The cloud name is the nearest one set (flag > env > profile > "public") and
// is always set; besides selecting the default API URL it is used as a path
// parameter by parts of the API.
//
// For the endpoint, sources are considered in precedence order as a whole, so
// an explicit --cloud flag beats a stored api_url for a different cloud. When
// the endpoint is determined by a named cloud, a lower-precedence source that
// pairs an explicit URL with that same cloud name still applies — a private
// cloud stores {cloud: acme, api_url: ...} and selecting --cloud acme must
// keep using its URL. APIURL is set only when an explicit URL is in effect.
func (s *Settings) resolveEndpoint(flags Flags, profile *Profile) {
	s.Cloud, s.Sources.Cloud = pick(
		candidate{flags.Cloud, "--cloud"},
		candidate{os.Getenv("CLEURA_CLOUD"), "$CLEURA_CLOUD"},
		candidate{profile.Cloud, "profile"},
		candidate{"public", "default"},
	)
	s.Sources.Endpoint = s.Sources.Cloud

	endpointSources := []struct{ url, urlSource, cloud string }{
		{flags.APIURL, "--api-url", flags.Cloud},
		{os.Getenv("CLEURA_API_URL"), "$CLEURA_API_URL", os.Getenv("CLEURA_CLOUD")},
		{profile.APIURL, "profile", profile.Cloud},
	}
	for i, src := range endpointSources {
		if src.url != "" {
			s.APIURL, s.Sources.APIURL = src.url, src.urlSource
			s.Sources.Endpoint = src.urlSource
			return
		}
		if src.cloud != "" {
			// This source's named cloud determines the endpoint. A lower
			// source whose URL is paired with the same cloud name provides it.
			for _, lower := range endpointSources[i+1:] {
				if lower.url != "" && lower.cloud == src.cloud {
					s.APIURL, s.Sources.APIURL = lower.url, lower.urlSource
					s.Sources.Endpoint = lower.urlSource
					return
				}
			}
			return
		}
	}
}

// ResolveURL returns the API base URL: an explicit URL if set, otherwise the
// predefined URL for the named cloud.
func (s Settings) ResolveURL() (string, error) {
	if s.APIURL != "" {
		return s.APIURL, nil
	}
	url, err := cleura.DefaultAPIURL(s.Cloud)
	if err != nil {
		return "", fmt.Errorf("%w; use --api-url (or CLEURA_API_URL) for private clouds, or --cloud public|compliant", err)
	}
	return url, nil
}

// candidate is one possible value for a setting together with its source name.
type candidate struct {
	value, source string
}

// pick returns the first non-empty candidate and its source.
func pick(candidates ...candidate) (string, string) {
	for _, c := range candidates {
		if c.value != "" {
			return c.value, c.source
		}
	}
	return "", ""
}
