package paths

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidateKnowledgeRelPath validates scope-root-relative knowledge paths.
// Allow only:
//   - AGENTS.md | SOUL.md | SYSTEM.md
//   - memory/** (.md files, including lessons.md)
//   - source/** (.md files; for index/read — compound must not write)
//
// Reject: empty, .., absolute, .agents/**, sessions/**, soul/** directory targets.
//
// This is intentionally separate from ValidateRelPath, which rejects memory/soul
// components for promote/source safety and must not be loosened.
func ValidateKnowledgeRelPath(rel string) error {
	if rel == "" || !utf8.ValidString(rel) {
		return pathErr("invalid_path", "path must be a non-empty UTF-8 relative POSIX path")
	}
	if strings.HasPrefix(rel, "/") || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || strings.Contains(rel, `\`) {
		return pathErr("invalid_path", "path must be a relative POSIX path")
	}
	if len(rel) > MaxPathBytes {
		return pathErr("path_too_long", "path exceeds 512 bytes")
	}

	// Exact instruction files at scope root.
	switch rel {
	case "AGENTS.md", "SOUL.md", "SYSTEM.md":
		return nil
	}

	parts := strings.Split(rel, "/")
	if len(parts) > MaxDepth {
		return pathErr("path_too_deep", "path exceeds 16 components")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return pathErr("invalid_path", "path contains an unsafe component")
		}
		if len(part) > MaxComponentBytes {
			return pathErr("component_too_long", "path component exceeds 255 bytes")
		}
		for _, r := range part {
			if r == 0 || unicode.IsControl(r) {
				return pathErr("invalid_path", "path contains a control character")
			}
		}
	}

	root := parts[0]
	switch root {
	case "memory", "source":
		// Must be under memory/ or source/ with at least one file segment.
		if len(parts) < 2 {
			return pathErr("invalid_path", "path must be a file under memory/ or source/")
		}
		// Reject reserved/non-knowledge roots disguised as nested names is N/A;
		// block soul/.agents/sessions as first component only (handled below).
		base := parts[len(parts)-1]
		if !strings.HasSuffix(base, ".md") {
			return pathErr("invalid_path", "knowledge paths must end in .md")
		}
		return nil
	case ".agents", "sessions", "soul":
		return pathErr("reserved_path", "path namespace not allowed for knowledge")
	default:
		return pathErr("invalid_path", "path must be instruction file, memory/**, or source/**")
	}
}
