package core

import (
	"fmt"
	"os"
)

// ValidateDataDirectory reports whether path can be used as Trail's data
// directory. It returns an error if path is empty, does not exist, or is
// not a directory.
func ValidateDataDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("config has no data directory; run trail init to repair it")
	}

	dataDirInfo, dataDirInfoErr := os.Stat(path)

	if os.IsNotExist(dataDirInfoErr) {
		return fmt.Errorf(
			"configured data directory was not found: %s; check whether it was moved or deleted",
			path,
		)
	}

	if dataDirInfoErr != nil {
		return fmt.Errorf("inspect configured data directory: %w", dataDirInfoErr)
	}

	if !dataDirInfo.IsDir() {
		return fmt.Errorf("configured data directory path is not a directory: %s", path)
	}

	return nil
}
