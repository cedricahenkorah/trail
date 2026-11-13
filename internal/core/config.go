package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is Trail's user configuration, stored at [ConfigPath].
type Config struct {
	DataDirectory string `json:"data_directory"`
}

// ConfigPath returns the path of trail/config.json in the user's config directory.
func ConfigPath() (string, error) {
	configDir, configDirErr := os.UserConfigDir()

	if configDirErr != nil {
		return "", fmt.Errorf("find Trail config directory: %w", configDirErr)
	}

	return filepath.Join(configDir, "trail", "config.json"), nil
}

// LoadConfig returns the config and its path, or [ErrNotInitialized] or
// [ErrEmptyConfig] if the file is missing or empty.
func LoadConfig() (Config, string, error) {
	var cfg Config

	configFilePath, configPathErr := ConfigPath()

	if configPathErr != nil {
		return cfg, "", configPathErr
	}

	configData, configDataErr := os.ReadFile(configFilePath)

	if os.IsNotExist(configDataErr) {
		return cfg, configFilePath, ErrNotInitialized
	}

	if configDataErr != nil {
		return cfg, configFilePath, fmt.Errorf("read Trail config: %w", configDataErr)
	}

	if len(bytes.TrimSpace(configData)) == 0 {
		return cfg, configFilePath, fmt.Errorf("%w at %s; run trail init to repair it", ErrEmptyConfig, configFilePath)
	}

	parseErr := json.Unmarshal(configData, &cfg)

	if parseErr != nil {
		return cfg, configFilePath, fmt.Errorf("parse Trail config at %s: %w", configFilePath, parseErr)
	}

	return cfg, configFilePath, nil
}

// WriteConfig saves cfg, replacing any existing config file.
func WriteConfig(cfg Config) error {
	configFilePath, configPathErr := ConfigPath()

	if configPathErr != nil {
		return configPathErr
	}

	configDirErr := os.MkdirAll(filepath.Dir(configFilePath), 0700)

	if configDirErr != nil {
		return fmt.Errorf("create Trail config directory: %w", configDirErr)
	}

	contents, encodeErr := json.MarshalIndent(cfg, "", " ")

	if encodeErr != nil {
		return fmt.Errorf("encode Trail config: %w", encodeErr)
	}

	contents = append(contents, '\n')

	writeErr := writeFileAtomic(configFilePath, contents, 0600)

	if writeErr != nil {
		return fmt.Errorf("write Trail config: %w", writeErr)
	}

	return nil
}

// ValidateDataDirectory returns an error if path isn't an existing directory.
func ValidateDataDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("config has no data directory; run trail init to repair it")
	}

	dataDirInfo, dataDirInfoErr := os.Stat(path)

	if os.IsNotExist(dataDirInfoErr) {
		return fmt.Errorf(
			"configured data directory was not found: %s; check whether it was moved or deleted",
			path,
		)
	}

	if dataDirInfoErr != nil {
		return fmt.Errorf("inspect configured data directory: %w", dataDirInfoErr)
	}

	if !dataDirInfo.IsDir() {
		return fmt.Errorf("configured data directory path is not a directory: %s", path)
	}

	return nil
}

// PrepareDataDirectory creates dataDir, or fails if it exists and isn't empty.
func PrepareDataDirectory(dataDir string) error {
	dataDirInfo, dirInfoErr := os.Stat(dataDir)

	if dirInfoErr != nil && !os.IsNotExist(dirInfoErr) {
		return fmt.Errorf("inspect data directory: %w", dirInfoErr)
	}

	if dirInfoErr == nil {
		if !dataDirInfo.IsDir() {
			return fmt.Errorf("data directory path is a file: %s", dataDir)
		}

		entries, readDirErr := os.ReadDir(dataDir)

		if readDirErr != nil {
			return fmt.Errorf("read data directory: %w", readDirErr)
		}

		if len(entries) > 0 {
			return fmt.Errorf("data directory is not empty: %s", dataDir)
		}
	}

	dirCreateErr := os.MkdirAll(dataDir, 0700)

	if dirCreateErr != nil {
		return fmt.Errorf("create data directory: %w", dirCreateErr)
	}

	return nil
}

// ExpandPath expands a leading ~ and returns path as an absolute path.
func ExpandPath(path string) (string, error) {
	path = strings.TrimSpace(path)

	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, homeDirErr := os.UserHomeDir()

		if homeDirErr != nil {
			return "", fmt.Errorf("find home directory: %w", homeDirErr)
		}

		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}

	absolutePath, absErr := filepath.Abs(path)

	if absErr != nil {
		return "", fmt.Errorf("resolve path %s: %w", path, absErr)
	}

	return absolutePath, nil
}
