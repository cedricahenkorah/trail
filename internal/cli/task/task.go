package task

import (
	"fmt"

	"github.com/spf13/cobra"
)

// taskCmd represents the task command
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage project tasks",
	Long: `Manage project tasks stored as Markdown checkboxes.
	Group tasks in dated or custom-named files and set optional due dates.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("task called")
	},
}

func NewCommand() *cobra.Command {
	taskCmd.AddCommand(addCmd)
	return taskCmd
}
