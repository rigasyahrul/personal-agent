package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebContainsPromoteAndReviewContracts(t *testing.T) {
	tests := map[string][]string{
		"../../web/src/components/sessions/PromoteDialog.svelte":  {"Save to source", "target_relative_path", "review_mode"},
		"../../web/src/lib/api/index.ts":                          {"operation_id", "scope="},
		"../../web/src/components/review/ReviewRunner.svelte":     {"project:", "caught_up", "row_version", "duration_ms"},
		"../../web/src/components/sessions/OperationBadges.svelte": {"Promoting…", "Promote failed — Retry", "Note saved; cards pending…", "Cards failed — Retry cards", "Ready"},
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
