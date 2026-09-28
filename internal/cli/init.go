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
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/config"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [data-directory]",
	Short: "Set up a directory for Trail to keep your data",
	Long: `Initialize Trail with a directory for your project notes, tasks, issues,
	and saved standups. Trail stores this data as Markdown files in the
	directory you choose.`,
	RunE: runInitCmd,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runInitCmd(cmd *cobra.Command, args []string) error {
	var cfg config.Config

	configDir, configDirErr := os.UserConfigDir()

	if configDirErr != nil {
		return fmt.Errorf("find Trail config directory %w", configDirErr)
	}

	trailConfigPath := filepath.Join(configDir, "trail", "config.json")

	configData, configDataErr := os.ReadFile(trailConfigPath)

	if configDataErr != nil && !os.IsNotExist(configDataErr) {
		return fmt.Errorf("read trail config: %w", configDataErr)
	}

	if os.IsNotExist(configDataErr) {
		// get the dir name from the arg / prompts
		// validate path
		// create the data directory
		// save the config with that path
		// return

		dataDir, chooseDirErr := chooseDataDirectory(args)

		if chooseDirErr != nil {
			return fmt.Errorf("choose data directory: %w", chooseDirErr)
		}

		prepareDataDirErr := prepareDataDirectory(dataDir)

		if prepareDataDirErr != nil {
			return prepareDataDirErr
		}

		trailConfigDir := filepath.Dir(trailConfigPath)

		trailConfigDirErr := os.MkdirAll(trailConfigDir, 0700)

		if trailConfigDirErr != nil {
			return fmt.Errorf("create Trail config directory: %w", trailConfigDirErr)
		}

		contents, encodeErr := setAndEncodeDataDirConfig(cfg, dataDir)

		if encodeErr != nil {
			return encodeErr
		}

		configFile, openFileErr := os.OpenFile(trailConfigPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)

		if openFileErr != nil {
			return fmt.Errorf("create Trail config file: %w", openFileErr)
		}

		_, writeErr := configFile.Write(contents)
		closeErr := configFile.Close()

		if writeErr != nil {
			removeConfigErr := os.Remove(trailConfigPath)

			if removeConfigErr != nil {
				return fmt.Errorf(
					"write Trail config file: %w; could not remove partial config at %s: %v",
					writeErr, trailConfigPath, removeConfigErr,
				)
			}

			return fmt.Errorf("write Trail config file: %w", writeErr)
		}

		if closeErr != nil {
			removeConfigErr := os.Remove(trailConfigPath)

			if removeConfigErr != nil {
				return fmt.Errorf(
					"close Trail config file: %w; could not remove partial config at %s: %v",
					closeErr, trailConfigPath, removeConfigErr,
				)
			}

			return fmt.Errorf("close Trail config file: %w", closeErr)
		}

		cmd.Printf(
			"Trail is ready! :)\n\nData directory: %s\nConfig file:     %s\n\nNext: run trail --help to see available commands.\n",
			dataDir,
			trailConfigPath,
		)
		return nil
	}

	if len(bytes.TrimSpace(configData)) > 0 {
		jsonParseErr := json.Unmarshal(configData, &cfg)

		if jsonParseErr != nil {
			return fmt.Errorf("parse trail config: %w", jsonParseErr)
		}
	}

	if cfg.DataDirectory == "" {

		// report the issue to the user, empty config
		// ask the user if we should set the data directory - yes, no
		// if no - stop and show the config path
		// if yes - follow the steps in the os.IsNotExist path

		cmd.Printf("Trail found a config file with no data directory:\n %s\n", trailConfigPath)

		var repair bool

		createTempFileErr := huh.NewConfirm().
			Title("Set a data directory now?").
			Value(&repair).
			Run()

		if createTempFileErr != nil {
			return fmt.Errorf("confirm Trail config repair: %w", createTempFileErr)
		}

		if !repair {
			cmd.Printf("Config remains unchanged: %s\n", trailConfigPath)
			return nil
		}

		dataDir, chooseDirErr := chooseDataDirectory(args)

		if chooseDirErr != nil {
			return fmt.Errorf("choose data directory: %w", chooseDirErr)
		}

		prepareDataDirErr := prepareDataDirectory(dataDir)

		if prepareDataDirErr != nil {
			return prepareDataDirErr
		}

		contents, encodeErr := setAndEncodeDataDirConfig(cfg, dataDir)

		if encodeErr != nil {
			return encodeErr
		}

		tempFile, createTempFileErr := os.CreateTemp(filepath.Dir(trailConfigPath), "config-*.tmp")

		if createTempFileErr != nil {
			return fmt.Errorf("create temporary config file: %w", createTempFileErr)
		}

		tempPath := tempFile.Name()

		defer os.Remove(tempPath)

		_, tempFileWriteErr := tempFile.Write(contents)

		if tempFileWriteErr != nil {
			tempFile.Close()
			return fmt.Errorf("write temporary config file: %w", tempFileWriteErr)
		}

		tempFileCloseErr := tempFile.Close()

		if tempFileCloseErr != nil {
			return fmt.Errorf("close temporary config file: %w", tempFileCloseErr)
		}

		renameTempFile := os.Rename(tempPath, trailConfigPath)

		if renameTempFile != nil {
			return fmt.Errorf("replace Trail config file: %w", renameTempFile)
		}

		cmd.Printf("Trail config repaired. Data directory: %s\nConfig file:     %s\n\nNext: run trail --help to see available commands.\n", dataDir, trailConfigPath)
		return nil
	}

	dataDirErr := config.ValidateDataDirectory(cfg.DataDirectory)

	if dataDirErr != nil {
		return dataDirErr
	}

	cmd.Printf("Trail is already initialized. Data directory: %s\nConfig file:     %s\n\nNext: run trail --help to see available commands.\n", cfg.DataDirectory, trailConfigPath)
	return nil
}

func chooseDataDirectory(args []string) (string, error) {
	var dataDir string

	cwd, cwdErr := os.Getwd()

	if cwdErr != nil {
		return "", fmt.Errorf("find current directory %w", cwdErr)
	}

	if len(args) == 1 {
		dataDir = strings.TrimSpace(args[0])

		if dataDir == "" {
			return "", fmt.Errorf("data directory path cannot be empty")
		}
	} else {
		var choice string

		dirSelectErr := huh.NewSelect[string]().
			Title("Where should Trail save its data?").
			Options(
				huh.NewOption(fmt.Sprintf("In this current directory (%s)", cwd), "current"),
				huh.NewOption("At another directory path", "custom"),
			).
			Value(&choice).
			Run()

		if dirSelectErr != nil {
			return "", fmt.Errorf("choose data directory location: %w", dirSelectErr)
		}

		if choice == "custom" {
			var custom string

			customDirPathErr := huh.NewInput().
				Title("Enter the directory path").
				Value(&custom).
				Run()

			if customDirPathErr != nil {
				return "", fmt.Errorf("read data directory path: %w", customDirPathErr)
			}

			dataDir = strings.TrimSpace(custom)

			if dataDir == "" {
				return "", fmt.Errorf("data directory path cannot be empty")
			}
		} else if choice == "current" {
			var dirName string

			chooseDirNameErr := huh.NewInput().
				Title(fmt.Sprintf("Directory name in (%s)", cwd)).
				Placeholder("trail-data").
				Value(&dirName).
				Run()

			if chooseDirNameErr != nil {
				return "", fmt.Errorf("read directory name: %w", chooseDirNameErr)
			}

			dirName = strings.TrimSpace(dirName)

			if dirName == "" {
				dirName = "trail-data"
			}

			if dirName == "." || dirName == ".." || filepath.Base(dirName) != dirName {
				return "", fmt.Errorf("enter a directory name, not a path")
			}

			dataDir = filepath.Join(cwd, dirName)

			dataDir = strings.TrimSpace(dataDir)
		}
	}

	dataDir = strings.TrimSpace(dataDir)

	if dataDir == "" {
		return "", fmt.Errorf("data directory path cannot be empty")
	}

	if dataDir == "~" || strings.HasPrefix(dataDir, "~/") {
		home, homeDirErr := os.UserHomeDir()

		if homeDirErr != nil {
			return "", fmt.Errorf("find home directory: %w", homeDirErr)
		}

		if dataDir == "~" {
			dataDir = home
		} else {
			dataDir = filepath.Join(home, strings.TrimPrefix(dataDir, "~/"))
		}
	}

	absolutePath, absErr := filepath.Abs(dataDir)

	if absErr != nil {
		return "", fmt.Errorf("resolve data directory: %w", absErr)
	}

	dataDir = absolutePath

	return dataDir, nil
}

func prepareDataDirectory(dataDir string) error {
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

func setAndEncodeDataDirConfig(cfg config.Config, dataDir string) ([]byte, error) {
	cfg.DataDirectory = dataDir

	contents, indentErr := json.MarshalIndent(cfg, "", " ")

	if indentErr != nil {
		return nil, fmt.Errorf("encode Trail config: %w", indentErr)
	}

	return append(contents, '\n'), nil
}
