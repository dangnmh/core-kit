package httplog

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FieldRule defines the masking rule for a specific field or pattern.
type FieldRule struct {
	Field    string   `yaml:"field"`
	Strategy Strategy `yaml:"strategy"`
}

// UnmarshalYAML supports both shorthand string (`- "password"`) and full object (`- field: "token", strategy: "remove"`).
func (f *FieldRule) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		f.Field = str
		f.Strategy = StrategyRedact
		return nil
	}

	type rawFieldRule FieldRule
	var raw rawFieldRule
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("decode field rule: %w", err)
	}

	*f = FieldRule(raw)
	if f.Strategy == "" {
		f.Strategy = StrategyRedact
	}
	return nil
}

// PayloadConfig defines filtering and logging configuration for request or response payloads.
type PayloadConfig struct {
	LogResponseOn any         `yaml:"log_response_on"`
	IgnoreFields  []FieldRule `yaml:"ignore_fields"`
}

// RuleConfig defines endpoint-specific matching and masking rules.
type RuleConfig struct {
	Direction    string         `yaml:"direction,omitempty"` // "inbound" or "outbound"
	Path         string         `yaml:"path"`
	Methods      []string       `yaml:"methods"`
	Enable       bool           `yaml:"enable"`
	Enabled      bool           `yaml:"enabled"`
	IgnoreFields []FieldRule    `yaml:"ignore_fields"`
	Request      *PayloadConfig `yaml:"request"`
	Response     *PayloadConfig `yaml:"response"`
}

// UnmarshalYAML decodes a RuleConfig from YAML, defaulting Enable and Enabled to true.
func (r *RuleConfig) UnmarshalYAML(value *yaml.Node) error {
	type rawRuleConfig RuleConfig
	raw := rawRuleConfig{
		Enable:  true,
		Enabled: true,
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	*r = RuleConfig(raw)
	return nil
}

// IsEnabled reports whether this rule is explicitly enabled for logging.
func (r *RuleConfig) IsEnabled() bool {
	if r == nil {
		return false
	}
	return r.Enable || r.Enabled
}

// GlobalConfig defines global defaults for log filtering and redaction.
type GlobalConfig struct {
	MaskValue       string   `yaml:"mask_value"`
	MaxBodySize     int      `yaml:"max_body_size"`
	LogResponseOn   any      `yaml:"log_response_on"`
	Headers         []string `yaml:"headers"`
	LogInbound      *bool    `yaml:"log_inbound,omitempty"`
	EnableInbound   *bool    `yaml:"enable_inbound,omitempty"`
	InboundEnabled  *bool    `yaml:"inbound_enabled,omitempty"`
	LogOutbound     *bool    `yaml:"log_outbound,omitempty"`
	EnableOutbound  *bool    `yaml:"enable_outbound,omitempty"`
	OutboundEnabled *bool    `yaml:"outbound_enabled,omitempty"`
	LogUnmatched    bool     `yaml:"log_unmatched"`
}

// Config is the root configuration struct for httplog.
type Config struct {
	Version  string       `yaml:"version"`
	Default  GlobalConfig `yaml:"default"`
	Inbound  []RuleConfig `yaml:"inbound,omitempty"`
	Outbound []RuleConfig `yaml:"outbound,omitempty"`
	Rules    []RuleConfig `yaml:"rules,omitempty"`
}

// UnmarshalYAML decodes YAML data into Config, supporting:
// 1. Explicit inbound and outbound sections under 'rules:' (e.g. rules.inbound, rules.outbound)
// 2. Top-level 'inbound:' and 'outbound:' sections
// 3. Legacy flat list under 'rules:'
// 4. Direction-specific toggle: log_inbound / log_outbound (or enable_inbound / enable_outbound)
func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	type rawConfig struct {
		Version         string       `yaml:"version"`
		Default         GlobalConfig `yaml:"default"`
		LogInbound      *bool        `yaml:"log_inbound"`
		EnableInbound   *bool        `yaml:"enable_inbound"`
		InboundEnabled  *bool        `yaml:"inbound_enabled"`
		LogOutbound     *bool        `yaml:"log_outbound"`
		EnableOutbound  *bool        `yaml:"enable_outbound"`
		OutboundEnabled *bool        `yaml:"outbound_enabled"`
		Inbound         []RuleConfig `yaml:"inbound"`
		Outbound        []RuleConfig `yaml:"outbound"`
		Rules           yaml.Node    `yaml:"rules"`
	}

	var raw rawConfig
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	c.Version = raw.Version
	c.Default = raw.Default
	c.Inbound = raw.Inbound
	c.Outbound = raw.Outbound

	// Resolve LogInbound flag priority
	if raw.Default.LogInbound != nil {
		c.Default.LogInbound = raw.Default.LogInbound
	} else if raw.Default.EnableInbound != nil {
		c.Default.LogInbound = raw.Default.EnableInbound
	} else if raw.Default.InboundEnabled != nil {
		c.Default.LogInbound = raw.Default.InboundEnabled
	} else if raw.LogInbound != nil {
		c.Default.LogInbound = raw.LogInbound
	} else if raw.EnableInbound != nil {
		c.Default.LogInbound = raw.EnableInbound
	} else if raw.InboundEnabled != nil {
		c.Default.LogInbound = raw.InboundEnabled
	}

	// Resolve LogOutbound flag priority
	if raw.Default.LogOutbound != nil {
		c.Default.LogOutbound = raw.Default.LogOutbound
	} else if raw.Default.EnableOutbound != nil {
		c.Default.LogOutbound = raw.Default.EnableOutbound
	} else if raw.Default.OutboundEnabled != nil {
		c.Default.LogOutbound = raw.Default.OutboundEnabled
	} else if raw.LogOutbound != nil {
		c.Default.LogOutbound = raw.LogOutbound
	} else if raw.EnableOutbound != nil {
		c.Default.LogOutbound = raw.EnableOutbound
	} else if raw.OutboundEnabled != nil {
		c.Default.LogOutbound = raw.OutboundEnabled
	}

	// Mark directions if not already specified
	for i := range c.Inbound {
		if c.Inbound[i].Direction == "" {
			c.Inbound[i].Direction = "inbound"
		}
	}
	for i := range c.Outbound {
		if c.Outbound[i].Direction == "" {
			c.Outbound[i].Direction = "outbound"
		}
	}

	if !raw.Rules.IsZero() {
		switch raw.Rules.Kind {
		case yaml.SequenceNode:
			var flatRules []RuleConfig
			if err := raw.Rules.Decode(&flatRules); err != nil {
				return fmt.Errorf("decode rules sequence: %w", err)
			}
			c.Rules = flatRules
			for _, r := range flatRules {
				switch r.Direction {
				case "inbound":
					c.Inbound = append(c.Inbound, r)

				case "outbound":
					c.Outbound = append(c.Outbound, r)
				}
			}
		case yaml.MappingNode:
			var rulesMap struct {
				Inbound  []RuleConfig `yaml:"inbound"`
				Outbound []RuleConfig `yaml:"outbound"`
			}
			if err := raw.Rules.Decode(&rulesMap); err != nil {
				return fmt.Errorf("decode rules mapping: %w", err)
			}
			for i := range rulesMap.Inbound {
				if rulesMap.Inbound[i].Direction == "" {
					rulesMap.Inbound[i].Direction = "inbound"
				}
			}
			for i := range rulesMap.Outbound {
				if rulesMap.Outbound[i].Direction == "" {
					rulesMap.Outbound[i].Direction = "outbound"
				}
			}
			if len(rulesMap.Inbound) > 0 {
				c.Inbound = append(c.Inbound, rulesMap.Inbound...)
			}
			if len(rulesMap.Outbound) > 0 {
				c.Outbound = append(c.Outbound, rulesMap.Outbound...)
			}
		}
	}

	// For backwards compatibility and unified rule lists:
	// If c.Rules is empty but Inbound or Outbound are specified, populate c.Rules with all of them.
	if len(c.Rules) == 0 && (len(c.Inbound) > 0 || len(c.Outbound) > 0) {
		c.Rules = make([]RuleConfig, 0, len(c.Inbound)+len(c.Outbound))
		c.Rules = append(c.Rules, c.Inbound...)
		c.Rules = append(c.Rules, c.Outbound...)
	}

	if c.Default.MaskValue == "" {
		c.Default.MaskValue = "[REDACTED]"
	}

	return nil
}

// IsLogInboundEnabled reports whether inbound HTTP logging is enabled.
// Returns true by default unless explicitly disabled via log_inbound: false.
func (c *Config) IsLogInboundEnabled() bool {
	if c == nil {
		return false
	}
	if c.Default.LogInbound != nil {
		return *c.Default.LogInbound
	}
	return true
}

// IsLogOutboundEnabled reports whether outbound HTTP logging is enabled.
// Returns true by default unless explicitly disabled via log_outbound: false.
func (c *Config) IsLogOutboundEnabled() bool {
	if c == nil {
		return true
	}
	if c.Default.LogOutbound != nil {
		return *c.Default.LogOutbound
	}
	return true
}

// IsLogUnmatchedEnabled reports whether HTTP requests that do not match any defined rule should be logged.
// Returns false by default unless explicitly enabled via log_unmatched: true.
func (c *Config) IsLogUnmatchedEnabled() bool {
	if c == nil {
		return false
	}
	return c.Default.LogUnmatched
}

// InboundRules returns all configured inbound rules.
func (c *Config) InboundRules() []RuleConfig {
	if c == nil {
		return nil
	}
	return c.Inbound
}

// OutboundRules returns all configured outbound rules.
func (c *Config) OutboundRules() []RuleConfig {
	if c == nil {
		return nil
	}
	return c.Outbound
}

// AllRules returns all rules across inbound and outbound.
func (c *Config) AllRules() []RuleConfig {
	if c == nil {
		return nil
	}
	return c.Rules
}

// ParseConfig parses YAML configuration bytes into a Config struct and applies defaults.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	if cfg.Default.MaskValue == "" {
		cfg.Default.MaskValue = "[REDACTED]"
	}

	return &cfg, nil
}

// LoadConfig reads and parses a YAML configuration file from the given path.
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	return ParseConfig(data)
}
