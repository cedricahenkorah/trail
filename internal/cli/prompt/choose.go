package prompt

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
)

// ResolveProject returns the active project linked to the current directory.
// If there isn't exactly one, it asks the user to pick one when interactive.
func ResolveProject(projects []core.Project, interactive bool) (core.Project, error) {
	cwd, cwdErr := os.Getwd()

	if cwdErr != nil {
		return core.Project{}, fmt.Errorf("find current directory %w", cwdErr)
	}

	resolvedProject, resolveErr := core.ResolveProject(projects, cwd)

	if resolveErr == nil {
		return resolvedProject, nil
	}

	var ambiguousErr *core.AmbiguousProjectError

	unresolved := errors.Is(resolveErr, core.ErrNoLinkedProject) ||
		errors.Is(resolveErr, core.ErrProjectArchived) ||
		errors.As(resolveErr, &ambiguousErr)

	if !unresolved {
		return core.Project{}, resolveErr
	}

	if !interactive {
		if errors.Is(resolveErr, core.ErrProjectArchived) {
			return core.Project{}, fmt.Errorf("%w; restore it or pass --project <name>", resolveErr)
		}

		return core.Project{}, fmt.Errorf("%w; pass --project <name>", resolveErr)
	}

	activeProjects := core.ActiveProjects(projects)

	if len(activeProjects) == 0 {
		return core.Project{}, fmt.Errorf("%w; add one with trail project add", core.ErrNoActiveProjects)
	}

	return ChooseProject(activeProjects)
}

// ChooseProject asks the user to pick one of projects.
func ChooseProject(projects []core.Project) (core.Project, error) {
	var chosenProjectName string
	var options []huh.Option[string]

	for _, p := range projects {
		options = append(options, huh.NewOption(p.Name, p.Name))
	}

	selectProjectErr := Run(huh.NewSelect[string]().
		Title("Select a project").
		Options(options...).
		Value(&chosenProjectName))

	if selectProjectErr != nil {
		return core.Project{}, fmt.Errorf("choose project: %w", selectProjectErr)
	}

	for _, p := range projects {
		if p.Name == chosenProjectName {
			return p, nil
		}
	}

	return core.Project{}, fmt.Errorf("selected project %q was not found", chosenProjectName)
}

// ChooseFile asks which file to add to: today's, an existing one, or a new name.
func ChooseFile(existing []core.FileSummary, today string) (string, error) {
	var files []string

	for _, file := range existing {
		files = append(files, file.Name)
	}

	todayFile := today + ".md"

	todayLabel := todayFile + " (today; create new)"

	if slices.Contains(files, todayFile) {
		todayLabel = todayFile + " (today; append)"
	}

	options := []huh.Option[string]{
		huh.NewOption(todayLabel, todayFile),
	}

	for _, file := range files {
		if file == todayFile {
			continue
		}

		options = append(options, huh.NewOption(file, file))
	}

	const customFileChoice = "create-custom"

	options = append(options, huh.NewOption("Create a custom-named file", customFileChoice))

	chosen := todayFile

	selectErr := Run(huh.NewSelect[string]().
		Title("Choose a file to save to").
		Options(options...).
		Value(&chosen))

	if selectErr != nil {
		return "", fmt.Errorf("choose file: %w", selectErr)
	}

	if chosen != customFileChoice {
		return chosen, nil
	}

	for {
		var custom string

		customErr := Run(huh.NewInput().
			Title("Enter the file name of your choice").
			Validate(func(s string) error {
				_, normalizeErr := core.NormalizeFilename(strings.TrimSpace(s))
				return normalizeErr
			}).
			Value(&custom))

		if customErr != nil {
			return "", fmt.Errorf("read file name: %w", customErr)
		}

		fileName, normalizeErr := core.NormalizeFilename(strings.TrimSpace(custom))

		if normalizeErr != nil {
			return "", normalizeErr
		}

		if !slices.Contains(files, fileName) {
			return fileName, nil
		}

		var collisionChoice string

		collisionErr := Run(huh.NewSelect[string]().
			Title(fileName+" already exists").
			Options(
				huh.NewOption("Append to it", "append"),
				huh.NewOption("Choose another name", "rename"),
			).
			Value(&collisionChoice))

		if collisionErr != nil {
			return "", fmt.Errorf("choose how to handle existing file: %w", collisionErr)
		}

		if collisionChoice == "append" {
			return fileName, nil
		}
	}
}
