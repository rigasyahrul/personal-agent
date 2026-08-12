package paths

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateRelPath(t *testing.T) {
	t.Run("accepts valid logical POSIX paths", func(t *testing.T) {
		for _, input := range []string{"a", "notes/a.md", "日本語/ノート.md"} {
			got, err := ValidateRelPath(input)
			if err != nil {
				t.Fatalf("ValidateRelPath(%q) error = %v", input, err)
			}
			if got != input {
				t.Errorf("ValidateRelPath(%q) = %q, want unchanged path", input, got)
			}
		}
	})

	t.Run("rejects malformed paths", func(t *testing.T) {
		bad := []string{"", ".", "..", "../a", "/a", "a/../b", "a/./b", "a//b", "a/", "a\x00b", "a\nb", "a\x7fb", string([]byte{0xff})}
		for _, input := range bad {
			_, err := ValidateRelPath(input)
			var pathErr *PathError
			if !errors.As(err, &pathErr) {
				t.Errorf("ValidateRelPath(%q) error = %v, want *PathError", input, err)
				continue
			}
			if pathErr.Code != "invalid_path" || pathErr.Message == "" {
				t.Errorf("ValidateRelPath(%q) error = %#v, want non-empty invalid_path", input, pathErr)
			}
		}
	})

	t.Run("enforces limits", func(t *testing.T) {
		bad := []string{
			strings.Repeat("a", MaxPathBytes+1),
			strings.Repeat("a", MaxComponentBytes+1),
			"a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a",
		}
		for _, input := range bad {
			if _, err := ValidateRelPath(input); err == nil {
				t.Errorf("ValidateRelPath accepted over-limit path %q", input)
			}
		}
	})
}

func TestContractConstants(t *testing.T) {
	if MaxPathBytes != 512 || MaxDepth != 16 || MaxComponentBytes != 255 || MaxMarkdownBytes != 1<<20 {
		t.Fatalf("contract constants changed: path=%d depth=%d component=%d markdown=%d", MaxPathBytes, MaxDepth, MaxComponentBytes, MaxMarkdownBytes)
	}
}

func TestPathErrorError(t *testing.T) {
	err := (&PathError{Code: "invalid_path", Message: "bad path"}).Error()
	if err != "invalid_path: bad path" {
		t.Fatalf("PathError.Error() = %q", err)
	}
}
