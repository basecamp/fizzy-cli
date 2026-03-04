// Package config handles configuration loading for the Fizzy CLI.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultAPIURL is the default Fizzy API URL.
	DefaultAPIURL = "https://app.fizzy.do"

	// LocalConfigFile is the name of the local project config file.
	LocalConfigFile = ".fizzy.yaml"
)

// testConfigDir is used to override global config directory for testing.
// When set, all global config operations use this directory instead of ~/.fizzy
var testConfigDir string

// testWorkingDir is used to override working directory for testing local config.
var testWorkingDir string

// SetTestConfigDir sets a custom global config directory for testing.
func SetTestConfigDir(dir string) {
	testConfigDir = dir
}

// ResetTestConfigDir resets the global config directory to default.
func ResetTestConfigDir() {
	testConfigDir = ""
}

// SetTestWorkingDir sets a custom working directory for testing local config.
func SetTestWorkingDir(dir string) {
	testWorkingDir = dir
}

// ResetTestWorkingDir resets the working directory to default.
func ResetTestWorkingDir() {
	testWorkingDir = ""
}

// Config holds the CLI configuration.
type Config struct {
	Token   string `yaml:"token"`
	Account string `yaml:"account"`
	APIURL  string `yaml:"api_url"`
	Board   string `yaml:"board"`
}

// globalConfigPaths returns the possible global configuration file paths in order of preference.
func globalConfigPaths() []string {
	if testConfigDir != "" {
		return []string{filepath.Join(testConfigDir, "config.yaml")}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "fizzy", "config.yaml"),
		filepath.Join(home, ".fizzy", "config.yaml"),
	}
}

// findLocalConfig walks up the directory tree looking for .fizzy.yaml
func findLocalConfig() string {
	var startDir string
	if testWorkingDir != "" {
		startDir = testWorkingDir
	} else {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}

	dir := startDir
	for {
		configPath := filepath.Join(dir, LocalConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return ""
}

// Load loads configuration from files, environment variables, and defaults.
// Priority (highest to lowest): flags > env vars > local config > global config > defaults
//
// Local config (.fizzy.yaml) is searched for in the current directory and parent
// directories. Values from local config override global config values.
func Load() *Config {
	cfg := &Config{
		APIURL: DefaultAPIURL,
	}

	// Load from global config file first
	for _, path := range globalConfigPaths() {
		if data, err := os.ReadFile(path); err == nil {
			_ = yaml.Unmarshal(data, cfg)
			break
		}
	}

	// Override with local config (walks up directory tree)
	if localPath := findLocalConfig(); localPath != "" {
		if data, err := os.ReadFile(localPath); err == nil {
			var localCfg Config
			if yaml.Unmarshal(data, &localCfg) == nil {
				// Only override non-empty values from local config
				if localCfg.Token != "" {
					cfg.Token = localCfg.Token
				}
				if localCfg.Account != "" {
					cfg.Account = localCfg.Account
				}
				if localCfg.APIURL != "" {
					cfg.APIURL = localCfg.APIURL
				}
				if localCfg.Board != "" {
					cfg.Board = localCfg.Board
				}
			}
		}
	}

	// Override with environment variables
	if token := os.Getenv("FIZZY_TOKEN"); token != "" {
		cfg.Token = token
	}
	if account := os.Getenv("FIZZY_ACCOUNT"); account != "" {
		cfg.Account = account
	}
	if apiURL := os.Getenv("FIZZY_API_URL"); apiURL != "" {
		cfg.APIURL = apiURL
	}
	if board := os.Getenv("FIZZY_BOARD"); board != "" {
		cfg.Board = board
	}

	ensureAPIURL(cfg)
	return cfg
}

// LoadGlobal loads configuration only from the global config file(s) and defaults.
// It does not apply local project config or environment variables.
func LoadGlobal() *Config {
	cfg := &Config{
		APIURL: DefaultAPIURL,
	}
	for _, path := range globalConfigPaths() {
		if data, err := os.ReadFile(path); err == nil {
			_ = yaml.Unmarshal(data, cfg)
			break
		}
	}
	ensureAPIURL(cfg)
	return cfg
}

func ensureAPIURL(cfg *Config) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.APIURL) == "" {
		cfg.APIURL = DefaultAPIURL
	}
}

// ConfigPath returns the path to the global config file (creating directory if needed).
func ConfigPath() (string, error) {
	paths := globalConfigPaths()
	if len(paths) == 0 {
		return "", fmt.Errorf("unable to determine config path")
	}

	// Prefer an existing config file path (so we don't end up with multiple global configs).
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return "", err
			}
			return path, nil
		}
	}

	// Otherwise, use the preferred path.
	preferred := paths[0]
	if err := os.MkdirAll(filepath.Dir(preferred), 0700); err != nil {
		return "", err
	}
	return preferred, nil
}

// Save saves the configuration to the global config file.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// Delete removes the global config file.
func Delete() error {
	for _, path := range globalConfigPaths() {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

// Exists checks if a global config file exists.
func Exists() bool {
	for _, path := range globalConfigPaths() {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// LocalConfigPath returns the path to the local config file if found.
// Returns empty string if no local config exists.
func LocalConfigPath() string {
	return findLocalConfig()
}

// SaveLocal saves the configuration to a local .fizzy.yaml file in the current directory.
func (c *Config) SaveLocal() error {
	var dir string
	if testWorkingDir != "" {
		dir = testWorkingDir
	} else {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	path := filepath.Join(dir, LocalConfigFile)
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
