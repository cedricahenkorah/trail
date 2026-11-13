package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic writes via a temp file and rename, so path is never half-written.
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

	syncErr := tempFile.Sync()

	if syncErr != nil {
		tempFile.Close()
		return fmt.Errorf("flush temporary file: %w", syncErr)
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

// dirExists reports whether path is a directory, erroring if it's something
// else. label names it in errors; symlinks are not followed.
func dirExists(path, label string) (bool, error) {
	info, statErr := os.Lstat(path)

	if os.IsNotExist(statErr) {
		return false, nil
	}

	if statErr != nil {
		return false, fmt.Errorf("check %s %s: %w", label, path, statErr)
	}

	if !info.IsDir() {
		return false, fmt.Errorf("%s is not a directory: %s", label, path)
	}

	return true, nil
}

// appendEntry appends entry to path, creating the file with header first if needed.
func appendEntry(path, header, entry string) error {
	newFile, createErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

	if createErr == nil {
		_, writeErr := newFile.WriteString(header + entry)
		closeErr := newFile.Close()

		if writeErr != nil {
			return fmt.Errorf("write %s: %w", path, writeErr)
		}

		if closeErr != nil {
			return fmt.Errorf("close %s: %w", path, closeErr)
		}

		return nil
	}

	if !os.IsExist(createErr) {
		return fmt.Errorf("create %s: %w", path, createErr)
	}

	existingFile, openErr := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0)

	if openErr != nil {
		return fmt.Errorf("open %s: %w", path, openErr)
	}

	info, statErr := existingFile.Stat()

	if statErr != nil {
		existingFile.Close()
		return fmt.Errorf("inspect %s: %w", path, statErr)
	}

	if info.Size() > 0 {
		lastByte := make([]byte, 1)

		_, readErr := existingFile.ReadAt(lastByte, info.Size()-1)

		if readErr != nil {
			existingFile.Close()
			return fmt.Errorf("read %s: %w", path, readErr)
		}

		if lastByte[0] != '\n' {
			entry = "\n" + entry
		}
	}

	_, writeErr := existingFile.WriteString(entry)
	closeErr := existingFile.Close()

	if writeErr != nil {
		return fmt.Errorf("append to %s: %w", path, writeErr)
	}

	if closeErr != nil {
		return fmt.Errorf("close %s: %w", path, closeErr)
	}

	return nil
}
