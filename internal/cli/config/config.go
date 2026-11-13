package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cedricahenkorah/trail/internal/core"
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

// ValidateDataDirectory has moved to core; this wrapper keeps the CLI
// working until its commands switch to core.Open.
func ValidateDataDirectory(path string) error {
	return core.ValidateDataDirectory(path)
}
