package cli

import (
	"os"

	"github.com/cedricahenkorah/trail/internal/cli/issue"
	"github.com/cedricahenkorah/trail/internal/cli/log"
	"github.com/cedricahenkorah/trail/internal/cli/project"
	"github.com/cedricahenkorah/trail/internal/cli/task"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "trail",
	Short: "Trail CLI",
	Long:  `Trail is a local CLI for tracking work across your projects. Capture devlog notes, manage tasks and issues in Markdown, and see what needs attention today. When it’s time for a standup, Trail brings together your recorded activity and optional Git commits to help you draft an update that you review and finalize yourself.`,
}

// Execute runs the trail command. version is shown by trail --version.
func Execute(version string) {
	rootCmd.Version = version

	addHints(rootCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(issue.NewCommand())
	rootCmd.AddCommand(log.NewCommand())
	rootCmd.AddCommand(project.NewCommand())
	rootCmd.AddCommand(task.NewCommand())
}
