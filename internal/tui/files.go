package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/bubbles/list"
)

var tabs = []tab{logTab, tasksTab, issuesTab, todayTab}

// fileItem is a file row in the file list.
type fileItem struct {
	file    core.FileSummary
	section core.Section
}

func (i fileItem) FilterValue() string { return i.file.Name }
func (i fileItem) title() string       { return i.file.Name }

// detail is the entry count for a log file, or open/total items otherwise.
func (i fileItem) detail() string {
	if i.section != core.Log {
		return fmt.Sprintf("%d/%d", i.file.Open, i.file.Total)
	}

	if i.file.Total == 1 {
		return "1 entry"
	}

	return fmt.Sprintf("%d entries", i.file.Total)
}

func newFileList() list.Model {
	return newList("file", "files")
}

// fileItems returns files as rows, newest first for the log.
func fileItems(files []core.FileSummary, section core.Section) []list.Item {
	items := make([]list.Item, 0, len(files))

	for _, file := range files {
		items = append(items, fileItem{file: file, section: section})
	}

	if section == core.Log {
		slices.Reverse(items)
	}

	return items
}

// selectedFileName returns the name of the file under the cursor. ok is
// false if the list is empty or a filter matches nothing.
func selectedFileName(l list.Model) (name string, ok bool) {
	item, ok := l.SelectedItem().(fileItem)

	return item.file.Name, ok
}

// fileIndex returns the position of the file named name in l, or -1.
func fileIndex(l list.Model, name string) int {
	for i, item := range l.Items() {
		if item.(fileItem).file.Name == name {
			return i
		}
	}

	return -1
}

// tabBar draws the tab names with active highlighted.
func tabBar(active tab, focused bool) string {
	names := make([]string, 0, len(tabs))

	for _, t := range tabs {
		switch {
		case t == active && focused:
			names = append(names, selectedStyle.Render(t.String()))
		case t == active:
			names = append(names, titleStyle.Render(t.String()))
		default:
			names = append(names, mutedStyle.Render(t.String()))
		}
	}

	return strings.Join(names, "  ")
}
