package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig              `yaml:"server"`
	Logging     LoggingConfig             `yaml:"logging"`
	Metrics     MetricsConfig             `yaml:"metrics"`
	Cache       CacheConfig               `yaml:"cache"`
	Providers   map[string]ProviderConfig `yaml:"providers"`
	Translation TranslationConfig         `yaml:"translation"`
	Vector      VectorConfig              `yaml:"vector"`
}

type ServerConfig struct {
	Transport string     `yaml:"transport"`
	Host      string     `yaml:"host"`
	Port      int        `yaml:"port"`
	Auth      AuthConfig `yaml:"auth"`
	CORS      CORSConfig `yaml:"cors"`
}

type AuthConfig struct {
	Type  string `yaml:"type"`
	Token string `yaml:"token"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type LoggingConfig struct {
	Format   string `yaml:"format"`
	Level    string `yaml:"level"`
	MaskText bool   `yaml:"mask_text"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type CacheConfig struct {
	DefaultTTL string          `yaml:"default_ttl"`
	L1         CacheTierConfig `yaml:"l1"`
	L2         CacheTierConfig `yaml:"l2"`
	L3         CacheTierConfig `yaml:"l3"`
}

type CacheTierConfig struct {
	Type       string `yaml:"type"`
	MaxEntries int    `yaml:"max_entries"`
	Address    string `yaml:"address"`
	Password   string `yaml:"password"`
	DSN        string `yaml:"dsn"`
	TTL        string `yaml:"ttl"`
}

type ProviderConfig map[string]any

type TranslationConfig struct {
	DefaultProvider       string   `yaml:"default_provider"`
	MaxTextLength         int      `yaml:"max_text_length"`
	MaxBatchItems         int      `yaml:"max_batch_items"`
	RequestTimeout        string   `yaml:"request_timeout"`
	FallbackChain         []string `yaml:"fallback_chain"`
	GlossaryPreprocessing bool     `yaml:"glossary_preprocessing"`
}

type VectorConfig struct {
	Enabled             bool    `yaml:"enabled"`
	Provider            string  `yaml:"provider"`
	EmbeddingModel      string  `yaml:"embedding_model"`
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
}

var envPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

func Load(path string) (*Config, error) {
	if path == "" {
		return defaults(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	data = []byte(expandEnv(string(data)))
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func defaults() *Config {
	cfg := &Config{}
	applyDefaults(cfg)
	return cfg
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Transport == "" {
		cfg.Server.Transport = "stdio"
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8787
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Metrics.Path == "" {
		cfg.Metrics.Path = "/metrics"
	}
	if cfg.Cache.DefaultTTL == "" {
		cfg.Cache.DefaultTTL = "24h"
	}
	if cfg.Translation.DefaultProvider == "" {
		cfg.Translation.DefaultProvider = "openai"
	}
	if cfg.Translation.MaxTextLength == 0 {
		cfg.Translation.MaxTextLength = 10000
	}
	if cfg.Translation.MaxBatchItems == 0 {
		cfg.Translation.MaxBatchItems = 50
	}
	if cfg.Translation.RequestTimeout == "" {
		cfg.Translation.RequestTimeout = "60s"
	}
	if cfg.Vector.SimilarityThreshold == 0 {
		cfg.Vector.SimilarityThreshold = 0.85
	}
}

func validate(cfg *Config) error {
	if cfg.Server.Transport != "stdio" && cfg.Server.Transport != "http" {
		return fmt.Errorf("server.transport must be stdio or http")
	}
	if cfg.Translation.MaxTextLength <= 0 {
		return fmt.Errorf("translation.max_text_length must be > 0")
	}
	if cfg.Translation.MaxBatchItems <= 0 {
		return fmt.Errorf("translation.max_batch_items must be > 0")
	}
	return nil
}

func expandEnv(input string) string {
	return envPattern.ReplaceAllStringFunc(input, func(match string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		parts := strings.SplitN(inner, ":-", 2)
		key := parts[0]
		val := os.Getenv(key)
		if val == "" && len(parts) == 2 {
			return parts[1]
		}
		return val
	})
}

func (p ProviderConfig) String(key string) string {
	v, ok := p[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func (p ProviderConfig) Bool(key string) bool {
	v, ok := p[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func (p ProviderConfig) Int(key string, def int) int {
	v, ok := p[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return def
}
