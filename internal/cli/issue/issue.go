package issue

import (
	"github.com/spf13/cobra"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage project issues",
	Long: `Manage project issues stored as Markdown checkboxes.
	Group issues in dated or custom-named files.`,
}

func NewCommand() *cobra.Command {
	issueCmd.AddCommand(addCmd)
	issueCmd.AddCommand(listCmd)
	return issueCmd
}
