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

// Open validates dataDir and returns a Core for it.
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

var (
	ErrNotInitialized   = errors.New("Trail is not initialized")
	ErrProjectNotFound  = errors.New("project is not registered")
	ErrProjectArchived  = errors.New("project is archived")
	ErrNoLinkedProject  = errors.New("no project is linked to the current directory")
	ErrNoActiveProjects = errors.New("no active projects available")
)

// AmbiguousProjectError is returned when more than one project is linked
// to the same directory.
type AmbiguousProjectError struct {
	Names []string
}

func (e *AmbiguousProjectError) Error() string {
	return fmt.Sprintf(
		"multiple projects are linked to the current directory: %s",
		strings.Join(e.Names, ", "),
	)
}
