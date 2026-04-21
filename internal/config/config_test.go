package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type memoryStore struct {
	values map[string]string
}

func (m *memoryStore) Get(profile string) (string, error) {
	value, ok := m.values[profile]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}

func (m *memoryStore) Set(profile, value string) error {
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[profile] = value
	return nil
}

func (m *memoryStore) Delete(profile string) error {
	if _, ok := m.values[profile]; !ok {
		return ErrSecretNotFound
	}
	delete(m.values, profile)
	return nil
}

func TestResolvePrecedence(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	store := &memoryStore{}
	manager := NewManager(home, store)
	if err := manager.UpsertProfile("team", Profile{
		BaseURL: "https://profile.example",
		Output:  "yaml",
	}, "profile-secret", true); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	tests := []struct {
		name      string
		overrides Overrides
		env       map[string]string
		want      Resolved
	}{
		{
			name: "profile defaults",
			want: Resolved{
				Profile:       "team",
				BaseURL:       "https://profile.example",
				Output:        "yaml",
				APIKey:        "profile-secret",
				ProfileSource: "config",
				BaseURLSource: "profile",
				OutputSource:  "profile",
				APIKeySource:  "profile",
				ConfigPath:    filepath.Join(home, ".config", "dixa", "config.toml"),
			},
		},
		{
			name: "env overrides profile",
			env: map[string]string{
				"DIXA_PROFILE":  "team",
				"DIXA_BASE_URL": "https://env.example",
				"DIXA_OUTPUT":   "json",
				"DIXA_API_KEY":  "env-secret",
			},
			want: Resolved{
				Profile:       "team",
				BaseURL:       "https://env.example",
				Output:        "json",
				APIKey:        "env-secret",
				ProfileSource: "env",
				BaseURLSource: "env",
				OutputSource:  "env",
				APIKeySource:  "env",
				ConfigPath:    filepath.Join(home, ".config", "dixa", "config.toml"),
			},
		},
		{
			name: "flags override env",
			overrides: Overrides{
				Profile:    "flag-profile",
				BaseURL:    "https://flag.example",
				Output:     "table",
				APIKey:     "flag-secret",
				ProfileSet: true,
				BaseURLSet: true,
				OutputSet:  true,
				APIKeySet:  true,
			},
			env: map[string]string{
				"DIXA_PROFILE":  "team",
				"DIXA_BASE_URL": "https://env.example",
				"DIXA_OUTPUT":   "json",
				"DIXA_API_KEY":  "env-secret",
			},
			want: Resolved{
				Profile:       "flag-profile",
				BaseURL:       "https://flag.example",
				Output:        "table",
				APIKey:        "flag-secret",
				ProfileSource: "flag",
				BaseURLSource: "flag",
				OutputSource:  "flag",
				APIKeySource:  "flag",
				ConfigPath:    filepath.Join(home, ".config", "dixa", "config.toml"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resolved, err := manager.Resolve(tt.overrides, func(key string) string {
				return tt.env[key]
			})
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if resolved != tt.want {
				t.Fatalf("resolved mismatch:\nwant %#v\ngot  %#v", tt.want, resolved)
			}
		})
	}
}

func TestUpsertAndDeleteProfile(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	store := &memoryStore{}
	manager := NewManager(home, store)
	if err := manager.UpsertProfile("default", Profile{BaseURL: "https://api.example", Output: "json"}, "super-secret", true); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	cfg, err := manager.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DefaultProfile != "default" {
		t.Fatalf("expected default profile to be set, got %q", cfg.DefaultProfile)
	}
	if _, ok := cfg.Profiles["default"]; !ok {
		t.Fatalf("expected saved profile in config")
	}
	if _, err := store.Get("default"); err != nil {
		t.Fatalf("expected secret to be stored: %v", err)
	}

	if err := manager.DeleteProfile("default", true); err != nil {
		t.Fatalf("delete profile: %v", err)
	}
	if _, err := store.Get("default"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("expected missing secret after delete, got %v", err)
	}
	if _, err := os.Stat(manager.Path); err != nil {
		t.Fatalf("expected config file to remain after profile deletion: %v", err)
	}
}

func TestMaskSecret(t *testing.T) {
	t.Parallel()

	if got := MaskSecret("abcdef123456"); got != "abc******456" {
		t.Fatalf("unexpected masked secret: %s", got)
	}
}
