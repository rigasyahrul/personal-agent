package domain

import "time"

type KnowledgeKind string

const (
	KnowledgeKindSource       KnowledgeKind = "source"
	KnowledgeKindMemoryDetail KnowledgeKind = "memory_detail"
	KnowledgeKindMemoryIndex  KnowledgeKind = "memory_index"
	KnowledgeKindAgents       KnowledgeKind = "agents"
	KnowledgeKindSoul         KnowledgeKind = "soul"
	KnowledgeKindSystem       KnowledgeKind = "system"
)

type KnowledgeNote struct {
	ID              string        `json:"id"`
	RelativePath    string        `json:"relative_path"`
	Title           string        `json:"title"`
	Kind            KnowledgeKind `json:"kind"`
	ProjectID       string        `json:"project_id"`
	VaultID         string        `json:"vault_id"`
	IsGlobal        bool          `json:"is_global"`
	SourceNoteID    string        `json:"source_note_id"`
	ContentSHA256   string        `json:"content_sha256"`
	ByteSize        int64         `json:"byte_size"`
	FrontmatterJSON string        `json:"frontmatter_json"`
	Status          string        `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type CompoundScope string

const (
	CompoundScopeProject CompoundScope = "project"
	CompoundScopeVault   CompoundScope = "vault"
	CompoundScopeGlobal  CompoundScope = "global"
)

type CompoundStatus string

const (
	CompoundStatusPending  CompoundStatus = "pending"
	CompoundStatusApproved CompoundStatus = "approved"
	CompoundStatusRejected CompoundStatus = "rejected"
	CompoundStatusFailed   CompoundStatus = "failed"
)

type CompoundProposal struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"session_id"`
	Scope      CompoundScope  `json:"scope"`
	ProjectID  string         `json:"project_id"`
	VaultID    string         `json:"vault_id"`
	Status     CompoundStatus `json:"status"`
	RequestKey string         `json:"request_key"`
	ItemsJSON  string         `json:"items_json"`
	Error      string         `json:"error"`
	CreatedAt  time.Time      `json:"created_at"`
	DecidedAt  *time.Time     `json:"decided_at"`
	FinishedAt *time.Time     `json:"finished_at"`
}
