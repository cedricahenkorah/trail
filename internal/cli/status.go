/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cli

import (
	"github.com/cedricahenkorah/trail/internal/cli/config"
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
	cfg, cfgFilePath, configLoadErr := config.Load()

	if configLoadErr != nil {
		return configLoadErr
	}

	dataDirErr := config.ValidateDataDirectory(cfg.DataDirectory)

	if dataDirErr != nil {
		return dataDirErr
	}

	cmd.Printf("Trail status\n\nData directory:   %s\nConfig file:      %s\nCurrent project:  none\n", cfg.DataDirectory, cfgFilePath)
	return nil
}
