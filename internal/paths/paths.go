package paths

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	MaxPathBytes      = 512
	MaxDepth          = 16
	MaxComponentBytes = 255
	MaxMarkdownBytes  = 1 << 20
)

type PathError struct {
	Code    string
	Message string
}

func (e *PathError) Error() string {
	return e.Code + ": " + e.Message
}

func reject(code, message string) (string, error) {
	return "", &PathError{Code: code, Message: message}
}

func ValidateRelPath(p string) (string, error) {
	if p == "" || !utf8.ValidString(p) {
		return reject("invalid_path", "path must be non-empty UTF-8")
	}
	if len(p) > MaxPathBytes || strings.HasPrefix(p, "/") {
		return reject("invalid_path", "path is absolute or too long")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return reject("invalid_path", "control characters are forbidden")
		}
	}

	parts := strings.Split(p, "/")
	if len(parts) > MaxDepth {
		return reject("invalid_path", "path is too deep")
	}
	for _, component := range parts {
		if component == "" || component == "." || component == ".." || len(component) > MaxComponentBytes {
			return reject("invalid_path", fmt.Sprintf("invalid component %q", component))
		}
	}

	return p, nil
}
