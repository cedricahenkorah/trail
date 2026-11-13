package core

// TodaySummary is a project's devlog for one day plus its open tasks and issues.
type TodaySummary struct {
	Date   string
	Devlog []string
	Tasks  []ChecklistFile
	Issues []ChecklistFile
}

// Today returns p's summary, where today is the current date as YYYY-MM-DD.
func (c *Core) Today(p Project, today string) (TodaySummary, error) {
	summary := TodaySummary{Date: today}

	devlog, devlogErr := c.LogEntries(p, today)

	if devlogErr != nil {
		return TodaySummary{}, devlogErr
	}

	summary.Devlog = devlog

	openTasks, openTasksErr := c.openItems(p, Tasks)

	if openTasksErr != nil {
		return TodaySummary{}, openTasksErr
	}

	summary.Tasks = openTasks

	openIssues, openIssuesErr := c.openItems(p, Issues)

	if openIssuesErr != nil {
		return TodaySummary{}, openIssuesErr
	}

	summary.Issues = openIssues

	return summary, nil
}

func (c *Core) openItems(p Project, section Section) ([]ChecklistFile, error) {
	dir, dirErr := c.projectSectionDir(p, section)

	if dirErr != nil {
		return nil, dirErr
	}

	files, readErr := readChecklistFiles(dir)

	if readErr != nil {
		return nil, readErr
	}

	return OpenItems(files), nil
}
