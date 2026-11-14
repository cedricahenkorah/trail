package cli

import (
	"fmt"
	"time"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/spf13/cobra"
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's devlog, open tasks, and unresolved issues",
	Long: `Show a project's work at a glance: today's devlog entries, every
	open task across all task files, and every unresolved issue. Tasks
	with a due date are labeled as overdue, due today, or upcoming.

	Without --project, uses the active project linked to the current
	directory, or prompts you to choose one. This command never changes
	any files.`,
	RunE: runTodayCommand,
	Args: cobra.NoArgs,
}

func init() {
	rootCmd.AddCommand(todayCmd)
	todayCmd.Flags().StringP("project", "p", "", "select an active project by name")
}

func runTodayCommand(cmd *cobra.Command, args []string) error {
	cfg, _, configLoadErr := core.LoadConfig()

	if configLoadErr != nil {
		return configLoadErr
	}

	c, openErr := core.Open(cfg.DataDirectory)

	if openErr != nil {
		return openErr
	}

	projects, projectsErr := c.Projects()

	if projectsErr != nil {
		return projectsErr
	}

	matchedProject, projectGiven, projectErr := prompt.ProjectFlag(cmd, c, "viewing today's summary")

	if projectErr != nil {
		return projectErr
	}

	if !projectGiven {
		resolvedProject, resolveErr := prompt.ResolveProject(projects, true)

		if resolveErr != nil {
			return resolveErr
		}

		matchedProject = resolvedProject
	}

	today := time.Now().Format("2006-01-02")

	summary, summaryErr := c.Today(matchedProject, today)

	if summaryErr != nil {
		return summaryErr
	}

	cmd.Printf("%s · %s\n\n", matchedProject.Name, today)

	cmd.Println("Devlog")

	if len(summary.Devlog) == 0 {
		cmd.Println("  No entries yet. Add one with trail log add.")
	}

	for _, entry := range summary.Devlog {
		cmd.Printf("  %s\n", entry)
	}

	cmd.Println()

	printOpenItems(cmd, "Open tasks", summary.Tasks, func(item core.Item) string {
		return formatDueLabel(item, today)
	})

	cmd.Println()
	printOpenItems(cmd, "Unresolved issues", summary.Issues, func(item core.Item) string {
		return item.Text
	})

	return nil
}

func printOpenItems(cmd *cobra.Command, heading string, files []core.ChecklistFile, format func(core.Item) string) {
	count := 0

	for _, file := range files {
		count += len(file.Items)
	}

	cmd.Printf("%s (%d)\n", heading, count)

	if count == 0 {
		cmd.Println("  None.")
		return
	}

	for _, file := range files {
		cmd.Printf("  %s\n", file.Name)

		for _, item := range file.Items {
			cmd.Printf("    %s\n", format(item))
		}
	}
}

func formatDueLabel(item core.Item, today string) string {
	rest, _, _ := core.ParseDue(item.Text)

	switch item.DueStatus(today) {
	case core.Overdue:
		return fmt.Sprintf("%s (overdue: %s)", rest, item.Due)
	case core.DueToday:
		return rest + " (due today)"
	case core.Upcoming:
		return fmt.Sprintf("%s (due %s)", rest, item.Due)
	default:
		return item.Text
	}
}
