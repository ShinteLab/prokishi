package server

import (
	"testing"

	"github.com/ShinteLab/prokishi/registry"
)

func TestVersion(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string maps to DevelopmentVersion", "", DevelopmentVersion},
		{"non-empty version passed through", "1.2.3", "1.2.3"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			if err := Version(tc.input)(cfg); err != nil {
				t.Fatalf("Version(%q)() error: %v", tc.input, err)
			}
			if cfg.Version != tc.want {
				t.Fatalf("cfg.Version = %q, want %q", cfg.Version, tc.want)
			}
		})
	}
}

func TestConfig_isDev(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    bool
	}{
		{"DevelopmentVersion is dev", DevelopmentVersion, true},
		{"empty string is not dev (isDev only checks the exact sentinel)", "", false},
		{"real version string is not dev", "1.2.3", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Version: tc.version}
			if got := cfg.isDev(); got != tc.want {
				t.Fatalf("isDev() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWithRegistry(t *testing.T) {
	reg := registry.New()

	cfg := &Config{}
	if err := WithRegistry(reg)(cfg); err != nil {
		t.Fatalf("WithRegistry()() error: %v", err)
	}
	if cfg.Registry != reg {
		t.Fatalf("cfg.Registry = %p, want %p", cfg.Registry, reg)
	}
}

func TestWithRegistry_Nil(t *testing.T) {
	cfg := &Config{Registry: registry.New()}
	if err := WithRegistry(nil)(cfg); err != nil {
		t.Fatalf("WithRegistry(nil)() error: %v", err)
	}
	if cfg.Registry != nil {
		t.Fatalf("cfg.Registry = %v, want nil", cfg.Registry)
	}
}
