package store

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

// BackfillReadySourceNotes upserts a source knowledge mirror + FTS row for each
// ready notes row that has no knowledge_notes mirror yet.
//
// Reads from SourceDir + notes.relative_path (fsroot / ValidateRelPath).
// Indexes at "source/"+notes.relative_path (ValidateKnowledgeRelPath).
// Does not loosen ValidateRelPath.
func (s *KnowledgeStore) BackfillReadySourceNotes(ctx context.Context, dataDir string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("%w: knowledge store not configured", ErrValidation)
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT n.id, n.project_id, n.relative_path, coalesce(p.vault_id, '')
		FROM notes n
		INNER JOIN projects p ON p.id = n.project_id
		WHERE n.status = 'ready'
		ORDER BY n.project_id, n.relative_path, n.id`)
	if err != nil {
		return err
	}
	type readyNote struct{ id, projectID, notesRel, vaultID string }
	var pending []readyNote
	for rows.Next() {
		var n readyNote
		if err := rows.Scan(&n.id, &n.projectID, &n.notesRel, &n.vaultID); err != nil {
			_ = rows.Close()
			return err
		}
		pending = append(pending, n)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	// Close the listing cursor before per-row lookups — SQLite often has MaxOpenConns=1.
	for _, n := range pending {
		if err := s.backfillOneReadyNote(ctx, dataDir, n.id, n.projectID, n.vaultID, n.notesRel); err != nil {
			return err
		}
	}
	return nil
}

func (s *KnowledgeStore) backfillOneReadyNote(ctx context.Context, dataDir, noteID, projectID, vaultID, notesRel string) error {
	knowledgeRel, err := sourceMirrorRelPath(notesRel)
	if err != nil {
		return nil
	}
	if _, err := s.ByScopePath(ctx, projectID, "", false, knowledgeRel); err == nil {
		return nil
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	body, err := readSourceNoteFile(dataDir, vaultID, projectID, notesRel)
	if err != nil {
		return nil
	}
	_, err = s.UpsertFromContent(ctx, UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindSource,
		ProjectID:    projectID,
		RelativePath: knowledgeRel,
		Content:      body,
		Status:       "ready",
		SourceNoteID: noteID,
	})
	return err
}

func sourceMirrorRelPath(notesRel string) (string, error) {
	rel := path.Join("source", notesRel)
	if err := paths.ValidateKnowledgeRelPath(rel); err != nil {
		return "", fmt.Errorf("%w: %v", ErrValidation, err)
	}
	return rel, nil
}

func readSourceNoteFile(dataDir, vaultID, projectID, notesRel string) ([]byte, error) {
	root, err := fsroot.Open(layout.SourceDir(layout.ProjectRoot(dataDir, vaultID, projectID)))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(notesRel, paths.MaxMarkdownBytes)
}
