package paths

import (
	"path/filepath"
	"strings"
	"unicode"
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

func pathErr(code, message string) error {
	return &PathError{Code: code, Message: message}
}

func reject(code, message string) (string, error) {
	return "", pathErr(code, message)
}

func ValidateRelPath(p string) (string, error) {
	if p == "" || !utf8.ValidString(p) {
		return reject("invalid_path", "path must be a non-empty UTF-8 relative POSIX path")
	}
	if strings.HasPrefix(p, "/") || filepath.IsAbs(p) || filepath.VolumeName(p) != "" || strings.Contains(p, `\`) {
		return reject("invalid_path", "path must be a relative POSIX path")
	}
	if len(p) > MaxPathBytes {
		return reject("path_too_long", "path exceeds 512 bytes")
	}
	parts := strings.Split(p, "/")
	if len(parts) > MaxDepth {
		return reject("path_too_deep", "path exceeds 16 components")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return reject("invalid_path", "path contains an unsafe component")
		}
		if len(part) > MaxComponentBytes {
			return reject("component_too_long", "path component exceeds 255 bytes")
		}
		if part == "memory" || part == "soul" {
			return reject("reserved_path", "memory and soul are reserved")
		}
		for _, r := range part {
			if r == 0 || unicode.IsControl(r) {
				return reject("invalid_path", "path contains a control character")
			}
		}
	}
	return p, nil
}

func ValidateMarkdownBody(body []byte) error {
	if len(body) > MaxMarkdownBytes {
		return pathErr("body_too_large", "markdown body exceeds 1 MiB")
	}
	return nil
}
