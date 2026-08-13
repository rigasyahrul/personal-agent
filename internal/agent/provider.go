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

type RunStore interface {
	BeginOrGet(context.Context, string, string) (string, bool, error)
	MarkRunning(context.Context, string) error
	MarkDone(context.Context, string, string, string) error
}

type SessionReader interface {
	Get(context.Context, string) (domain.Session, error)
}
