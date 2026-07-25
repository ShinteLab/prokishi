package prokishi_test

import (
	"github.com/ShinteLab/prokishi"
	"testing"
)

func TestCode(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{name: "sets code", code: "ABC123", want: "ABC123"},
		{name: "empty code", code: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var conf prokishi.Config
			opt := prokishi.Code(tt.code)
			if err := opt(&conf); err != nil {
				t.Fatalf("Code(%q) returned error: %v", tt.code, err)
			}
			if conf.Code != tt.want {
				t.Errorf("conf.Code = %q, want %q", conf.Code, tt.want)
			}
		})
	}
}

func TestEngine(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{name: "sets engine id", id: "engine-uuid", want: "engine-uuid"},
		{name: "empty engine id", id: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var conf prokishi.Config
			opt := prokishi.Engine(tt.id)
			if err := opt(&conf); err != nil {
				t.Fatalf("Engine(%q) returned error: %v", tt.id, err)
			}
			if conf.EngineId != tt.want {
				t.Errorf("conf.EngineId = %q, want %q", conf.EngineId, tt.want)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want string
	}{
		{name: "empty version becomes Development sentinel", v: "", want: "Development"},
		{name: "non empty version is preserved as-is", v: "1.2.3", want: "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var conf prokishi.Config
			opt := prokishi.Version(tt.v)
			if err := opt(&conf); err != nil {
				t.Fatalf("Version(%q) returned error: %v", tt.v, err)
			}
			if conf.Version != tt.want {
				t.Errorf("conf.Version = %q, want %q", conf.Version, tt.want)
			}
		})
	}
}
