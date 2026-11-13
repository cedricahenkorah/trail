package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cedricahenkorah/trail/internal/cli/checklist"
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

	projectFilter, projectFilterErr := cmd.Flags().GetString("project")

	if projectFilterErr != nil {
		return fmt.Errorf("error reading the project filter: %w", projectFilterErr)
	}

	projectFilter = strings.TrimSpace(projectFilter)

	if cmd.Flags().Changed("project") && projectFilter == "" {
		return fmt.Errorf("--project requires a project name")
	}

	var matchedProject core.Project

	if projectFilter != "" {
		foundProject, findErr := c.Project(projectFilter)

		if findErr != nil {
			return findErr
		}

		if foundProject.Archived {
			return fmt.Errorf("project %q is archived; restore it before viewing today's summary", foundProject.Name)
		}

		matchedProject = foundProject
	} else {
		resolvedProject, resolveErr := prompt.ResolveProject(projects, true)

		if resolveErr != nil {
			return resolveErr
		}

		matchedProject = resolvedProject
	}

	projectDir, projectDirErr := c.ProjectDir(matchedProject)

	if projectDirErr != nil {
		return projectDirErr
	}

	today := time.Now().Format("2006-01-02")

	devlog, devlogErr := readDevlog(projectDir, today)

	if devlogErr != nil {
		return devlogErr
	}

	openTasks, openTasksErr := readOpenItems(projectDir, checklist.Tasks)

	if openTasksErr != nil {
		return openTasksErr
	}

	openIssues, openIssuesErr := readOpenItems(projectDir, checklist.Issues)

	if openIssuesErr != nil {
		return openIssuesErr
	}

	cmd.Printf("%s · %s\n\n", matchedProject.Name, today)

	cmd.Println("Devlog")

	if len(devlog) == 0 {
		cmd.Println("  No entries yet. Add one with trail log add.")
	}

	for _, entry := range devlog {
		cmd.Printf("  %s\n", entry)
	}

	cmd.Println()

	printOpenItems(cmd, "Open tasks", openTasks, func(text string) string {
		return formatDueLabel(text, today)
	})

	cmd.Println()
	printOpenItems(cmd, "Unresolved issues", openIssues, func(text string) string {
		return text
	})

	return nil
}

func readDevlog(projectDir, today string) ([]string, error) {
	logFilePath := filepath.Join(projectDir, "log", today+".md")

	content, readErr := os.ReadFile(logFilePath)

	if os.IsNotExist(readErr) {
		return nil, nil
	}

	if readErr != nil {
		return nil, fmt.Errorf("read log file %s: %w", logFilePath, readErr)
	}

	var entries []string

	for line := range strings.SplitSeq(string(content), "\n") {
		line = strings.TrimSuffix(line, "\r")

		if entry, ok := strings.CutPrefix(line, "- "); ok {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func readOpenItems(projectDir string, kind checklist.Kind) ([]checklist.File, error) {
	dir, dirErr := checklist.ResolveDir(projectDir, kind)

	if dirErr != nil {
		return nil, dirErr
	}

	files, readErr := checklist.ReadFiles(dir)

	if readErr != nil {
		return nil, readErr
	}

	var openFiles []checklist.File

	for _, file := range files {
		var openItems []checklist.Item

		for _, item := range file.Items {
			if !item.Done {
				openItems = append(openItems, item)
			}
		}

		if len(openItems) > 0 {
			openFiles = append(openFiles, checklist.File{Name: file.Name, Items: openItems})
		}
	}

	return openFiles, nil
}

func printOpenItems(cmd *cobra.Command, heading string, files []checklist.File, format func(string) string) {
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
			cmd.Printf("    %s\n", format(item.Text))
		}
	}
}

func formatDueLabel(text, today string) string {
	const prefix = " @due("

	start := strings.LastIndex(text, prefix)

	if start == -1 || !strings.HasSuffix(text, ")") {
		return text
	}

	date := text[start+len(prefix) : len(text)-1]

	if _, parseErr := time.Parse("2006-01-02", date); parseErr != nil {
		return text
	}

	rest := text[:start]

	switch {
	case date < today:
		return fmt.Sprintf("%s (overdue: %s)", rest, date)
	case date == today:
		return rest + " (due today)"
	default:
		return fmt.Sprintf("%s (due %s)", rest, date)
	}
}
