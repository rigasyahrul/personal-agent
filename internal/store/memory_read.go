package store

import (
	"errors"
	"os"

	"github.com/rigasyahrul/personal-agent/internal/layout"
)

// ReadLessonsIndex returns the contents of memory/lessons.md under scopeRoot.
// Missing file → "", nil (not an error). Empty file → "", nil.
func ReadLessonsIndex(scopeRoot string) (string, error) {
	body, err := os.ReadFile(layout.LessonsPath(scopeRoot))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(body), nil
}
