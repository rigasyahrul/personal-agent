package domain

import (
	"time"

	"github.com/rigasyahrul/personal-agent/internal/layout"
)

type ReviewMode string

type Vault struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Project struct {
	ID        string    `json:"id"`
	VaultID   string    `json:"vault_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Session struct {
	ID                  string             `json:"id"`
	Home                layout.SessionHome `json:"home"`
	VaultID             *string            `json:"vault_id"`
	ProjectID           *string            `json:"project_id"`
	Status              string             `json:"status"`
	Provider            string             `json:"provider"`
	ModelID             string             `json:"model_id"`
	ModelParametersJSON string             `json:"model_parameters_json"`
	ToolGrantsJSON      string             `json:"tool_grants_json"`
	Title               string             `json:"title"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	DeletedAt           *time.Time         `json:"deleted_at"`
}

const (
	MessageRoleSystem    = "system"
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleTool      = "tool"

	MessageStatusPending  = "pending"
	MessageStatusComplete = "complete"
	MessageStatusFailed   = "failed"

	AgentRunStatusQueued    = "queued"
	AgentRunStatusRunning   = "running"
	AgentRunStatusCompleted = "completed"
	AgentRunStatusFailed    = "failed"
	AgentRunStatusCancelled = "cancelled"
)

type Message struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	RunID         *string   `json:"run_id"`
	Sequence      int       `json:"sequence"`
	Role          string    `json:"role"`
	Content       string    `json:"content"`
	ToolCallsJSON *string   `json:"tool_calls_json"`
	ToolCallID    *string   `json:"tool_call_id"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type AgentRun struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	RequestKey  string     `json:"request_key"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Error       *string    `json:"error"`
	CreatedAt   time.Time  `json:"created_at"`
}
