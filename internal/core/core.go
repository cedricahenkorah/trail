// Package core reads and writes Trail's data directory: the project
// registry, devlogs, and task and issue files. It contains no UI code,
// so the CLI and the TUI can both build on it.
package core

import (
	"errors"
	"fmt"
	"strings"
)

// Core is a handle to a validated Trail data directory.
type Core struct {
	dataDir string
}

// Open returns a Core for dataDir. It returns an error if dataDir is
// empty, does not exist, or is not a directory.
func Open(dataDir string) (*Core, error) {
	dataDirErr := ValidateDataDirectory(dataDir)

	if dataDirErr != nil {
		return nil, dataDirErr
	}

	return &Core{dataDir: dataDir}, nil
}

// DataDir returns the data directory this Core was opened with.
func (c *Core) DataDir() string {
	return c.dataDir
}

// Errors returned by this package. Callers can check for them with
// [errors.Is].
var (
	ErrNotInitialized   = errors.New("Trail is not initialized; run trail init")
	ErrEmptyConfig      = errors.New("Trail config is empty")
	ErrProjectNotFound  = errors.New("project is not registered")
	ErrProjectArchived  = errors.New("project is archived")
	ErrNoLinkedProject  = errors.New("no project is linked to the current directory")
	ErrNoActiveProjects = errors.New("no active projects available")
)

// AmbiguousProjectError is returned when more than one project is linked
// to the same directory. Names lists the matching projects.
type AmbiguousProjectError struct {
	Names []string
}

func (e *AmbiguousProjectError) Error() string {
	return fmt.Sprintf(
		"multiple projects are linked to the current directory: %s",
		strings.Join(e.Names, ", "),
	)
}
