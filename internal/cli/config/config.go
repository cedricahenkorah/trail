package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DataDirectory string `json:"data_directory"`
}

func Load() (Config, string, error) {
	var cfg Config
	var configFilePath string

	configDir, configDirErr := os.UserConfigDir()

	if configDirErr != nil {
		return cfg, configFilePath, fmt.Errorf("find Trail config directory: %w", configDirErr)
	}

	configFilePath = filepath.Join(configDir, "trail", "config.json")

	configData, configDataErr := os.ReadFile(configFilePath)

	if os.IsNotExist(configDataErr) {
		return cfg, configFilePath, fmt.Errorf("Trail is not initialized; run trail init")
	}

	if configDataErr != nil {
		return cfg, configFilePath, fmt.Errorf("read Trail config: %w", configDataErr)
	}

	if len(bytes.TrimSpace(configData)) == 0 {
		return cfg, configFilePath, fmt.Errorf(
			"Trail config is empty at %s; run trail init to repair it",
			configFilePath,
		)
	}

	parseErr := json.Unmarshal(configData, &cfg)

	if parseErr != nil {
		return cfg, configFilePath, fmt.Errorf("parse Trail config at %s: %w", configFilePath, parseErr)
	}

	return cfg, configFilePath, nil
}

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
