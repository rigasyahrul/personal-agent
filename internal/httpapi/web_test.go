package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebContainsPromoteAndReviewContracts(t *testing.T) {
	tests := map[string][]string{
		"../../web/js/pages/sessions.js":           {"Save to source", "target_relative_path", "review_mode", "operation_id"},
		"../../web/js/pages/review.js":             {"project:", "scope=", "caught_up", "row_version", "duration_ms"},
		"../../web/js/components/status-badges.js": {"Promoting…", "Promote failed — Retry", "Note saved; cards pending…", "Cards failed — Retry cards", "Ready"},
	}
	for file, wants := range tests {
		t.Run(filepath.Base(file), func(t *testing.T) {
			body, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range wants {
				if !strings.Contains(string(body), want) {
					t.Errorf("%s missing %q", file, want)
				}
			}
		})
	}
}
