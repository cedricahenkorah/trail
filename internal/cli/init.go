package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

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
}

func runInitCmd(cmd *cobra.Command, args []string) error {
	cfg, trailConfigPath, loadConfigErr := core.LoadConfig()

	if errors.Is(loadConfigErr, core.ErrNotInitialized) {
		dataDir, chooseDirErr := chooseDataDirectory(args)

		if chooseDirErr != nil {
			return fmt.Errorf("choose data directory: %w", chooseDirErr)
		}

		prepareDataDirErr := core.PrepareDataDirectory(dataDir)

		if prepareDataDirErr != nil {
			return prepareDataDirErr
		}

		cfg.DataDirectory = dataDir

		writeConfigErr := core.WriteConfig(cfg)

		if writeConfigErr != nil {
			return writeConfigErr
		}

		cmd.Printf(
			"Trail is ready! :)\n\nData directory: %s\nConfig file:     %s\n\nNext: run trail --help to see available commands.\n",
			dataDir,
			trailConfigPath,
		)
		return nil
	}

	// An empty config file is repaired below, the same as one with no
	// data directory.
	if loadConfigErr != nil && !errors.Is(loadConfigErr, core.ErrEmptyConfig) {
		return loadConfigErr
	}

	if cfg.DataDirectory == "" {
		cmd.Printf("Trail found a config file with no data directory:\n %s\n", trailConfigPath)

		var repair bool

		confirmRepairErr := prompt.Run(huh.NewConfirm().
			Title("Set a data directory now?").
			Value(&repair))

		if confirmRepairErr != nil {
			return fmt.Errorf("confirm Trail config repair: %w", confirmRepairErr)
		}

		if !repair {
			cmd.Printf("Config remains unchanged: %s\n", trailConfigPath)
			return nil
		}

		dataDir, chooseDirErr := chooseDataDirectory(args)

		if chooseDirErr != nil {
			return fmt.Errorf("choose data directory: %w", chooseDirErr)
		}

		prepareDataDirErr := core.PrepareDataDirectory(dataDir)

		if prepareDataDirErr != nil {
			return prepareDataDirErr
		}

		cfg.DataDirectory = dataDir

		writeConfigErr := core.WriteConfig(cfg)

		if writeConfigErr != nil {
			return writeConfigErr
		}

		cmd.Printf("Trail config repaired. Data directory: %s\nConfig file:     %s\n\nNext: run trail --help to see available commands.\n", dataDir, trailConfigPath)
		return nil
	}

	dataDirErr := core.ValidateDataDirectory(cfg.DataDirectory)

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

		dirSelectErr := prompt.Run(huh.NewSelect[string]().
			Title("Where should Trail save its data?").
			Options(
				huh.NewOption(fmt.Sprintf("In this current directory (%s)", cwd), "current"),
				huh.NewOption("At another directory path", "custom"),
			).
			Value(&choice))

		if dirSelectErr != nil {
			return "", fmt.Errorf("choose data directory location: %w", dirSelectErr)
		}

		if choice == "custom" {
			var custom string

			customDirPathErr := prompt.Run(huh.NewInput().
				Title("Enter the directory path").
				Value(&custom))

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

	return core.ExpandPath(dataDir)
}
