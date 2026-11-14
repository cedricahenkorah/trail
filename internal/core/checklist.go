package core

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Item is one "- [ ]" or "- [x]" line in a checklist file.
type Item struct {
	Text string
	Done bool
	// Due is the date from a trailing @due(YYYY-MM-DD), or empty.
	Due string
	// MarkerOffset is the byte offset of the space or x inside the brackets.
	MarkerOffset int
}

// ChecklistFile is a checklist file and its items.
type ChecklistFile struct {
	Name  string
	Items []Item
}

// ParseItems returns the checklist items in content, skipping ones with no text.
func ParseItems(content []byte) []Item {
	var items []Item
	offset := 0

	for _, rawLine := range bytes.SplitAfter(content, []byte("\n")) {
		lineOffset := offset
		offset += len(rawLine)

		line := strings.TrimSuffix(string(rawLine), "\n")
		line = strings.TrimSuffix(line, "\r")

		isItem := strings.HasPrefix(line, "- [ ] ") ||
			strings.HasPrefix(line, "- [x] ") ||
			strings.HasPrefix(line, "- [X] ")

		if !isItem {
			continue
		}

		text := line[6:]

		if strings.TrimSpace(text) == "" {
			continue
		}

		_, due, _ := ParseDue(text)

		items = append(items, Item{
			Text:         text,
			Done:         line[3] != ' ',
			Due:          due,
			MarkerOffset: lineOffset + 3,
		})
	}

	return items
}

// readChecklistFiles parses every Markdown file in dir.
func readChecklistFiles(dir string) ([]ChecklistFile, error) {
	names, namesErr := markdownFiles(dir)

	if namesErr != nil {
		return nil, namesErr
	}

	var files []ChecklistFile

	for _, name := range names {
		path := filepath.Join(dir, name)

		content, readErr := os.ReadFile(path)

		if readErr != nil {
			return nil, fmt.Errorf("read file %s: %w", path, readErr)
		}

		files = append(files, ChecklistFile{Name: name, Items: ParseItems(content)})
	}

	return files, nil
}

// listChecklistFiles summarizes the open and total items in each file in dir.
func listChecklistFiles(dir string) ([]FileSummary, error) {
	files, readErr := readChecklistFiles(dir)

	if readErr != nil {
		return nil, readErr
	}

	var summaries []FileSummary

	for _, file := range files {
		open := 0

		for _, item := range file.Items {
			if !item.Done {
				open++
			}
		}

		summaries = append(summaries, FileSummary{Name: file.Name, Open: open, Total: len(file.Items)})
	}

	return summaries, nil
}

// OpenItems returns files with only their open items, dropping files with none.
func OpenItems(files []ChecklistFile) []ChecklistFile {
	var openFiles []ChecklistFile

	for _, file := range files {
		var openItems []Item

		for _, item := range file.Items {
			if !item.Done {
				openItems = append(openItems, item)
			}
		}

		if len(openItems) > 0 {
			openFiles = append(openFiles, ChecklistFile{Name: file.Name, Items: openItems})
		}
	}

	return openFiles
}

// AddItem adds an open item to fileName in p's Tasks or Issues section and
// returns the file's path. A new file gets date front matter and a heading.
func (c *Core) AddItem(p Project, section Section, fileName, today, text string) (string, error) {
	dir, dirErr := c.projectSectionDir(p, section)

	if dirErr != nil {
		return "", dirErr
	}

	mkdirErr := os.MkdirAll(dir, 0700)

	if mkdirErr != nil {
		return "", fmt.Errorf("create %s dir: %w", section, mkdirErr)
	}

	filePath := filepath.Join(dir, fileName)
	heading := strings.TrimSuffix(fileName, ".md")
	header := fmt.Sprintf("---\ndate: %s\n---\n\n# %s\n\n", today, heading)

	appendErr := appendEntry(filePath, header, fmt.Sprintf("- [ ] %s\n", text))

	if appendErr != nil {
		return "", appendErr
	}

	return filePath, nil
}

// SetDone marks the items at doneIndexes done and the rest open, changing
// only their markers. It reports whether the file at path was rewritten.
func SetDone(path string, content []byte, items []Item, doneIndexes []int) (bool, error) {
	updated := bytes.Clone(content)
	changed := false

	for i, item := range items {
		nowDone := slices.Contains(doneIndexes, i)

		if nowDone == item.Done {
			continue
		}

		if nowDone {
			updated[item.MarkerOffset] = 'x'
		} else {
			updated[item.MarkerOffset] = ' '
		}

		changed = true
	}

	if !changed {
		return false, nil
	}

	writeErr := writeFileAtomic(path, updated, 0600)

	if writeErr != nil {
		return false, fmt.Errorf("save file %s: %w", path, writeErr)
	}

	return true, nil
}
