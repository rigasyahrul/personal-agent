package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

const (
	// MaxCompoundItems is the maximum number of items in one proposal.
	MaxCompoundItems = 20
	// MaxCompoundItemBytes is the per-item content size cap (256 KiB).
	MaxCompoundItemBytes = 256 << 10
)

// Compound item kinds (proposal JSON).
const (
	CompoundKindAgentsPatch     = "agents_patch"
	CompoundKindMemoryDetail    = "memory_detail"
	CompoundKindLessonsIndexRow = "lessons_index_row"
)

// Compound item actions.
const (
	CompoundActionCreate = "create"
	CompoundActionUpdate = "update"
)

// memoryDetailPathRE matches Canonical memory_detail paths:
// memory/YYYYMMDD-HHMM-slug.md
var memoryDetailPathRE = regexp.MustCompile(`^memory/[0-9]{8}-[0-9]{4}-[a-z0-9-]+\.md$`)

// CompoundItem is one proposed file change inside a compound proposal.
type CompoundItem struct {
	Kind          string `json:"kind"` // agents_patch|memory_detail|lessons_index_row
	Path          string `json:"path"`
	Action        string `json:"action"` // create|update
	Title         string `json:"title,omitempty"`
	Content       string `json:"content"`
	ContentSHA256 string `json:"content_sha256"`
}

// CreateProposalInput creates a pending compound proposal.
// Scope, ProjectID, and VaultID MUST be filled by the handler from the session row.
type CreateProposalInput struct {
	SessionID  string
	RequestKey string
	Scope      domain.CompoundScope
	ProjectID  string
	VaultID    string
	Items      []CompoundItem
	Now        time.Time
}

// CompoundStore persists compound proposals.
type CompoundStore struct {
	DB      *sql.DB
	Clock   clock.Clock
	Barrier MutBarrier
}

// ValidateCompoundItems enforces Canonical compound item rules.
// Call on CreatePending, Decide(approve) with final items, and PublishApproved.
func ValidateCompoundItems(scope domain.CompoundScope, items []CompoundItem) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: compound items must be non-empty", ErrValidation)
	}
	if len(items) > MaxCompoundItems {
		return fmt.Errorf("%w: at most %d compound items", ErrValidation, MaxCompoundItems)
	}

	switch scope {
	case domain.CompoundScopeProject, domain.CompoundScopeVault, domain.CompoundScopeGlobal:
	default:
		return fmt.Errorf("%w: invalid compound scope", ErrValidation)
	}

	var hasMemoryDetail, hasLessonsIndex bool
	for i, it := range items {
		if err := validateOneCompoundItem(scope, it); err != nil {
			return fmt.Errorf("%w: item[%d]: %v", ErrValidation, i, err)
		}
		switch it.Kind {
		case CompoundKindMemoryDetail:
			hasMemoryDetail = true
		case CompoundKindLessonsIndexRow:
			if it.Path == "memory/lessons.md" {
				hasLessonsIndex = true
			}
		}
	}
	if hasMemoryDetail && !hasLessonsIndex {
		return fmt.Errorf("%w: memory_detail requires lessons_index_row with path memory/lessons.md", ErrValidation)
	}
	return nil
}

func validateOneCompoundItem(scope domain.CompoundScope, it CompoundItem) error {
	switch it.Kind {
	case CompoundKindAgentsPatch, CompoundKindMemoryDetail, CompoundKindLessonsIndexRow:
	default:
		return fmt.Errorf("invalid kind %q", it.Kind)
	}
	switch it.Action {
	case CompoundActionCreate, CompoundActionUpdate:
	default:
		return fmt.Errorf("invalid action %q", it.Action)
	}

	if len(it.Content) > MaxCompoundItemBytes {
		return fmt.Errorf("content exceeds %d bytes", MaxCompoundItemBytes)
	}
	sum := sha256.Sum256([]byte(it.Content))
	want := hex.EncodeToString(sum[:])
	if !strings.EqualFold(strings.TrimSpace(it.ContentSHA256), want) {
		return fmt.Errorf("content_sha256 mismatch")
	}

	path := it.Path
	if err := paths.ValidateKnowledgeRelPath(path); err != nil {
		return err
	}

	// Compound allowlist: AGENTS.md or memory/** only — never source/**, .agents/**, SOUL/SYSTEM.
	switch {
	case path == "AGENTS.md":
		if it.Kind != CompoundKindAgentsPatch {
			return fmt.Errorf("AGENTS.md requires kind agents_patch")
		}
		if scope == domain.CompoundScopeVault {
			return fmt.Errorf("vault scope forbids agents_patch")
		}
	case path == "SOUL.md" || path == "SYSTEM.md":
		return fmt.Errorf("path %q is forbidden for compound", path)
	case strings.HasPrefix(path, "source/"):
		return fmt.Errorf("source/** is forbidden for compound")
	case strings.HasPrefix(path, ".agents/"):
		return fmt.Errorf(".agents/** is forbidden for compound")
	case strings.HasPrefix(path, "memory/"):
		switch it.Kind {
		case CompoundKindMemoryDetail:
			if !memoryDetailPathRE.MatchString(path) {
				return fmt.Errorf("memory_detail path must match memory/YYYYMMDD-HHMM-slug.md")
			}
		case CompoundKindLessonsIndexRow:
			if path != "memory/lessons.md" {
				return fmt.Errorf("lessons_index_row path must be memory/lessons.md")
			}
		case CompoundKindAgentsPatch:
			return fmt.Errorf("agents_patch path must be AGENTS.md")
		}
	default:
		return fmt.Errorf("path not allowlisted for compound")
	}

	// Kind/path consistency for agents_patch.
	if it.Kind == CompoundKindAgentsPatch && path != "AGENTS.md" {
		return fmt.Errorf("agents_patch path must be AGENTS.md")
	}
	return nil
}

// CreatePending inserts a pending compound proposal.
// Idempotent: same session_id+request_key with equivalent items_json returns the existing row.
// Different items under the same key → ErrConflict.
func (s *CompoundStore) CreatePending(ctx context.Context, in CreateProposalInput) (domain.CompoundProposal, error) {
	var out domain.CompoundProposal
	err := s.withBarrier(func() error {
		var e error
		out, e = s.createPending(ctx, in)
		return e
	})
	return out, err
}

func (s *CompoundStore) withBarrier(fn func() error) error {
	if s.Barrier == nil {
		return fn()
	}
	return s.Barrier.Mutate(fn)
}

func (s *CompoundStore) createPending(ctx context.Context, in CreateProposalInput) (domain.CompoundProposal, error) {
	if strings.TrimSpace(in.SessionID) == "" || strings.TrimSpace(in.RequestKey) == "" {
		return domain.CompoundProposal{}, fmt.Errorf("%w: session_id and request_key required", ErrValidation)
	}
	if err := ValidateCompoundItems(in.Scope, in.Items); err != nil {
		return domain.CompoundProposal{}, err
	}

	itemsJSON, err := marshalCompoundItems(in.Items)
	if err != nil {
		return domain.CompoundProposal{}, err
	}

	now := in.Now
	if now.IsZero() && s.Clock != nil {
		now = s.Clock.Now()
	}
	now = now.UTC()

	id := ids.NewID()
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO compound_proposals(
			id, session_id, scope, project_id, vault_id, status, request_key, items_json, created_at
		) VALUES(?,?,?,?,?,?,?,?,?)`,
		id,
		in.SessionID,
		string(in.Scope),
		nullable(in.ProjectID),
		nullable(in.VaultID),
		string(domain.CompoundStatusPending),
		in.RequestKey,
		itemsJSON,
		formatTime(now),
	)
	if err == nil {
		return domain.CompoundProposal{
			ID:         id,
			SessionID:  in.SessionID,
			Scope:      in.Scope,
			ProjectID:  in.ProjectID,
			VaultID:    in.VaultID,
			Status:     domain.CompoundStatusPending,
			RequestKey: in.RequestKey,
			ItemsJSON:  itemsJSON,
			CreatedAt:  now,
		}, nil
	}

	// Unique(session_id, request_key) conflict — compare items fingerprint.
	existing, lookupErr := s.getBySessionRequestKey(ctx, in.SessionID, in.RequestKey)
	if lookupErr != nil {
		if errors.Is(lookupErr, ErrNotFound) {
			// Not a unique conflict we understand; surface original insert error.
			return domain.CompoundProposal{}, err
		}
		return domain.CompoundProposal{}, lookupErr
	}
	if compoundItemsEquivalent(existing.ItemsJSON, itemsJSON) {
		return existing, nil
	}
	return domain.CompoundProposal{}, ErrConflict
}

func (s *CompoundStore) getBySessionRequestKey(ctx context.Context, sessionID, requestKey string) (domain.CompoundProposal, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, session_id, scope, coalesce(project_id,''), coalesce(vault_id,''), status,
		       request_key, items_json, coalesce(error,''), created_at, decided_at, finished_at
		FROM compound_proposals
		WHERE session_id=? AND request_key=?`, sessionID, requestKey)
	return scanCompoundProposal(row)
}

func scanCompoundProposal(row interface{ Scan(dest ...any) error }) (domain.CompoundProposal, error) {
	var p domain.CompoundProposal
	var scope, status, createdAt string
	var decidedAt, finishedAt sql.NullString
	if err := row.Scan(
		&p.ID, &p.SessionID, &scope, &p.ProjectID, &p.VaultID, &status,
		&p.RequestKey, &p.ItemsJSON, &p.Error, &createdAt, &decidedAt, &finishedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, ErrNotFound
		}
		return p, err
	}
	p.Scope = domain.CompoundScope(scope)
	p.Status = domain.CompoundStatus(status)
	var err error
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return p, err
	}
	if decidedAt.Valid {
		t, err := parseTime(decidedAt.String)
		if err != nil {
			return p, err
		}
		p.DecidedAt = &t
	}
	if finishedAt.Valid {
		t, err := parseTime(finishedAt.String)
		if err != nil {
			return p, err
		}
		p.FinishedAt = &t
	}
	return p, nil
}

func marshalCompoundItems(items []CompoundItem) (string, error) {
	// Normalize empty slice to [] for stable JSON (validation already rejects empty).
	if items == nil {
		items = []CompoundItem{}
	}
	b, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// compoundItemsEquivalent compares canonical JSON encodings of item slices.
func compoundItemsEquivalent(a, b string) bool {
	if a == b {
		return true
	}
	var ia, ib []CompoundItem
	if err := json.Unmarshal([]byte(a), &ia); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &ib); err != nil {
		return false
	}
	// Re-marshal with the same encoder for structural equality.
	ba, err := json.Marshal(ia)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(ib)
	if err != nil {
		return false
	}
	return string(ba) == string(bb)
}
