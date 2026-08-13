package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

type TreeEntry struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	NoteID string `json:"note_id,omitempty"`
}
type NoteDocument struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	RelativePath  string `json:"relative_path"`
	ContentSHA256 string `json:"content_sha256"`
	ByteSize      int64  `json:"byte_size"`
	Revision      int    `json:"revision"`
	Body          []byte `json:"-"`
}
type NoteStore struct {
	db      *sql.DB
	dataDir string
}

func NewNoteStore(db *sql.DB, dataDir string) *NoteStore { return &NoteStore{db: db, dataDir: dataDir} }

func (s *NoteStore) sourceRoot(ctx context.Context, projectID string) (*fsroot.Root, error) {
	var vault sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT vault_id FROM projects WHERE id=?`, projectID).Scan(&vault)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	r, err := fsroot.Open(layout.SourceDir(layout.ProjectRoot(s.dataDir, vault.String, projectID)))
	if err != nil {
		return nil, ErrIntegrity
	}
	return r, nil
}

func (s *NoteStore) Tree(ctx context.Context, projectID string) ([]TreeEntry, error) {
	r, err := s.sourceRoot(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	rows, err := s.db.QueryContext(ctx, `SELECT relative_path,id,content_sha256,byte_size FROM notes WHERE project_id=? AND status='ready'`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type indexedNote struct {
		id, hash string
		size     int64
	}
	ids := map[string]indexedNote{}
	for rows.Next() {
		var p, id string
		var hash sql.NullString
		var size sql.NullInt64
		if err := rows.Scan(&p, &id, &hash, &size); err != nil {
			return nil, err
		}
		if _, err = paths.ValidateRelPath(p); err != nil || path.Ext(p) != ".md" || !hash.Valid || !size.Valid {
			return nil, ErrIntegrity
		}
		ids[p] = indexedNote{id: id, hash: hash.String, size: size.Int64}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	out := []TreeEntry{}
	seen := map[string]bool{}
	err = r.Walk(func(p string, info fs.FileInfo) error {
		if info.IsDir() {
			out = append(out, TreeEntry{Path: p, Kind: "folder"})
			return nil
		}
		if !info.Mode().IsRegular() {
			return ErrIntegrity
		}
		indexed, ok := ids[p]
		if !ok {
			return ErrIntegrity
		}
		body, err := r.ReadFile(p)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(body)) != indexed.hash || int64(len(body)) != indexed.size {
			return ErrIntegrity
		}
		seen[p] = true
		out = append(out, TreeEntry{Path: p, Kind: "note", NoteID: indexed.id})
		return nil
	})
	if err != nil {
		return nil, ErrIntegrity
	}
	for p := range ids {
		if !seen[p] {
			return nil, ErrIntegrity
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (s *NoteStore) Get(ctx context.Context, id string) (NoteDocument, error) {
	var n NoteDocument
	var vault, hash sql.NullString
	var size sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT n.id,n.project_id,n.relative_path,n.content_sha256,n.byte_size,n.revision,p.vault_id FROM notes n JOIN projects p ON p.id=n.project_id WHERE n.id=? AND n.status='ready'`, id).Scan(&n.ID, &n.ProjectID, &n.RelativePath, &hash, &size, &n.Revision, &vault)
	if errors.Is(err, sql.ErrNoRows) {
		return n, ErrNotFound
	}
	if err != nil {
		return n, err
	}
	if _, err = paths.ValidateRelPath(n.RelativePath); err != nil || path.Ext(n.RelativePath) != ".md" || !hash.Valid || !size.Valid {
		return n, ErrIntegrity
	}
	n.ContentSHA256, n.ByteSize = hash.String, size.Int64
	r, err := fsroot.Open(layout.SourceDir(layout.ProjectRoot(s.dataDir, vault.String, n.ProjectID)))
	if err != nil {
		return n, ErrIntegrity
	}
	defer r.Close()
	b, err := r.ReadFile(n.RelativePath)
	if err != nil {
		return n, ErrIntegrity
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != n.ContentSHA256 || int64(len(b)) != n.ByteSize {
		return n, ErrIntegrity
	}
	n.Body = b
	return n, nil
}
