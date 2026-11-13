package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic writes data to a temporary file next to path and then
// renames it over path, so readers see either the old file or the new
// one, never a partial write.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tempFile, createTempErr := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")

	if createTempErr != nil {
		return fmt.Errorf("create temporary file: %w", createTempErr)
	}

	tempPath := tempFile.Name()

	defer os.Remove(tempPath)

	_, writeErr := tempFile.Write(data)

	if writeErr != nil {
		tempFile.Close()
		return fmt.Errorf("write temporary file: %w", writeErr)
	}

	chmodErr := tempFile.Chmod(perm)

	if chmodErr != nil {
		tempFile.Close()
		return fmt.Errorf("set permissions on temporary file: %w", chmodErr)
	}

	closeErr := tempFile.Close()

	if closeErr != nil {
		return fmt.Errorf("close temporary file: %w", closeErr)
	}

	renameErr := os.Rename(tempPath, path)

	if renameErr != nil {
		return fmt.Errorf("replace %s: %w", path, renameErr)
	}

	return nil
}
