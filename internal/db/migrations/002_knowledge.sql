CREATE TABLE knowledge_notes (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL CHECK(kind IN ('source','memory_detail','memory_index','agents','soul','system')),
  project_id TEXT NULL REFERENCES projects(id),
  vault_id TEXT NULL REFERENCES vaults(id),
  is_global INTEGER NOT NULL DEFAULT 0 CHECK(is_global IN (0,1)),
  relative_path TEXT NOT NULL,
  title TEXT,
  content_sha256 TEXT,
  byte_size INTEGER,
  frontmatter_json TEXT,
  status TEXT NOT NULL,
  source_note_id TEXT NULL REFERENCES notes(id),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK(
    (is_global=1 AND project_id IS NULL AND vault_id IS NULL)
    OR (is_global=0 AND project_id IS NOT NULL AND vault_id IS NULL)
    OR (is_global=0 AND project_id IS NULL AND vault_id IS NOT NULL)
  )
);

CREATE UNIQUE INDEX knowledge_notes_project_path ON knowledge_notes(project_id, relative_path)
  WHERE project_id IS NOT NULL;
CREATE UNIQUE INDEX knowledge_notes_vault_path ON knowledge_notes(vault_id, relative_path)
  WHERE vault_id IS NOT NULL AND is_global=0;
CREATE UNIQUE INDEX knowledge_notes_global_path ON knowledge_notes(relative_path)
  WHERE is_global=1;

CREATE TABLE compound_proposals (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES sessions(id),
  scope TEXT NOT NULL CHECK(scope IN ('project','vault','global')),
  project_id TEXT REFERENCES projects(id),
  vault_id TEXT REFERENCES vaults(id),
  status TEXT NOT NULL CHECK(status IN ('pending','approved','rejected','failed')),
  request_key TEXT NOT NULL,
  items_json TEXT NOT NULL CHECK(json_valid(items_json)),
  error TEXT,
  created_at TEXT NOT NULL,
  decided_at TEXT,
  finished_at TEXT,
  UNIQUE(session_id, request_key)
);

CREATE TABLE note_links (
  id TEXT PRIMARY KEY,
  from_note_id TEXT NOT NULL REFERENCES knowledge_notes(id) ON DELETE CASCADE,
  raw_target TEXT NOT NULL,
  to_path TEXT NOT NULL,
  to_note_id TEXT REFERENCES knowledge_notes(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX note_links_to_note ON note_links(to_note_id);
CREATE INDEX note_links_to_path ON note_links(to_path);
CREATE INDEX note_links_from ON note_links(from_note_id);

CREATE VIRTUAL TABLE knowledge_fts USING fts5(
  note_id UNINDEXED,
  title,
  path,
  body,
  tokenize = 'unicode61'
);
