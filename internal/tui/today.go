package tui

import (
	"fmt"
	"strings"

	"github.com/cedricahenkorah/trail/internal/core"
)

// todayMarkdown turns project's summary into Markdown for the content pane.
func todayMarkdown(project string, summary core.TodaySummary) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s · %s\n\n", project, summary.Date)

	b.WriteString("## Devlog\n\n")

	if len(summary.Devlog) == 0 {
		b.WriteString("_No entries yet. Add one with `trail log add`._\n\n")
	}

	for _, entry := range summary.Devlog {
		fmt.Fprintf(&b, "- %s\n", entry)
	}

	if len(summary.Devlog) > 0 {
		b.WriteString("\n")
	}

	writeOpenItems(&b, "Open tasks", summary.Tasks, func(item core.Item) string {
		return dueLabel(item, summary.Date)
	})

	writeOpenItems(&b, "Unresolved issues", summary.Issues, func(item core.Item) string {
		return item.Text
	})

	return b.String()
}

// writeOpenItems writes a heading with the item count, then each file's items.
func writeOpenItems(b *strings.Builder, heading string, files []core.ChecklistFile, format func(core.Item) string) {
	count := 0

	for _, file := range files {
		count += len(file.Items)
	}

	fmt.Fprintf(b, "## %s (%d)\n\n", heading, count)

	if count == 0 {
		b.WriteString("_None._\n\n")
		return
	}

	for _, file := range files {
		fmt.Fprintf(b, "### %s\n\n", file.Name)

		for _, item := range file.Items {
			fmt.Fprintf(b, "- [ ] %s\n", format(item))
		}

		b.WriteString("\n")
	}
}

// dueLabel replaces an item's @due marker with how it compares to today.
func dueLabel(item core.Item, today string) string {
	rest, _, _ := core.ParseDue(item.Text)

	switch item.DueStatus(today) {
	case core.Overdue:
		return fmt.Sprintf("%s **(overdue: %s)**", rest, item.Due)
	case core.DueToday:
		return rest + " **(due today)**"
	case core.Upcoming:
		return fmt.Sprintf("%s _(due %s)_", rest, item.Due)
	default:
		return item.Text
	}
}
