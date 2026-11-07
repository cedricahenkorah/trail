package checklist

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Item struct {
	Text         string
	Done         bool
	MarkerOffset int
}

type FileSummary struct {
	Name  string
	Open  int
	Total int
}

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

		items = append(items, Item{
			Text:         text,
			Done:         line[3] != ' ',
			MarkerOffset: lineOffset + 3,
		})
	}

	return items
}

func ListFiles(dir string) ([]FileSummary, error) {
	entries, err := os.ReadDir(dir)

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	var summaries []FileSummary

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			continue
		}

		path := filepath.Join(dir, name)

		content, readErr := os.ReadFile(path)

		if readErr != nil {
			return nil, fmt.Errorf("read file %s: %w", path, readErr)
		}

		items := ParseItems(content)
		open := 0

		for _, item := range items {
			if !item.Done {
				open++
			}
		}

		summaries = append(summaries, FileSummary{Name: name, Open: open, Total: len(items)})
	}

	return summaries, nil
}

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

	if err := os.WriteFile(path, updated, 0600); err != nil {
		return false, fmt.Errorf("save file %s: %w", path, err)
	}

	return true, nil
}
