/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Trail's setup and current directory context",
	Long: `Show Trail's data and config locations, plus the project linked
	to the current directory, if any. Trail can contain multiple projects;
	this command only identifies the one relevant to where you are.`,
	RunE: runStatusCmd,
	Args: cobra.NoArgs,
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runStatusCmd(cmd *cobra.Command, args []string) error {
	var cfg config

	configDir, configDirErr := os.UserConfigDir()

	if configDirErr != nil {
		return fmt.Errorf("find Trail config directory %w", configDirErr)
	}

	trailConfigPath := filepath.Join(configDir, "trail", "config.json")

	configData, configDataErr := os.ReadFile(trailConfigPath)

	if os.IsNotExist(configDataErr) {
		return fmt.Errorf("Trail is not initialized; run trail init")
	}

	if configDataErr != nil {
		return fmt.Errorf("read Trail config: %w", configDataErr)
	}

	if len(bytes.TrimSpace(configData)) == 0 {
		return fmt.Errorf(
			"Trail config is empty at %s; run trail init to repair it",
			trailConfigPath,
		)
	}

	indentErr := json.Unmarshal(configData, &cfg)

	if indentErr != nil {
		return fmt.Errorf("parse Trail config at %s: %w", trailConfigPath, indentErr)
	}

	if cfg.DataDirectory == "" {
		return fmt.Errorf("config has no data directory; run trail init to repair it")
	}

	dataDirInfo, dataDirInfoErr := os.Stat(cfg.DataDirectory)

	if os.IsNotExist(dataDirInfoErr) {
		return fmt.Errorf(
			"configured data directory was not found: %s; check whether it was moved or deleted",
			cfg.DataDirectory,
		)
	}

	if dataDirInfoErr != nil {
		return fmt.Errorf("inspect configured data directory: %w", dataDirInfoErr)
	}

	if !dataDirInfo.IsDir() {
		return fmt.Errorf("configured data directory path is not a directory: %s", cfg.DataDirectory)
	}

	cmd.Printf("Trail status\n\nData directory:   %s\nConfig file:      %s\nCurrent project:  none\n", cfg.DataDirectory, trailConfigPath)
	return nil
}
