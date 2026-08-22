package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestKnowledgeKindConstants(t *testing.T) {
	kinds := []KnowledgeKind{
		KnowledgeKindSource,
		KnowledgeKindMemoryDetail,
		KnowledgeKindMemoryIndex,
		KnowledgeKindAgents,
		KnowledgeKindSoul,
		KnowledgeKindSystem,
	}
	want := []string{
		"source",
		"memory_detail",
		"memory_index",
		"agents",
		"soul",
		"system",
	}
	if len(kinds) != len(want) {
		t.Fatalf("kind count: got %d want %d", len(kinds), len(want))
	}
	for i, k := range kinds {
		if string(k) != want[i] {
			t.Errorf("kind[%d]: got %q want %q", i, k, want[i])
		}
	}
}

func TestCompoundScopeConstants(t *testing.T) {
	scopes := []CompoundScope{
		CompoundScopeProject,
		CompoundScopeVault,
		CompoundScopeGlobal,
	}
	want := []string{"project", "vault", "global"}
	if len(scopes) != len(want) {
		t.Fatalf("scope count: got %d want %d", len(scopes), len(want))
	}
	for i, s := range scopes {
		if string(s) != want[i] {
			t.Errorf("scope[%d]: got %q want %q", i, s, want[i])
		}
	}
}

func TestCompoundStatusConstants(t *testing.T) {
	statuses := []CompoundStatus{
		CompoundStatusPending,
		CompoundStatusApproved,
		CompoundStatusRejected,
		CompoundStatusFailed,
	}
	want := []string{"pending", "approved", "rejected", "failed"}
	if len(statuses) != len(want) {
		t.Fatalf("status count: got %d want %d", len(statuses), len(want))
	}
	for i, s := range statuses {
		if string(s) != want[i] {
			t.Errorf("status[%d]: got %q want %q", i, s, want[i])
		}
	}
}

func TestKnowledgeNoteJSONRoundTrip(t *testing.T) {
	created := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	in := KnowledgeNote{
		ID:              "kn-1",
		RelativePath:    "memory/detail/foo.md",
		Title:           "Foo",
		Kind:            KnowledgeKindMemoryDetail,
		ProjectID:       "proj-1",
		VaultID:         "vault-1",
		IsGlobal:        false,
		SourceNoteID:    "src-1",
		ContentSHA256:   "abc123",
		ByteSize:        42,
		FrontmatterJSON: `{"tags":["a"]}`,
		Status:          "active",
		CreatedAt:       created,
		UpdatedAt:       updated,
	}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// snake_case keys matching domain style
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	for _, key := range []string{
		"id", "relative_path", "title", "kind", "project_id", "vault_id",
		"is_global", "source_note_id", "content_sha256", "byte_size",
		"frontmatter_json", "status", "created_at", "updated_at",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing JSON key %q in %s", key, raw)
		}
	}

	var out KnowledgeNote
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", out, in)
	}
}

func TestCompoundProposalJSONRoundTrip(t *testing.T) {
	created := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	decided := created.Add(time.Minute)
	finished := created.Add(2 * time.Minute)
	in := CompoundProposal{
		ID:         "cp-1",
		SessionID:  "sess-1",
		Scope:      CompoundScopeProject,
		ProjectID:  "proj-1",
		VaultID:    "vault-1",
		Status:     CompoundStatusPending,
		RequestKey: "rk-1",
		ItemsJSON:  `[]`,
		Error:      "",
		CreatedAt:  created,
		DecidedAt:  &decided,
		FinishedAt: &finished,
	}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	for _, key := range []string{
		"id", "session_id", "scope", "project_id", "vault_id", "status",
		"request_key", "items_json", "error", "created_at", "decided_at", "finished_at",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing JSON key %q in %s", key, raw)
		}
	}

	var out CompoundProposal
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID != in.ID || out.SessionID != in.SessionID || out.Scope != in.Scope ||
		out.ProjectID != in.ProjectID || out.VaultID != in.VaultID || out.Status != in.Status ||
		out.RequestKey != in.RequestKey || out.ItemsJSON != in.ItemsJSON || out.Error != in.Error ||
		!out.CreatedAt.Equal(in.CreatedAt) {
		t.Errorf("round-trip scalar mismatch:\n got %+v\nwant %+v", out, in)
	}
	if out.DecidedAt == nil || !out.DecidedAt.Equal(*in.DecidedAt) {
		t.Errorf("DecidedAt: got %v want %v", out.DecidedAt, in.DecidedAt)
	}
	if out.FinishedAt == nil || !out.FinishedAt.Equal(*in.FinishedAt) {
		t.Errorf("FinishedAt: got %v want %v", out.FinishedAt, in.FinishedAt)
	}
}
