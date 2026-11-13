package core

import (
	"fmt"
	"path/filepath"
	"strings"
)

// LinkedTo returns the projects whose linked directory is dir or one of its parents.
func LinkedTo(projects []Project, dir string) ([]Project, error) {
	var linkedProjects []Project

	for _, registeredProject := range projects {
		if registeredProject.LinkedPath == "" {
			continue
		}

		rel, relErr := filepath.Rel(registeredProject.LinkedPath, dir)

		if relErr != nil {
			return nil, fmt.Errorf("compare directory with project %q: %w", registeredProject.Name, relErr)
		}

		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			linkedProjects = append(linkedProjects, registeredProject)
		}
	}

	return linkedProjects, nil
}

// ResolveProject returns the one active project linked to dir. It returns
// [ErrNoLinkedProject], [ErrProjectArchived], or an [AmbiguousProjectError].
func ResolveProject(projects []Project, dir string) (Project, error) {
	linkedProjects, linkedErr := LinkedTo(projects, dir)

	if linkedErr != nil {
		return Project{}, linkedErr
	}

	activeLinked := ActiveProjects(linkedProjects)

	switch {
	case len(activeLinked) == 1:
		return activeLinked[0], nil

	case len(activeLinked) > 1:
		names := make([]string, 0, len(activeLinked))

		for _, linkedProject := range activeLinked {
			names = append(names, linkedProject.Name)
		}

		return Project{}, &AmbiguousProjectError{Names: names}

	case len(linkedProjects) > 0:
		return Project{}, fmt.Errorf("%w: %s", ErrProjectArchived, linkedProjects[0].Name)

	default:
		return Project{}, ErrNoLinkedProject
	}
}
