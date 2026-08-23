package knowledge

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Wikilink is a path-based [[target|alias]] match.
type Wikilink struct {
	RawTarget      string
	Alias          string
	NormalizedPath string // scope-root WITH .md
}

// Canonical path-wikilink matcher. Heading fragments (#) are not captured.
var wikilinkRE = regexp.MustCompile(`\[\[([^\]|#]+)(?:\|([^\]]+))?\]\]`)

// ParseWikilinks returns path wikilinks from body. Invalid targets are skipped
// so one hostile [[../x]] does not fail the whole parse.
func ParseWikilinks(body string) []Wikilink {
	matches := wikilinkRE.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]Wikilink, 0, len(matches))
	for _, m := range matches {
		raw := strings.TrimSpace(m[1])
		normalized, err := NormalizeWikilinkTarget(raw)
		if err != nil {
			continue
		}
		out = append(out, Wikilink{
			RawTarget:      raw,
			Alias:          m[2],
			NormalizedPath: normalized,
		})
	}
	return out
}

// NormalizeWikilinkTarget trims target and returns the scope-root join key
// (always with .md). Rejects empty, NUL, "..", and absolute paths.
// Bare AGENTS|SOUL|SYSTEM map to AGENTS.md|SOUL.md|SYSTEM.md.
// Existing .md suffixes are kept (never stripped).
func NormalizeWikilinkTarget(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("empty wikilink target")
	}
	if strings.ContainsRune(target, 0) {
		return "", fmt.Errorf("wikilink target contains NUL")
	}
	if strings.Contains(target, "..") {
		return "", fmt.Errorf("wikilink target must not contain ..")
	}
	if strings.HasPrefix(target, "/") || filepath.IsAbs(target) || filepath.VolumeName(target) != "" {
		return "", fmt.Errorf("wikilink target must be relative")
	}
	switch target {
	case "AGENTS", "SOUL", "SYSTEM":
		return target + ".md", nil
	}
	if !strings.HasSuffix(target, ".md") {
		target += ".md"
	}
	return target, nil
}
