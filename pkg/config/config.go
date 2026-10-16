package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/tailscale/hujson"
	"gopkg.in/yaml.v3"
)

// Options defines custom rules for resolving the configuration file path.
type Options struct {
	FlagName     string                      // CLI flag name (default: "config")
	EnvVar       string                      // Environment variable name (default: "CONFIG_PATH")
	DefaultFiles []string                    // Candidate file paths to check in order
	EnvLookup    func(string) (string, bool) // Environment variable lookup function (default: os.LookupEnv)
}

// Option is a functional option for customizing Options.
type Option func(*Options)

// WithEnvLookup sets a custom environment lookup function for struct tag overrides.
func WithEnvLookup(fn func(string) (string, bool)) Option {
	return func(o *Options) {
		o.EnvLookup = fn
	}
}

// WithFlagName sets a custom CLI flag name.
func WithFlagName(name string) Option {
	return func(o *Options) {
		o.FlagName = name
	}
}

// WithEnvVar sets a custom environment variable name.
func WithEnvVar(name string) Option {
	return func(o *Options) {
		o.EnvVar = name
	}
}

// WithDefaultFiles sets candidate file paths to search in order.
func WithDefaultFiles(files ...string) Option {
	return func(o *Options) {
		o.DefaultFiles = files
	}
}

// WithFilePath sets an explicit file path to load directly.
func WithFilePath(path string) Option {
	return func(o *Options) {
		if path != "" {
			o.DefaultFiles = []string{path}
			o.FlagName = ""
			o.EnvVar = ""
		}
	}
}

// DefaultOptions returns the standard options used across services.
func DefaultOptions() Options {
	return Options{
		FlagName: "config",
		EnvVar:   "CONFIG_PATH",
		DefaultFiles: []string{
			"config.yaml",
			"config.docker.yaml",
			"/app/config.yaml",
			"/app/config.docker.yaml",
		},
		EnvLookup: os.LookupEnv,
	}
}

// ResolvePath locates the configuration file according to the search hierarchy:
// 1. CLI flag (if flag.Parsed() and flag exists with non-empty value)
// 2. Environment variable (e.g. CONFIG_PATH)
// 3. Candidate files list in order (first existing file on disk)
func ResolvePath(opts ...Option) string {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// 1. CLI flag
	if options.FlagName != "" && flag.Parsed() {
		f := flag.Lookup(options.FlagName)
		if f != nil && f.Value.String() != "" {
			return f.Value.String()
		}
	}

	// 2. Environment variable
	if options.EnvVar != "" {
		if val := os.Getenv(options.EnvVar); val != "" {
			return val
		}
	}

	// 3. Candidate files
	for _, path := range options.DefaultFiles {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func unmarshalConfigFile(path string, data []byte, target any) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" || ext == ".jsonc" {
		std, err := hujson.Standardize(data)
		if err != nil {
			return fmt.Errorf("failed to standardize json %q: %w", path, err)
		}
		if err := json.Unmarshal(std, target); err != nil {
			return fmt.Errorf("failed to unmarshal json %q: %w", path, err)
		}
		return nil
	}
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal yaml %q: %w", path, err)
	}
	return nil
}

// Load loads YAML, JSON, or JSONC from the resolved path into a generic struct T.
// If defaultVal is provided, its values are copied to the result before unmarshaling.
// Environment variable overrides (via `env` struct tags) are applied after file unmarshaling.
// Struct validation is performed unconditionally.
// Returns the loaded config struct pointer, the resolved file path (if any), and any error.
func Load[T any](defaultVal *T, opts ...Option) (*T, string, error) {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	cfg := new(T)
	if defaultVal != nil {
		*cfg = *defaultVal
	}

	path := ResolvePath(opts...)
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, path, fmt.Errorf("failed to read config file %q: %w", path, err)
		}

		if err := unmarshalConfigFile(path, data, cfg); err != nil {
			return cfg, path, err
		}
	}

	lookup := options.EnvLookup
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if err := applyEnv(cfg, lookup); err != nil {
		return cfg, path, err
	}

	// Validate configuration struct using go-playground/validator
	v := validator.New()
	if err := v.Struct(cfg); err != nil {
		if path != "" {
			return cfg, path, fmt.Errorf("validate config %q: %w", path, err)
		}
		return cfg, path, fmt.Errorf("validate config: %w", err)
	}

	return cfg, path, nil
}

// LoadInto loads YAML, JSON, or JSONC from the resolved path into an existing pointer target struct.
// Environment variable overrides (via `env` struct tags) are applied after unmarshaling.
// Struct validation is performed unconditionally.
func LoadInto(target any, opts ...Option) (string, error) {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	path := ResolvePath(opts...)
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return path, fmt.Errorf("failed to read config file %q: %w", path, err)
		}

		if err := unmarshalConfigFile(path, data, target); err != nil {
			return path, err
		}
	}

	lookup := options.EnvLookup
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if err := applyEnv(target, lookup); err != nil {
		return path, err
	}

	v := validator.New()
	if err := v.Struct(target); err != nil {
		if path != "" {
			return path, fmt.Errorf("validate config %q: %w", path, err)
		}
		return path, fmt.Errorf("validate config: %w", err)
	}

	return path, nil
}

// Validate checks struct constraints using go-playground/validator/v10.
func Validate(target any) error {
	if target == nil {
		return fmt.Errorf("cannot validate nil target")
	}
	v := validator.New()
	if err := v.Struct(target); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	return nil
}
