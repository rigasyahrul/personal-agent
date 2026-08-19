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
		cases := []struct {
			input string
			code  string
		}{
			{strings.Repeat("a", MaxPathBytes+1), "path_too_long"},
			{strings.Repeat("a", MaxComponentBytes+1), "component_too_long"},
			{"a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a", "path_too_deep"},
		}
		for _, tc := range cases {
			_, err := ValidateRelPath(tc.input)
			var pe *PathError
			if !errors.As(err, &pe) || pe.Code != tc.code {
				t.Errorf("ValidateRelPath(%q) error = %v, want code %q", tc.input, err, tc.code)
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

func TestValidateRelPathRejectsHostileAndOversizeCorpus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		code string
	}{
		{"parent", "../secret.md", "invalid_path"},
		{"nested parent", "notes/../../secret.md", "invalid_path"},
		{"dot component", "notes/./secret.md", "invalid_path"},
		{"absolute unix", "/etc/passwd.md", "invalid_path"},
		{"absolute windows drive", `C:\secret.md`, "invalid_path"},
		{"windows separator", `notes\secret.md`, "invalid_path"},
		{"empty component", "notes//secret.md", "invalid_path"},
		{"reserved memory", "memory/secret.md", "reserved_path"},
		{"reserved soul", "soul/secret.md", "reserved_path"},
		{"reserved nested memory", "notes/memory/secret.md", "reserved_path"},
		{"control", "notes/secret\x00.md", "invalid_path"},
		{"too many components", strings.Repeat("a/", MaxDepth) + "x.md", "path_too_deep"},
		{"component too long", strings.Repeat("a", MaxComponentBytes+1) + ".md", "component_too_long"},
		{"path too long", strings.Repeat("abc/", 128) + "x.md", "path_too_long"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateRelPath(tc.path)
			var pe *PathError
			if !errors.As(err, &pe) {
				t.Fatalf("ValidateRelPath(%q) error = %v, want PathError", tc.path, err)
			}
			if pe.Code != tc.code {
				t.Fatalf("ValidateRelPath(%q) code = %q, want %q", tc.path, pe.Code, tc.code)
			}
		})
	}
}

func FuzzValidateRelPathNeverReturnsUnsafePath(f *testing.F) {
	for _, seed := range []string{"../x.md", "/x.md", "memory/x.md", "a/b.md", "a//b.md", `a\b.md`, "\x00.md"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		clean, err := ValidateRelPath(input)
		if err != nil {
			return
		}
		if clean == "" || strings.HasPrefix(clean, "/") || strings.Contains(clean, `\`) {
			t.Fatalf("accepted unsafe path %q as %q", input, clean)
		}
		parts := strings.Split(clean, "/")
		if len(parts) > MaxDepth || len(clean) > MaxPathBytes {
			t.Fatalf("accepted over-limit path %q", clean)
		}
		for _, part := range parts {
			if part == "" || part == "." || part == ".." || part == "memory" || part == "soul" || len(part) > MaxComponentBytes {
				t.Fatalf("accepted unsafe component %q in %q", part, clean)
			}
		}
	})
}

func TestValidateMarkdownBodyRejectsOversize(t *testing.T) {
	err := ValidateMarkdownBody([]byte(strings.Repeat("x", MaxMarkdownBytes+1)))
	var pe *PathError
	if !errors.As(err, &pe) || pe.Code != "body_too_large" {
		t.Fatalf("error = %v, want body_too_large PathError", err)
	}
	if err := ValidateMarkdownBody([]byte(strings.Repeat("x", MaxMarkdownBytes))); err != nil {
		t.Fatalf("exact limit rejected: %v", err)
	}
}
