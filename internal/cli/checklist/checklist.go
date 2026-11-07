package checklist

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/charmbracelet/huh"
)

type Kind string

const (
	Tasks  Kind = "tasks"
	Issues Kind = "issues"
)

func ResolveDir(projectDir string, kind Kind) (string, error) {
	dir := filepath.Join(projectDir, string(kind))

	info, err := os.Lstat(dir)

	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("check %s dir: %w", kind, err)
	}

	if err == nil && !info.IsDir() {
		return "", fmt.Errorf("project %s folder is not a directory: %s", kind, dir)
	}

	return dir, nil
}

func NormalizeFilename(fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("filename cannot be left blank")
	}

	if strings.HasPrefix(fileName, ".") || strings.ContainsAny(fileName, `/\`) {
		return "", fmt.Errorf("filename contains invalid characters")
	}

	ext := filepath.Ext(fileName)

	if ext != "" && ext != ".md" {
		return "", fmt.Errorf("filename must have no extension or end in .md")
	}

	if ext == "" {
		fileName += ".md"
	}

	return fileName, nil
}

func ChooseFile(dir string, kind Kind, today string) (string, error) {
	entries, err := os.ReadDir(dir)

	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s directory %s: %w", kind, dir, err)
	}

	var files []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		files = append(files, entry.Name())
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

	selectErr := prompt.Run(huh.NewSelect[string]().
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

		customErr := prompt.Run(huh.NewInput().
			Title("Enter the file name of your choice").
			Validate(func(s string) error {
				_, normalizeErr := NormalizeFilename(strings.TrimSpace(s))
				return normalizeErr
			}).
			Value(&custom))

		if customErr != nil {
			return "", fmt.Errorf("read file name: %w", customErr)
		}

		fileName, normalizeErr := NormalizeFilename(strings.TrimSpace(custom))

		if normalizeErr != nil {
			return "", normalizeErr
		}

		customPath := filepath.Join(dir, fileName)

		_, checkErr := os.Lstat(customPath)

		if os.IsNotExist(checkErr) {
			return fileName, nil
		}

		if checkErr != nil {
			return "", fmt.Errorf("check file %s: %w", customPath, checkErr)
		}

		var collisionChoice string

		collisionErr := prompt.Run(huh.NewSelect[string]().
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

func AddItem(projectDir string, kind Kind, fileName, today, item string) (string, error) {
	dir, dirErr := ResolveDir(projectDir, kind)

	if dirErr != nil {
		return "", dirErr
	}

	if mkdirErr := os.MkdirAll(dir, 0700); mkdirErr != nil {
		return "", fmt.Errorf("create %s dir: %w", kind, mkdirErr)
	}

	filePath := filepath.Join(dir, fileName)
	entry := fmt.Sprintf("- [ ] %s\n", item)

	file, openErr := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

	if os.IsExist(openErr) {
		existing, readErr := os.ReadFile(filePath)

		if readErr != nil {
			return "", fmt.Errorf("read file %s: %w", filePath, readErr)
		}

		if len(existing) > 0 && existing[len(existing)-1] != '\n' {
			entry = "\n" + entry
		}

		file, openErr = os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, 0)

		if openErr != nil {
			return "", fmt.Errorf("open file %s: %w", filePath, openErr)
		}

		_, writeErr := file.Write([]byte(entry))
		closeErr := file.Close()

		if writeErr != nil {
			return "", fmt.Errorf("append file %s: %w", filePath, writeErr)
		}

		if closeErr != nil {
			return "", fmt.Errorf("close file %s: %w", filePath, closeErr)
		}

		return filePath, nil
	}

	if openErr != nil {
		return "", fmt.Errorf("create file %s: %w", filePath, openErr)
	}

	heading := strings.TrimSuffix(fileName, ".md")
	content := fmt.Sprintf("---\ndate: %s\n---\n\n# %s\n\n%s", today, heading, entry)

	_, writeErr := file.Write([]byte(content))
	closeErr := file.Close()

	if writeErr != nil {
		return "", fmt.Errorf("write file %s: %w", filePath, writeErr)
	}

	if closeErr != nil {
		return "", fmt.Errorf("close file %s: %w", filePath, closeErr)
	}

	return filePath, nil
}
