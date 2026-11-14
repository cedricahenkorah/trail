package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ValidateDate returns value if it is a valid YYYY-MM-DD date.
func ValidateDate(value string) (string, error) {
	parsedDate, parsedDateErr := time.Parse("2006-01-02", value)

	if parsedDateErr != nil {
		return "", fmt.Errorf("invalid date %q: use YYYY-MM-DD", value)
	}

	return parsedDate.Format("2006-01-02"), nil
}

// AddLogEntry appends note to p's log for date and returns the log file's path.
func (c *Core) AddLogEntry(p Project, date, note string) (string, error) {
	logDir, logDirErr := c.projectSectionDir(p, Log)

	if logDirErr != nil {
		return "", logDirErr
	}

	mkdirErr := os.MkdirAll(logDir, 0700)

	if mkdirErr != nil {
		return "", fmt.Errorf("create log dir: %w", mkdirErr)
	}

	logFilePath := filepath.Join(logDir, date+".md")

	appendErr := appendEntry(logFilePath, fmt.Sprintf("# %s\n\n", date), fmt.Sprintf("- %s\n", note))

	if appendErr != nil {
		return "", appendErr
	}

	return logFilePath, nil
}

// ParseLogEntries returns the text of each "- " line in a log file.
func ParseLogEntries(content []byte) []string {
	var entries []string

	for line := range strings.SplitSeq(string(content), "\n") {
		line = strings.TrimSuffix(line, "\r")

		if entry, ok := strings.CutPrefix(line, "- "); ok {
			entries = append(entries, entry)
		}
	}

	return entries
}

// LogEntries returns p's log entries for date, or none if there is no log for that day.
func (c *Core) LogEntries(p Project, date string) ([]string, error) {
	logFilePath, pathErr := c.FilePath(p, Log, date+".md")

	if pathErr != nil {
		return nil, pathErr
	}

	content, readErr := os.ReadFile(logFilePath)

	if os.IsNotExist(readErr) {
		return nil, nil
	}

	if readErr != nil {
		return nil, fmt.Errorf("read log file %s: %w", logFilePath, readErr)
	}

	return ParseLogEntries(content), nil
}
