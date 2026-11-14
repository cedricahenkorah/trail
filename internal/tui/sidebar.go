package tui

import (
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/bubbles/list"
)

// projectItem is a project row in the sidebar.
type projectItem struct {
	project core.Project
}

func (i projectItem) FilterValue() string { return i.project.Name }
func (i projectItem) title() string       { return i.project.Name }
func (i projectItem) detail() string      { return "" }

func newProjectList() list.Model {
	return newList("project", "projects")
}

func projectItems(projects []core.Project) []list.Item {
	items := make([]list.Item, 0, len(projects))

	for _, p := range projects {
		items = append(items, projectItem{project: p})
	}

	return items
}

// selectedProject returns the project under the cursor. ok is false if
// the list is empty or a filter matches nothing.
func selectedProject(l list.Model) (p core.Project, ok bool) {
	item, ok := l.SelectedItem().(projectItem)

	return item.project, ok
}

// projectIndex returns the position of the project named name in l, or -1.
func projectIndex(l list.Model, name string) int {
	for i, item := range l.Items() {
		if item.(projectItem).project.Name == name {
			return i
		}
	}

	return -1
}
