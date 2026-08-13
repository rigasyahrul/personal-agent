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
