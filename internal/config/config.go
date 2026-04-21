package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	keyring "github.com/zalando/go-keyring"
)

const (
	DefaultProfileName = "default"
	DefaultBaseURL     = "https://dev.dixa.io/v1"
	DefaultOutput      = "auto"
	ServiceName        = "dixa-cli"
)

var ErrSecretNotFound = errors.New("secret not found")

type SecretStore interface {
	Get(profile string) (string, error)
	Set(profile, value string) error
	Delete(profile string) error
}

type KeyringStore struct {
	Service string
}

func (s KeyringStore) Get(profile string) (string, error) {
	value, err := keyring.Get(s.serviceName(), profile)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrSecretNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s KeyringStore) Set(profile, value string) error {
	return keyring.Set(s.serviceName(), profile, value)
}

func (s KeyringStore) Delete(profile string) error {
	err := keyring.Delete(s.serviceName(), profile)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrSecretNotFound
	}
	return err
}

func (s KeyringStore) serviceName() string {
	if s.Service != "" {
		return s.Service
	}
	return ServiceName
}

type File struct {
	DefaultProfile string             `toml:"default_profile,omitempty"`
	Profiles       map[string]Profile `toml:"profiles,omitempty"`
}

type Profile struct {
	BaseURL string `toml:"base_url,omitempty"`
	Output  string `toml:"output,omitempty"`
}

type Overrides struct {
	Profile    string
	BaseURL    string
	APIKey     string
	Output     string
	ProfileSet bool
	BaseURLSet bool
	APIKeySet  bool
	OutputSet  bool
}

type Resolved struct {
	Profile       string `json:"profile"`
	BaseURL       string `json:"base_url"`
	Output        string `json:"output"`
	APIKey        string `json:"-"`
	ProfileSource string `json:"profile_source"`
	BaseURLSource string `json:"base_url_source"`
	OutputSource  string `json:"output_source"`
	APIKeySource  string `json:"api_key_source"`
	ConfigPath    string `json:"config_path"`
}

type Manager struct {
	Path  string
	Store SecretStore
}

func DefaultConfigPath(home string) string {
	return filepath.Join(home, ".config", "dixa", "config.toml")
}

func NewManager(home string, store SecretStore) *Manager {
	return &Manager{
		Path:  DefaultConfigPath(home),
		Store: store,
	}
}

func (m *Manager) Load() (File, error) {
	if m == nil {
		return File{}, errors.New("config manager is nil")
	}
	data, err := os.ReadFile(m.Path)
	if errors.Is(err, os.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read config %s: %w", m.Path, err)
	}
	var cfg File
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return File{}, fmt.Errorf("decode config %s: %w", m.Path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

func (m *Manager) Save(cfg File) error {
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if err := os.MkdirAll(filepath.Dir(m.Path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(m.Path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (m *Manager) UpsertProfile(name string, profile Profile, apiKey string, setDefault bool) error {
	cfg, err := m.Load()
	if err != nil {
		return err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	cfg.Profiles[name] = profile
	if setDefault || cfg.DefaultProfile == "" {
		cfg.DefaultProfile = name
	}
	if err := m.Save(cfg); err != nil {
		return err
	}
	if m.Store != nil && apiKey != "" {
		if err := m.Store.Set(name, apiKey); err != nil {
			return fmt.Errorf("store API key for profile %q: %w", name, err)
		}
	}
	return nil
}

func (m *Manager) DeleteProfile(name string, removeProfile bool) error {
	if m.Store != nil {
		err := m.Store.Delete(name)
		if err != nil && !errors.Is(err, ErrSecretNotFound) {
			return fmt.Errorf("delete API key for profile %q: %w", name, err)
		}
	}
	if !removeProfile {
		return nil
	}
	cfg, err := m.Load()
	if err != nil {
		return err
	}
	delete(cfg.Profiles, name)
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = ""
	}
	return m.Save(cfg)
}

func (m *Manager) Resolve(overrides Overrides, getenv func(string) string) (Resolved, error) {
	cfg, err := m.Load()
	if err != nil {
		return Resolved{}, err
	}
	if getenv == nil {
		getenv = os.Getenv
	}

	profile, profileSource := resolveProfile(cfg, overrides, getenv)
	profileCfg := cfg.Profiles[profile]
	baseURL, baseURLSource := resolveValue(overrides.BaseURLSet, overrides.BaseURL, getenv("DIXA_BASE_URL"), profileCfg.BaseURL, DefaultBaseURL)
	output, outputSource := resolveValue(overrides.OutputSet, overrides.Output, getenv("DIXA_OUTPUT"), profileCfg.Output, DefaultOutput)
	output = strings.TrimSpace(output)
	if output == "" {
		output = DefaultOutput
	}

	var apiKey, apiKeySource string
	switch {
	case overrides.APIKeySet && strings.TrimSpace(overrides.APIKey) != "":
		apiKey, apiKeySource = strings.TrimSpace(overrides.APIKey), "flag"
	case strings.TrimSpace(getenv("DIXA_API_KEY")) != "":
		apiKey, apiKeySource = strings.TrimSpace(getenv("DIXA_API_KEY")), "env"
	case m.Store != nil:
		secret, err := m.Store.Get(profile)
		if err == nil && strings.TrimSpace(secret) != "" {
			apiKey, apiKeySource = strings.TrimSpace(secret), "profile"
		} else if err != nil && !errors.Is(err, ErrSecretNotFound) {
			return Resolved{}, fmt.Errorf("resolve API key from profile %q: %w", profile, err)
		}
	}

	return Resolved{
		Profile:       profile,
		BaseURL:       baseURL,
		Output:        output,
		APIKey:        apiKey,
		ProfileSource: profileSource,
		BaseURLSource: baseURLSource,
		OutputSource:  outputSource,
		APIKeySource:  apiKeySource,
		ConfigPath:    m.Path,
	}, nil
}

func resolveProfile(cfg File, overrides Overrides, getenv func(string) string) (string, string) {
	switch {
	case overrides.ProfileSet && strings.TrimSpace(overrides.Profile) != "":
		return strings.TrimSpace(overrides.Profile), "flag"
	case strings.TrimSpace(getenv("DIXA_PROFILE")) != "":
		return strings.TrimSpace(getenv("DIXA_PROFILE")), "env"
	case strings.TrimSpace(cfg.DefaultProfile) != "":
		return strings.TrimSpace(cfg.DefaultProfile), "config"
	default:
		return DefaultProfileName, "default"
	}
}

func resolveValue(flagSet bool, flagValue, envValue, profileValue, defaultValue string) (string, string) {
	switch {
	case flagSet && strings.TrimSpace(flagValue) != "":
		return strings.TrimSpace(flagValue), "flag"
	case strings.TrimSpace(envValue) != "":
		return strings.TrimSpace(envValue), "env"
	case strings.TrimSpace(profileValue) != "":
		return strings.TrimSpace(profileValue), "profile"
	default:
		return defaultValue, "default"
	}
}

func MaskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 6 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:3] + strings.Repeat("*", len(secret)-6) + secret[len(secret)-3:]
}
