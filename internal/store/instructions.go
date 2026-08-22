package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

// InstructionName is the API form: soul | system | agents (also accepts SOUL.md etc.).
type InstructionName string

// ScopeMeta describes where an instruction file lives.
type ScopeMeta struct {
	DataDir   string
	Scope     domain.CompoundScope
	ProjectID string
	VaultID   string
}

// InstructionStore reads/writes SOUL.md, SYSTEM.md, AGENTS.md under a scope root
// and upserts knowledge_notes index rows.
type InstructionStore struct {
	DB      *sql.DB
	Clock   clock.Clock
	Barrier MutBarrier
}

// NormalizeInstructionFile maps API or filename forms to the on-disk name and kind.
func NormalizeInstructionFile(name string) (fileName string, kind domain.KnowledgeKind, err error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", "", fmt.Errorf("%w: empty instruction name", ErrValidation)
	}
	// Reject path-like input early (multi-segment, traversal).
	if strings.Contains(n, "/") || strings.Contains(n, `\`) || strings.Contains(n, "..") {
		return "", "", fmt.Errorf("%w: invalid instruction name", ErrValidation)
	}
	switch strings.ToLower(n) {
	case "agents", "agents.md":
		return "AGENTS.md", domain.KnowledgeKindAgents, nil
	case "soul", "soul.md":
		return "SOUL.md", domain.KnowledgeKindSoul, nil
	case "system", "system.md":
		return "SYSTEM.md", domain.KnowledgeKindSystem, nil
	default:
		return "", "", fmt.Errorf("%w: unknown instruction name", ErrValidation)
	}
}

func (s *InstructionStore) withBarrier(fn func() error) error {
	if s.Barrier == nil {
		return fn()
	}
	return s.Barrier.Mutate(fn)
}

// Get reads an instruction file for the given scope. Missing file → ErrNotFound.
// If the file exists but knowledge_notes has no row yet, content is still returned
// with a partial note (empty ID). Index lookup is always scope-keyed (never path+hash alone).
func (s *InstructionStore) Get(ctx context.Context, meta ScopeMeta, name InstructionName) (content string, note domain.KnowledgeNote, err error) {
	fileName, kind, err := NormalizeInstructionFile(string(name))
	if err != nil {
		return "", domain.KnowledgeNote{}, err
	}
	if err := paths.ValidateKnowledgeRelPath(fileName); err != nil {
		return "", domain.KnowledgeNote{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	scopeRoot, projectID, vaultID, isGlobal, err := resolveInstructionScope(meta)
	if err != nil {
		return "", domain.KnowledgeNote{}, err
	}

	root, err := fsroot.Open(scopeRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", domain.KnowledgeNote{}, ErrNotFound
		}
		return "", domain.KnowledgeNote{}, err
	}
	defer root.Close()

	body, err := root.ReadFile(fileName, paths.MaxMarkdownBytes)
	if err != nil {
		if isNotExist(err) || errors.Is(err, fsroot.ErrInvalidPath) {
			return "", domain.KnowledgeNote{}, ErrNotFound
		}
		return "", domain.KnowledgeNote{}, err
	}

	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	note = domain.KnowledgeNote{
		RelativePath:  fileName,
		Kind:          kind,
		ProjectID:     projectID,
		VaultID:       vaultID,
		IsGlobal:      isGlobal,
		ContentSHA256: sum,
		ByteSize:      int64(len(body)),
		Status:        "ready",
	}
	if s.DB != nil {
		if id, ierr := s.findInstructionNoteID(ctx, projectID, vaultID, isGlobal, fileName); ierr == nil && id != "" {
			if indexed, ierr := s.loadKnowledgeNote(ctx, id); ierr == nil {
				// Prefer on-disk content hash/size; keep indexed identity/timestamps.
				indexed.ContentSHA256 = sum
				indexed.ByteSize = int64(len(body))
				indexed.Kind = kind
				indexed.RelativePath = fileName
				note = indexed
			}
		}
	}
	return string(body), note, nil
}

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	// filepath / os.Root wrap
	return strings.Contains(err.Error(), "no such file") || strings.Contains(strings.ToLower(err.Error()), "not found")
}

func (s *InstructionStore) loadKnowledgeNote(ctx context.Context, id string) (domain.KnowledgeNote, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, relative_path, coalesce(title,''), kind, coalesce(project_id,''), coalesce(vault_id,''),
		       is_global, coalesce(source_note_id,''), coalesce(content_sha256,''), coalesce(byte_size,0),
		       coalesce(frontmatter_json,''), status, created_at, updated_at
		FROM knowledge_notes WHERE id=?`, id)
	return scanKnowledgeNote(row)
}

// Put writes an instruction file atomically and upserts knowledge_notes.
// Vault scope is forbidden (no vault SOUL/SYSTEM/AGENTS).
func (s *InstructionStore) Put(ctx context.Context, meta ScopeMeta, name InstructionName, content string) (domain.KnowledgeNote, error) {
	var out domain.KnowledgeNote
	err := s.withBarrier(func() error {
		var e error
		out, e = s.put(ctx, meta, name, content)
		return e
	})
	return out, err
}

func (s *InstructionStore) put(ctx context.Context, meta ScopeMeta, name InstructionName, content string) (domain.KnowledgeNote, error) {
	fileName, kind, err := NormalizeInstructionFile(string(name))
	if err != nil {
		return domain.KnowledgeNote{}, err
	}
	if err := paths.ValidateKnowledgeRelPath(fileName); err != nil {
		return domain.KnowledgeNote{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := paths.ValidateMarkdownBody([]byte(content)); err != nil {
		return domain.KnowledgeNote{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	scopeRoot, projectID, vaultID, isGlobal, err := resolveInstructionScope(meta)
	if err != nil {
		return domain.KnowledgeNote{}, err
	}

	if err := os.MkdirAll(scopeRoot, 0700); err != nil {
		return domain.KnowledgeNote{}, err
	}

	root, err := fsroot.Open(scopeRoot)
	if err != nil {
		return domain.KnowledgeNote{}, err
	}
	defer root.Close()

	body := []byte(content)
	// Single-segment name only — fsroot valid() uses ValidateRelPath which allows AGENTS.md.
	if err := root.WriteFileAtomic(fileName, body, 0600); err != nil {
		return domain.KnowledgeNote{}, err
	}

	now := s.Clock.Now().UTC()
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	byteSize := int64(len(body))
	title := instructionTitle(fileName, content)

	// Upsert knowledge_notes via partial unique indexes.
	existingID, err := s.findInstructionNoteID(ctx, projectID, vaultID, isGlobal, fileName)
	if err != nil {
		return domain.KnowledgeNote{}, err
	}

	var note domain.KnowledgeNote
	if existingID == "" {
		note = domain.KnowledgeNote{
			ID:            ids.NewID(),
			RelativePath:  fileName,
			Title:         title,
			Kind:          kind,
			ProjectID:     projectID,
			VaultID:       vaultID,
			IsGlobal:      isGlobal,
			ContentSHA256: sum,
			ByteSize:      byteSize,
			Status:        "ready",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		_, err = s.DB.ExecContext(ctx, `
			INSERT INTO knowledge_notes(
				id, kind, project_id, vault_id, is_global, relative_path, title,
				content_sha256, byte_size, status, created_at, updated_at
			) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			note.ID, string(note.Kind), nullable(projectID), nullable(vaultID), boolToInt(isGlobal),
			note.RelativePath, nullable(title), note.ContentSHA256, note.ByteSize, note.Status,
			formatTime(note.CreatedAt), formatTime(note.UpdatedAt),
		)
		if err != nil {
			return domain.KnowledgeNote{}, err
		}
	} else {
		note = domain.KnowledgeNote{
			ID:            existingID,
			RelativePath:  fileName,
			Title:         title,
			Kind:          kind,
			ProjectID:     projectID,
			VaultID:       vaultID,
			IsGlobal:      isGlobal,
			ContentSHA256: sum,
			ByteSize:      byteSize,
			Status:        "ready",
			UpdatedAt:     now,
		}
		// Preserve created_at
		var createdAt string
		err = s.DB.QueryRowContext(ctx, `SELECT created_at FROM knowledge_notes WHERE id=?`, existingID).Scan(&createdAt)
		if err != nil {
			return domain.KnowledgeNote{}, err
		}
		note.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return domain.KnowledgeNote{}, err
		}
		_, err = s.DB.ExecContext(ctx, `
			UPDATE knowledge_notes SET
				kind=?, title=?, content_sha256=?, byte_size=?, status='ready', updated_at=?
			WHERE id=?`,
			string(kind), nullable(title), sum, byteSize, formatTime(now), existingID,
		)
		if err != nil {
			return domain.KnowledgeNote{}, err
		}
	}
	// Link reparse deferred to P2 — stub OK.
	return note, nil
}

func resolveInstructionScope(meta ScopeMeta) (scopeRoot, projectID, vaultID string, isGlobal bool, err error) {
	switch meta.Scope {
	case domain.CompoundScopeVault:
		return "", "", "", false, fmt.Errorf("%w: vault scope has no instruction files", ErrValidation)
	case domain.CompoundScopeGlobal:
		if meta.ProjectID != "" || meta.VaultID != "" {
			return "", "", "", false, fmt.Errorf("%w: global instructions must not set project/vault id", ErrValidation)
		}
		return layout.GlobalRoot(meta.DataDir), "", "", true, nil
	case domain.CompoundScopeProject:
		if meta.ProjectID == "" {
			return "", "", "", false, fmt.Errorf("%w: project scope requires project_id", ErrValidation)
		}
		// VaultID on project is layout placement only; knowledge_notes project rows use project_id only.
		root := layout.ProjectRoot(meta.DataDir, meta.VaultID, meta.ProjectID)
		return root, meta.ProjectID, "", false, nil
	default:
		return "", "", "", false, fmt.Errorf("%w: invalid instruction scope", ErrValidation)
	}
}

func (s *InstructionStore) findInstructionNoteID(ctx context.Context, projectID, vaultID string, isGlobal bool, fileName string) (string, error) {
	var id string
	var err error
	switch {
	case isGlobal:
		err = s.DB.QueryRowContext(ctx, `
			SELECT id FROM knowledge_notes WHERE is_global=1 AND relative_path=?`, fileName).Scan(&id)
	case projectID != "":
		err = s.DB.QueryRowContext(ctx, `
			SELECT id FROM knowledge_notes WHERE project_id=? AND relative_path=?`, projectID, fileName).Scan(&id)
	case vaultID != "":
		err = s.DB.QueryRowContext(ctx, `
			SELECT id FROM knowledge_notes WHERE vault_id=? AND is_global=0 AND relative_path=?`, vaultID, fileName).Scan(&id)
	default:
		return "", fmt.Errorf("%w: missing scope for lookup", ErrValidation)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func scanKnowledgeNote(row interface{ Scan(...any) error }) (domain.KnowledgeNote, error) {
	var n domain.KnowledgeNote
	var isGlobal int
	var createdAt, updatedAt string
	err := row.Scan(
		&n.ID, &n.RelativePath, &n.Title, &n.Kind, &n.ProjectID, &n.VaultID,
		&isGlobal, &n.SourceNoteID, &n.ContentSHA256, &n.ByteSize,
		&n.FrontmatterJSON, &n.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return domain.KnowledgeNote{}, err
	}
	n.IsGlobal = isGlobal == 1
	if n.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.KnowledgeNote{}, err
	}
	if n.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.KnowledgeNote{}, err
	}
	return n, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func instructionTitle(fileName, content string) string {
	// Prefer first markdown heading; fall back to basename without extension.
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return strings.TrimSuffix(path.Base(fileName), path.Ext(fileName))
}
