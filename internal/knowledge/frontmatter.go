package knowledge

import (
	"fmt"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const maxFrontmatterBytes = 64 * 1024

// Frontmatter is the typed YAML header of a knowledge markdown file.
type Frontmatter struct {
	Title        string
	Date         string
	Tags         []string
	CodifiedInto []string
	Raw          map[string]any
}

// SplitFrontmatter splits a markdown document into YAML frontmatter and body.
// Missing frontmatter returns an empty Frontmatter and the original document.
// A frontmatter block larger than 64KiB, or invalid YAML, fails closed.
func SplitFrontmatter(md string) (Frontmatter, string, error) {
	yamlBlock, body, ok, err := extractFrontmatter(md)
	if err != nil {
		return Frontmatter{}, "", err
	}
	if !ok {
		return Frontmatter{}, md, nil
	}
	fm, err := parseFrontmatter(yamlBlock)
	if err != nil {
		return Frontmatter{}, "", err
	}
	return fm, body, nil
}

// TitleOrStem returns fm.Title when set, otherwise the filename stem of relativePath.
// Example: memory/x.md → x
func TitleOrStem(fm Frontmatter, relativePath string) string {
	if fm.Title != "" {
		return fm.Title
	}
	base := path.Base(relativePath)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	ext := path.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func extractFrontmatter(md string) (yamlBlock, body string, ok bool, err error) {
	rest, ok := trimOpeningFence(md)
	if !ok {
		return "", "", false, nil
	}
	offset := 0
	for {
		if offset > maxFrontmatterBytes {
			return "", "", false, fmt.Errorf("frontmatter exceeds 64KiB")
		}
		line, next, found := nextLine(rest, offset)
		if isFence(line) {
			if offset > maxFrontmatterBytes {
				return "", "", false, fmt.Errorf("frontmatter exceeds 64KiB")
			}
			return rest[:offset], rest[next:], true, nil
		}
		if !found {
			if len(rest) > maxFrontmatterBytes {
				return "", "", false, fmt.Errorf("frontmatter exceeds 64KiB")
			}
			return "", "", false, nil
		}
		offset = next
	}
}

func trimOpeningFence(md string) (string, bool) {
	if !strings.HasPrefix(md, "---") {
		return "", false
	}
	rest := md[3:]
	i := 0
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
		i++
	}
	if i >= len(rest) {
		return "", false
	}
	switch rest[i] {
	case '\n':
		return rest[i+1:], true
	case '\r':
		if i+1 < len(rest) && rest[i+1] == '\n' {
			return rest[i+2:], true
		}
		return rest[i+1:], true
	default:
		return "", false
	}
}

func nextLine(s string, offset int) (line string, next int, found bool) {
	if offset >= len(s) {
		return "", offset, false
	}
	rel := strings.IndexAny(s[offset:], "\r\n")
	if rel < 0 {
		return s[offset:], len(s), false
	}
	line = s[offset : offset+rel]
	next = offset + rel
	if s[next] == '\r' && next+1 < len(s) && s[next+1] == '\n' {
		next += 2
	} else {
		next++
	}
	return line, next, true
}

func isFence(line string) bool {
	return strings.TrimSpace(line) == "---"
}

func parseFrontmatter(block string) (Frontmatter, error) {
	block = strings.TrimSpace(block)
	if block == "" {
		return Frontmatter{}, nil
	}
	raw := map[string]any{}
	if err := yaml.Unmarshal([]byte(block), &raw); err != nil {
		return Frontmatter{}, fmt.Errorf("invalid frontmatter: %w", err)
	}
	fm := Frontmatter{Raw: raw}
	if v, ok := raw["title"]; ok {
		fm.Title = scalarString(v)
	}
	if v, ok := raw["date"]; ok {
		fm.Date = scalarString(v)
	}
	if v, ok := raw["tags"]; ok {
		fm.Tags = stringSlice(v)
	}
	if v, ok := raw["codified_into"]; ok {
		fm.CodifiedInto = stringSlice(v)
	}
	return fm, nil
}

func scalarString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case time.Time:
		if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
			return t.Format("2006-01-02")
		}
		return t.Format(time.RFC3339)
	default:
		return fmt.Sprint(t)
	}
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, scalarString(item))
		}
		return out
	case []string:
		return t
	default:
		s := scalarString(t)
		if s == "" {
			return nil
		}
		return []string{s}
	}
}
