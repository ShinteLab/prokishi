package server

import "github.com/ShinteLab/prokishi/registry"

type Config struct {
	Version  string
	Registry *registry.Registry
	UseAuth  bool
}

const (
	DevelopmentVersion = "Development"
)

func (c *Config) isDev() bool {
	if c.Version == DevelopmentVersion {
		return true
	}
	return false
}

type Option func(*Config) error

func Version(v string) Option {
	return func(c *Config) error {
		if v == "" {
			c.Version = DevelopmentVersion
		} else {
			c.Version = v
		}
		return nil
	}
}

func WithRegistry(r *registry.Registry) Option {
	return func(c *Config) error {
		c.Registry = r
		return nil
	}
}

func WithAuth(useAuth bool) Option {
	return func(c *Config) error {
		c.UseAuth = useAuth
		return nil
	}
}
