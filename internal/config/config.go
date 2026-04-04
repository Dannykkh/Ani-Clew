package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type ProviderSettings struct {
	APIKey  string `json:"apiKey,omitempty"`
	BaseURL string `json:"baseUrl,omitempty"`
}

type Config struct {
	Port            int                         `json:"port"`
	DefaultProvider string                      `json:"defaultProvider"`
	DefaultModel    string                      `json:"defaultModel"`
	RouterEnabled   bool                        `json:"routerEnabled"`
	ResponseLang    string                      `json:"responseLang"`    // "ko", "en", "ja", "zh", "auto"
	UILang          string                      `json:"uiLang"`          // "ko", "en"
	SkillSource     string                      `json:"skillSource"`     // "claude", "codex", "gemini", "all", "none"
	SkillDirs       []string                    `json:"skillDirs"`       // extra custom skill directories
	MCPConfigPaths  []string                    `json:"mcpConfigPaths"`  // extra MCP config file paths
	WorkDir         string                      `json:"workDir"`         // default workspace
	Providers       map[string]ProviderSettings  `json:"providers"`
}

func DefaultConfig() Config {
	return Config{
		Port:      4000,
		Providers: map[string]ProviderSettings{},
	}
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-proxy")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func Load() Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	if cfg.Providers == nil {
		cfg.Providers = map[string]ProviderSettings{}
	}
	return cfg
}

func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

func ConfigPath() string {
	return configPath()
}

func _ () string { return runtime.GOOS } // keep import
