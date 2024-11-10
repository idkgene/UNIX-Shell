package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
    HistoryFile    string            `json:"history_file"`
    AliasFile      string            `json:"alias_file"`
    MaxHistory     int               `json:"max_history"`
    Prompt         string            `json:"prompt"`
    DefaultEditor  string            `json:"default_editor"`
    Aliases        map[string]string `json:"aliases"`
    ColorScheme    ColorScheme       `json:"color_scheme"`
    AutoComplete   bool              `json:"auto_complete"`
    PluginsEnabled bool              `json:"plugins_enabled"`
    PluginsDir     string            `json:"plugins_dir"`
}

type ColorScheme struct {
	Prompt    string `json:"prompt"`
	Command   string `json:"command"`
	Error     string `json:"error"`
	Warning   string `json:"warning"`
	Success   string `json:"success"`
}

var defaultConfig = Config{
	MaxHistory:     1000,
	Prompt:         "\\u@\\h:\\w$ ",
	DefaultEditor:  "vim",
	AutoComplete:   true,
	PluginsEnabled: true,
	ColorScheme: ColorScheme{
			Prompt:    "\033[1;32m", // green
			Command:   "\033[0m",    // what
			Error:     "\033[1;31m", // red
			Warning:   "\033[1;33m", // yellow
			Success:   "\033[1;32m", // green
	},
}

func Load() (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
			return nil, err
	}

	config := defaultConfig

	config.HistoryFile = filepath.Join(configDir, "history")
	config.AliasFile = filepath.Join(configDir, "aliases")
	config.PluginsDir = filepath.Join(configDir, "plugins")

	configFile := filepath.Join(configDir, "config.json")
	if err := config.loadFromFile(configFile); err != nil {
			if os.IsNotExist(err) {
					if err := config.Save(); err != nil {
							return nil, err
					}
			} else {
					return nil, err
			}
	}

	return &config, nil
}

func (c *Config) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
			return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(c)
}

func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
			return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
	}

	configFile := filepath.Join(configDir, "config.json")
	file, err := os.Create(configFile)
	if err != nil {
			return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return encoder.Encode(c)
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
			return "", err
	}
	return filepath.Join(homeDir, ".gosh"), nil
}
