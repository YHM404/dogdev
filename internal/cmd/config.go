package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM struct {
		Provider string `yaml:"provider"`
		Model    string `yaml:"model"`
		BaseURL  string `yaml:"base_url,omitempty"`
		APIKey   string `yaml:"api_key,omitempty"`
	} `yaml:"llm"`
	Embedding struct {
		Provider string `yaml:"provider"`
		Model    string `yaml:"model"`
	} `yaml:"embedding"`
	Qdrant struct {
		URL            string  `yaml:"url"`
		APIKey         string  `yaml:"api_key,omitempty"`
		Collection     string  `yaml:"collection"`
		TopK           int     `yaml:"top_k"`
		ScoreThreshold float32 `yaml:"score_threshold"`
		ChunkSize      int     `yaml:"chunk_size"`
		ChunkOverlap   int     `yaml:"chunk_overlap"`
	} `yaml:"qdrant"`
}

func loadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		// Try to find config in default locations
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		// Check common config locations
		configLocations := []string{
			".dogdev.yaml",                                // Current directory
			filepath.Join(homeDir, ".dogdev.yaml"),        // Home directory
			filepath.Join(homeDir, ".config/dogdev.yaml"), // XDG config directory
			"/etc/dogdev.yaml",                            // System-wide config
		}

		for _, loc := range configLocations {
			if _, err := os.Stat(loc); err == nil {
				configPath = loc
				break
			}
		}

		if configPath == "" {
			return getDefaultConfig(), nil
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and normalize Qdrant URL
	if config.Qdrant.URL != "" {
		if !strings.HasPrefix(config.Qdrant.URL, "http://") && !strings.HasPrefix(config.Qdrant.URL, "https://") {
			config.Qdrant.URL = "http://" + config.Qdrant.URL
		}
		_, err := url.Parse(config.Qdrant.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid Qdrant URL: %w", err)
		}
	}

	return &config, nil
}

func getDefaultConfig() *Config {
	var config Config
	config.LLM.Provider = "ollama"
	config.LLM.Model = "llama3.2:latest"
	config.Embedding.Provider = "ollama"
	config.Embedding.Model = "nomic-embed-text:latest"
	config.Qdrant.URL = "http://localhost:6334"
	config.Qdrant.Collection = "monitor"
	config.Qdrant.TopK = 4
	config.Qdrant.ScoreThreshold = 0.7
	config.Qdrant.ChunkSize = 500
	config.Qdrant.ChunkOverlap = 50
	return &config
}
