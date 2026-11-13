package prompt

import (
	"errors"
	"fmt"
	"os"

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
