package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/knowledge"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

// UpsertKnowledgeInput indexes one knowledge markdown document by scope+path.
type UpsertKnowledgeInput struct {
	Kind         domain.KnowledgeKind
	ProjectID    string
	VaultID      string
	IsGlobal     bool
	RelativePath string
	Content      []byte
	Status       string // ready
	SourceNoteID string // optional; set for source mirror
}

// KnowledgeStore upserts knowledge_notes and reindexes note_links.
// It does not write files or update FTS (those are later tasks).
type KnowledgeStore struct {
	DB    *sql.DB
	Clock clock.Clock
}

const knowledgeNoteSelect = `
	id, relative_path, coalesce(title,''), kind, coalesce(project_id,''), coalesce(vault_id,''),
	is_global, coalesce(source_note_id,''), coalesce(content_sha256,''), coalesce(byte_size,0),
	coalesce(frontmatter_json,''), status, created_at, updated_at`

func (s *KnowledgeStore) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

// UpsertFromContent hashes content, stores the knowledge_notes row (unique by
// scope+path), replaces outbound note_links, and resolves inbound links in the
// same scope. knowledge_notes.id is always a new UUID on insert — never notes.id.
func (s *KnowledgeStore) UpsertFromContent(ctx context.Context, in UpsertKnowledgeInput) (domain.KnowledgeNote, error) {
	if err := paths.ValidateKnowledgeRelPath(in.RelativePath); err != nil {
		return domain.KnowledgeNote{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := paths.ValidateMarkdownBody(in.Content); err != nil {
		return domain.KnowledgeNote{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := validateKnowledgeKind(in.Kind); err != nil {
		return domain.KnowledgeNote{}, err
	}
	scopeSQL, scopeArgs, err := knowledgeScopeWhere(in.ProjectID, in.VaultID, in.IsGlobal)
	if err != nil {
		return domain.KnowledgeNote{}, err
	}

	fm, body, err := knowledge.SplitFrontmatter(string(in.Content))
	if err != nil {
		return domain.KnowledgeNote{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	title := knowledge.TitleOrStem(fm, in.RelativePath)
	frontmatterJSON, err := marshalFrontmatterJSON(fm)
	if err != nil {
		return domain.KnowledgeNote{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	status := in.Status
	if status == "" {
		status = "ready"
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(in.Content))
	byteSize := int64(len(in.Content))
	now := s.now()

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.KnowledgeNote{}, err
	}
	defer func() { _ = tx.Rollback() }()

	existingID, err := findKnowledgeNoteID(ctx, tx, scopeSQL, scopeArgs, in.RelativePath)
	if err != nil {
		return domain.KnowledgeNote{}, err
	}

	var note domain.KnowledgeNote
	if existingID == "" {
		note = domain.KnowledgeNote{
			ID:              ids.NewID(),
			RelativePath:    in.RelativePath,
			Title:           title,
			Kind:            in.Kind,
			ProjectID:       in.ProjectID,
			VaultID:         in.VaultID,
			IsGlobal:        in.IsGlobal,
			SourceNoteID:    in.SourceNoteID,
			ContentSHA256:   sum,
			ByteSize:        byteSize,
			FrontmatterJSON: frontmatterJSON,
			Status:          status,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO knowledge_notes(
				id, kind, project_id, vault_id, is_global, relative_path, title,
				content_sha256, byte_size, frontmatter_json, status, source_note_id,
				created_at, updated_at
			) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			note.ID, string(note.Kind), nullable(in.ProjectID), nullable(in.VaultID), boolToInt(in.IsGlobal),
			note.RelativePath, nullable(title), note.ContentSHA256, note.ByteSize, nullable(frontmatterJSON),
			note.Status, nullable(in.SourceNoteID), formatTime(note.CreatedAt), formatTime(note.UpdatedAt),
		)
		if err != nil {
			return domain.KnowledgeNote{}, err
		}
	} else {
		var createdAt string
		if err := tx.QueryRowContext(ctx, `SELECT created_at FROM knowledge_notes WHERE id=?`, existingID).Scan(&createdAt); err != nil {
			return domain.KnowledgeNote{}, err
		}
		created, err := parseTime(createdAt)
		if err != nil {
			return domain.KnowledgeNote{}, err
		}
		note = domain.KnowledgeNote{
			ID:              existingID,
			RelativePath:    in.RelativePath,
			Title:           title,
			Kind:            in.Kind,
			ProjectID:       in.ProjectID,
			VaultID:         in.VaultID,
			IsGlobal:        in.IsGlobal,
			SourceNoteID:    in.SourceNoteID,
			ContentSHA256:   sum,
			ByteSize:        byteSize,
			FrontmatterJSON: frontmatterJSON,
			Status:          status,
			CreatedAt:       created,
			UpdatedAt:       now,
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE knowledge_notes SET
				kind=?, title=?, content_sha256=?, byte_size=?, frontmatter_json=?,
				status=?, source_note_id=?, updated_at=?
			WHERE id=?`,
			string(in.Kind), nullable(title), sum, byteSize, nullable(frontmatterJSON),
			status, nullable(in.SourceNoteID), formatTime(now), existingID,
		)
		if err != nil {
			return domain.KnowledgeNote{}, err
		}
	}

	if err := replaceNoteLinks(ctx, tx, note.ID, body, in.ProjectID, in.VaultID, in.IsGlobal, now); err != nil {
		return domain.KnowledgeNote{}, err
	}
	if err := resolveInboundLinks(ctx, tx, note.ID, note.RelativePath, scopeSQL, scopeArgs); err != nil {
		return domain.KnowledgeNote{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.KnowledgeNote{}, err
	}
	return note, nil
}

func (s *KnowledgeStore) ByID(ctx context.Context, id string) (domain.KnowledgeNote, error) {
	if id == "" {
		return domain.KnowledgeNote{}, ErrNotFound
	}
	row := s.DB.QueryRowContext(ctx, `SELECT `+knowledgeNoteSelect+` FROM knowledge_notes WHERE id=?`, id)
	note, err := scanKnowledgeNote(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.KnowledgeNote{}, ErrNotFound
	}
	return note, err
}

func (s *KnowledgeStore) ByScopePath(ctx context.Context, projectID, vaultID string, isGlobal bool, rel string) (domain.KnowledgeNote, error) {
	if err := paths.ValidateKnowledgeRelPath(rel); err != nil {
		return domain.KnowledgeNote{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	scopeSQL, scopeArgs, err := knowledgeScopeWhere(projectID, vaultID, isGlobal)
	if err != nil {
		return domain.KnowledgeNote{}, err
	}
	args := append(append([]any{}, scopeArgs...), rel)
	row := s.DB.QueryRowContext(ctx, `SELECT `+knowledgeNoteSelect+` FROM knowledge_notes WHERE `+scopeSQL+` AND relative_path=?`, args...)
	note, err := scanKnowledgeNote(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.KnowledgeNote{}, ErrNotFound
	}
	return note, err
}

func (s *KnowledgeStore) DeleteLinksFrom(ctx context.Context, fromID string) error {
	if fromID == "" {
		return fmt.Errorf("%w: empty from_note_id", ErrValidation)
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM note_links WHERE from_note_id=?`, fromID)
	return err
}

func replaceNoteLinks(ctx context.Context, tx *sql.Tx, fromID, body, projectID, vaultID string, isGlobal bool, now time.Time) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM note_links WHERE from_note_id=?`, fromID); err != nil {
		return err
	}
	for _, link := range knowledge.ParseWikilinks(body) {
		toNoteID, err := lookupSameScopePath(ctx, tx, projectID, vaultID, isGlobal, link.NormalizedPath)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO note_links(id, from_note_id, raw_target, to_path, to_note_id, created_at)
			VALUES(?,?,?,?,?,?)`,
			ids.NewID(), fromID, link.RawTarget, link.NormalizedPath, nullable(toNoteID), formatTime(now),
		); err != nil {
			return err
		}
	}
	return nil
}

func resolveInboundLinks(ctx context.Context, tx *sql.Tx, noteID, relPath, scopeSQL string, scopeArgs []any) error {
	args := append([]any{noteID, relPath}, scopeArgs...)
	_, err := tx.ExecContext(ctx, `
		UPDATE note_links SET to_note_id=?
		WHERE to_path=? AND from_note_id IN (SELECT id FROM knowledge_notes WHERE `+scopeSQL+`)`,
		args...,
	)
	return err
}

func lookupSameScopePath(ctx context.Context, tx *sql.Tx, projectID, vaultID string, isGlobal bool, rel string) (string, error) {
	scopeSQL, scopeArgs, err := knowledgeScopeWhere(projectID, vaultID, isGlobal)
	if err != nil {
		return "", err
	}
	return findKnowledgeNoteID(ctx, tx, scopeSQL, scopeArgs, rel)
}

func findKnowledgeNoteID(ctx context.Context, tx *sql.Tx, scopeSQL string, scopeArgs []any, rel string) (string, error) {
	if rel == "" {
		return "", nil
	}
	args := append(append([]any{}, scopeArgs...), rel)
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM knowledge_notes WHERE `+scopeSQL+` AND relative_path=?`, args...).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func knowledgeScopeWhere(projectID, vaultID string, isGlobal bool) (string, []any, error) {
	switch {
	case isGlobal:
		if projectID != "" || vaultID != "" {
			return "", nil, fmt.Errorf("%w: global scope excludes project and vault", ErrValidation)
		}
		return "is_global=1", nil, nil
	case projectID != "" && vaultID == "":
		return "project_id=?", []any{projectID}, nil
	case vaultID != "" && projectID == "":
		return "vault_id=? AND is_global=0", []any{vaultID}, nil
	default:
		return "", nil, fmt.Errorf("%w: scope must be project XOR vault XOR global", ErrValidation)
	}
}

func validateKnowledgeKind(kind domain.KnowledgeKind) error {
	switch kind {
	case domain.KnowledgeKindSource, domain.KnowledgeKindMemoryDetail, domain.KnowledgeKindMemoryIndex,
		domain.KnowledgeKindAgents, domain.KnowledgeKindSoul, domain.KnowledgeKindSystem:
		return nil
	default:
		return fmt.Errorf("%w: invalid knowledge kind", ErrValidation)
	}
}

func marshalFrontmatterJSON(fm knowledge.Frontmatter) (string, error) {
	if fm.Raw == nil {
		return "", nil
	}
	b, err := json.Marshal(fm.Raw)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
