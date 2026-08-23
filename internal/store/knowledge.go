package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

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

// KnowledgeStore upserts knowledge_notes, reindexes note_links, and maintains knowledge_fts.
// It does not write files.
type KnowledgeStore struct {
	DB    *sql.DB
	Clock clock.Clock
}

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
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
	if err := reindexFTS(ctx, tx, note, title, body); err != nil {
		return domain.KnowledgeNote{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.KnowledgeNote{}, err
	}
	return note, nil
}

// ReindexFTS replaces the FTS row for note. note_id is knowledge_notes.id.
func (s *KnowledgeStore) ReindexFTS(ctx context.Context, note domain.KnowledgeNote, title, body string) error {
	return reindexFTS(ctx, s.DB, note, title, body)
}

// RemoveFTS deletes the FTS row for noteID (knowledge_notes.id).
func (s *KnowledgeStore) RemoveFTS(ctx context.Context, noteID string) error {
	return removeFTS(ctx, s.DB, noteID)
}

func reindexFTS(ctx context.Context, exec sqlExecer, note domain.KnowledgeNote, title, body string) error {
	if err := removeFTS(ctx, exec, note.ID); err != nil {
		return err
	}
	_, err := exec.ExecContext(ctx, `
		INSERT INTO knowledge_fts(note_id, title, path, body) VALUES(?,?,?,?)`,
		note.ID, title, note.RelativePath, body,
	)
	return err
}

func removeFTS(ctx context.Context, exec sqlExecer, noteID string) error {
	if noteID == "" {
		return fmt.Errorf("%w: empty note id", ErrValidation)
	}
	_, err := exec.ExecContext(ctx, `DELETE FROM knowledge_fts WHERE note_id=?`, noteID)
	return err
}

// SearchHit is one project-scoped FTS match. KnowledgeID is knowledge_notes.id
// (API json: knowledge_id) — never v1 notes.id.
type SearchHit struct {
	KnowledgeID  string
	Path         string
	Title        string
	Snippet      string
	Kind         domain.KnowledgeKind
	SourceNoteID string
	Rank         float64
}

const (
	searchLimitDefault = 20
	searchLimitMax     = 50
	snippetRadius      = 60
)

// SearchProject runs FTS over the current project's knowledge corpus only.
// Empty/whitespace query returns no hits. limit<=0 defaults to 20; max 50.
func (s *KnowledgeStore) SearchProject(ctx context.Context, projectID, query string, limit int) ([]SearchHit, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id required", ErrValidation)
	}
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = searchLimitDefault
	}
	if limit > searchLimitMax {
		limit = searchLimitMax
	}
	match, needle, ok := sanitizeFTSQuery(query)
	if !ok {
		return nil, nil
	}
	needleLower := strings.ToLower(needle)

	rows, err := s.DB.QueryContext(ctx, `
		SELECT kn.id, kn.relative_path, coalesce(kn.title,''), kn.kind,
		       coalesce(kn.source_note_id,''), coalesce(knowledge_fts.body,''),
		       bm25(knowledge_fts)
		FROM knowledge_fts
		INNER JOIN knowledge_notes kn ON kn.id = knowledge_fts.note_id
		WHERE kn.project_id = ?
		  AND kn.kind IN ('source','memory_detail','memory_index','agents','soul','system')
		  AND knowledge_fts MATCH ?
		ORDER BY
		  CASE WHEN instr(lower(coalesce(kn.title,'')), ?) > 0 THEN 0
		       WHEN instr(lower(kn.relative_path), ?) > 0 THEN 1
		       ELSE 2 END,
		  bm25(knowledge_fts),
		  kn.updated_at DESC
		LIMIT ?`,
		projectID, match, needleLower, needleLower, limit,
	)
	if err != nil {
		if isFTSQueryError(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var out []SearchHit
	for rows.Next() {
		var hit SearchHit
		var body string
		var bm25 float64
		if err := rows.Scan(&hit.KnowledgeID, &hit.Path, &hit.Title, &hit.Kind, &hit.SourceNoteID, &body, &bm25); err != nil {
			return nil, err
		}
		hit.Snippet = snippetAround(hit.Title, body, needle)
		hit.Rank = -bm25
		out = append(out, hit)
	}
	if err := rows.Err(); err != nil {
		if isFTSQueryError(err) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

func sanitizeFTSQuery(query string) (match, needle string, ok bool) {
	var b strings.Builder
	for _, r := range query {
		switch {
		case r == '"' || r == '*' || r == '(' || r == ')' || r == ':' ||
			r == '{' || r == '}' || r == '^' || r == '[' || r == ']':
			b.WriteByte(' ')
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	var quoted []string
	for _, part := range strings.Fields(b.String()) {
		switch strings.ToUpper(part) {
		case "AND", "OR", "NOT", "NEAR":
			continue
		}
		quoted = append(quoted, `"`+part+`"`)
		if needle == "" {
			needle = part
		}
	}
	if len(quoted) == 0 {
		return "", "", false
	}
	return strings.Join(quoted, " "), needle, true
}

func isFTSQueryError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "fts5") || strings.Contains(msg, "fts3") || strings.Contains(msg, "match")
}

func snippetAround(title, body, needle string) string {
	if s, ok := windowAround(body, needle, snippetRadius); ok {
		return s
	}
	if s, ok := windowAround(title, needle, snippetRadius); ok {
		return s
	}
	return clipRunes(body, 2*snippetRadius)
}

func windowAround(text, needle string, radius int) (string, bool) {
	if text == "" || needle == "" {
		return "", false
	}
	idx := strings.Index(strings.ToLower(text), strings.ToLower(needle))
	if idx < 0 {
		return "", false
	}
	runes := []rune(text)
	start := len([]rune(text[:idx])) - radius
	if start < 0 {
		start = 0
	}
	end := len([]rune(text[:idx])) + len([]rune(needle)) + radius
	if end > len(runes) {
		end = len(runes)
	}
	out := string(runes[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(runes) {
		out = out + "…"
	}
	return strings.TrimSpace(out), true
}

func clipRunes(text string, max int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "…"
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

// Backlink is an inbound wikilink to a knowledge note. Snippet is empty in v1.
type Backlink struct {
	FromNoteID string
	FromPath   string
	FromTitle  string
	Snippet    string
}

// Backlinks returns notes that link to noteID (knowledge_notes.id only).
// Matches resolved edges (to_note_id) and same-scope unresolved path edges.
func (s *KnowledgeStore) Backlinks(ctx context.Context, noteID string) ([]Backlink, error) {
	note, err := s.ByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	scopeSQL, scopeArgs, err := knowledgeScopeWhere(note.ProjectID, note.VaultID, note.IsGlobal)
	if err != nil {
		return nil, err
	}
	args := append([]any{note.ID, note.RelativePath}, scopeArgs...)
	rows, err := s.DB.QueryContext(ctx, `
		SELECT kn.id, kn.relative_path, coalesce(kn.title,'')
		FROM note_links nl
		INNER JOIN knowledge_notes kn ON kn.id = nl.from_note_id
		WHERE nl.to_note_id = ?
		   OR (nl.to_path = ? AND nl.from_note_id IN (SELECT id FROM knowledge_notes WHERE `+scopeSQL+`))
		ORDER BY kn.relative_path, kn.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Backlink
	seen := map[string]bool{}
	for rows.Next() {
		var bl Backlink
		if err := rows.Scan(&bl.FromNoteID, &bl.FromPath, &bl.FromTitle); err != nil {
			return nil, err
		}
		if seen[bl.FromNoteID] {
			continue
		}
		seen[bl.FromNoteID] = true
		out = append(out, bl)
	}
	return out, rows.Err()
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
