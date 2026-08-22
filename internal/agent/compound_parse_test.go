package agent

import (
	"strings"
	"testing"
)

func TestParseCompoundItemsFromAssistant_FencedJSON(t *testing.T) {
	content := "Here you go:\n```json\n" +
		`[{"kind":"agents_patch","path":"AGENTS.md","action":"update","content":"# A\n","content_sha256":"deadbeef"}]` +
		"\n```\nthanks"
	items, err := ParseCompoundItemsFromAssistant(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].Kind != "agents_patch" || items[0].Path != "AGENTS.md" || items[0].Action != "update" {
		t.Fatalf("item = %+v", items[0])
	}
	if items[0].Content != "# A\n" {
		t.Fatalf("content = %q", items[0].Content)
	}
	if items[0].ContentSHA256 != "deadbeef" {
		t.Fatalf("parser must keep model hash (server overwrites later); got %q", items[0].ContentSHA256)
	}
}

func TestParseCompoundItemsFromAssistant_RawArray(t *testing.T) {
	content := `[{"kind":"lessons_index_row","path":"memory/lessons.md","action":"update","content":"- row\n"}]`
	items, err := ParseCompoundItemsFromAssistant(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 1 || items[0].Kind != "lessons_index_row" || items[0].Path != "memory/lessons.md" {
		t.Fatalf("items = %+v", items)
	}
}

func TestParseCompoundItemsFromAssistant_Invalid(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
	}{
		{name: "prose", content: "I think we should update AGENTS.md"},
		{name: "object", content: `{"kind":"agents_patch"}`},
		{name: "broken json", content: "```json\n[not-json]\n```"},
		{name: "unclosed fence", content: "```json\n[]"},
		{name: "empty", content: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCompoundItemsFromAssistant(tc.content)
			if err == nil {
				t.Fatalf("content %q: want error", tc.content)
			}
			if strings.TrimSpace(err.Error()) == "" {
				t.Fatal("error message empty")
			}
		})
	}
}
