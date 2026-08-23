package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rigasyahrul/personal-agent/internal/store"
)

const ToolSearchProject = "search_project"

// ProjectSearcher is the store surface used by search_project.
type ProjectSearcher interface {
	SearchProject(ctx context.Context, projectID, query string, limit int) ([]store.SearchHit, error)
}

// KnowledgeToolHandler executes project knowledge tools. Task 63: search_project only.
// Task 65 wires this into Runner; workspace tools stay on Workspace.Execute.
type KnowledgeToolHandler struct {
	Searcher  ProjectSearcher
	ProjectID string
}

func NewKnowledgeToolHandler(searcher ProjectSearcher, projectID string) *KnowledgeToolHandler {
	return &KnowledgeToolHandler{Searcher: searcher, ProjectID: projectID}
}

func (h *KnowledgeToolHandler) Execute(ctx context.Context, name string, raw json.RawMessage) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch name {
	case ToolSearchProject:
		return h.searchProject(ctx, raw)
	default:
		return nil, fmt.Errorf("knowledge tool %q is not allowed", name)
	}
}

type knowledgeHit struct {
	KnowledgeID  string `json:"knowledge_id"`
	Path         string `json:"path"`
	Title        string `json:"title"`
	Snippet      string `json:"snippet"`
	Kind         string `json:"kind"`
	SourceNoteID string `json:"source_note_id,omitempty"`
}

type searchResult struct {
	Hits []knowledgeHit `json:"hits"`
}

func (h *KnowledgeToolHandler) searchProject(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var a struct {
		Query string `json:"query"`
		Limit *int   `json:"limit"`
	}
	if err := decode(raw, &a); err != nil {
		return nil, err
	}
	if h.Searcher == nil {
		return nil, fmt.Errorf("knowledge searcher is not configured")
	}
	if h.ProjectID == "" {
		return nil, fmt.Errorf("project id required")
	}
	limit := 0
	if a.Limit != nil {
		limit = *a.Limit
	}
	hits, err := h.Searcher.SearchProject(ctx, h.ProjectID, a.Query, limit)
	if err != nil {
		return nil, err
	}
	out := searchResult{Hits: make([]knowledgeHit, 0, len(hits))}
	for _, hit := range hits {
		out.Hits = append(out.Hits, knowledgeHit{
			KnowledgeID:  hit.KnowledgeID,
			Path:         hit.Path,
			Title:        hit.Title,
			Snippet:      hit.Snippet,
			Kind:         string(hit.Kind),
			SourceNoteID: hit.SourceNoteID,
		})
	}
	return json.Marshal(out)
}
