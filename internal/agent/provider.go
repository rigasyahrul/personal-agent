package agent

import (
	"context"
	"encoding/json"

	"github.com/rigasyahrul/personal-agent/internal/domain"
)

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type ChatRequest struct {
	Model      string           `json:"model"`
	Messages   []ChatMessage    `json:"messages"`
	Tools      []ToolDefinition `json:"tools,omitempty"`
	Parameters map[string]any   `json:"-"`
}

type ChatResponse struct {
	Content   string
	ToolCalls []ToolCall
	Raw       json.RawMessage
}

type Provider interface {
	Chat(context.Context, ChatRequest) (ChatResponse, error)
}

type MessageStore interface {
	List(context.Context, string) ([]domain.Message, error)
	Append(context.Context, domain.Message) error
}

// RunStore is the low-level run surface used by tests and wrappers.
// Production admission goes through store.RunStore.Admit via Runner.Runs (RunAdmissions).
type RunStore interface {
	BeginOrGet(context.Context, string, string) (string, bool, error)
	MarkRunning(context.Context, string) error
	MarkDone(context.Context, string, string, string) error
	ByID(context.Context, string) (domain.AgentRun, error)
}

type SessionReader interface {
	Get(context.Context, string) (domain.Session, error)
}

var workspaceToolDefinitions = []ToolDefinition{
	{Name: "read_file", Description: "Read a regular workspace file", Parameters: objectSchema("path")},
	{Name: "write_file", Description: "Atomically replace a workspace file", Parameters: objectSchema("path", "content")},
	{Name: "edit_file", Description: "Replace one exact occurrence in a workspace file", Parameters: objectSchema("path", "old", "replacement")},
	{Name: "mkdir", Description: "Create workspace directories", Parameters: objectSchema("path")},
}

// knowledgeToolDefinitions are registered for project home even when
// workspace_files is false. No write_knowledge in slice 1.
var knowledgeToolDefinitions = []ToolDefinition{
	{
		Name:        "search_project",
		Description: "Search project knowledge notes",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
				"limit": map[string]any{"type": "integer"},
			},
			"required":             []string{"query"},
			"additionalProperties": false,
		},
	},
	{Name: "read_knowledge", Description: "Read a project knowledge file", Parameters: objectSchema("path")},
	{
		Name:        "list_knowledge",
		Description: "List project knowledge files",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"additionalProperties": false,
		},
	},
}

func objectSchema(required ...string) map[string]any {
	properties := make(map[string]any, len(required))
	for _, name := range required {
		properties[name] = map[string]any{"type": "string"}
	}
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}
