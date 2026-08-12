## Phase 2: Projects + source tree

### Task 9: Derive and create project layout

**Files:**
- Create: `internal/layout/layout.go`
- Test: `internal/layout/layout_test.go`

**Interfaces:**
- Consumes: trusted `dataDir`, database IDs, and `internal/fsroot` containment guarantees from Phase 1.
- Produces: `type SessionHome string`, `ProjectRoot(dataDir, vaultID, projectID string) string`, `SourceDir(projectRoot string) string`, `SessionWorkspace(dataDir string, home SessionHome, vaultID, projectID, sessionID string) string`, and `EnsureProjectDirs(dataDir, vaultID, projectID string) error`.

- [ ] **Step 1: Write the failing test**

```go
package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectLayoutAndCreation(t *testing.T) {
	d := t.TempDir()
	if got, want := ProjectRoot(d, "", "p1"), filepath.Join(d, "files", "global", "projects", "p1"); got != want { t.Fatalf("root=%q want %q", got, want) }
	if got, want := ProjectRoot(d, "v1", "p1"), filepath.Join(d, "files", "vaults", "v1", "projects", "p1"); got != want { t.Fatalf("vault root=%q want %q", got, want) }
	if got := SourceDir(ProjectRoot(d, "", "p1")); got != filepath.Join(d, "files", "global", "projects", "p1", "source") { t.Fatal(got) }
	if got := SessionWorkspace(d, SessionHome("project"), "v1", "p1", "s1"); got != filepath.Join(d, "files", "vaults", "v1", "projects", "p1", "sessions", "s1") { t.Fatal(got) }
	if err := EnsureProjectDirs(d, "v1", "p1"); err != nil { t.Fatal(err) }
	for _, name := range []string{"source", "memory", "soul"} {
		if st, err := os.Stat(filepath.Join(ProjectRoot(d, "v1", "p1"), name)); err != nil || !st.IsDir() { t.Fatalf("%s: %v", name, err) }
	}
}

func TestSessionWorkspaceAllHomes(t *testing.T) {
	d := t.TempDir()
	cases := map[SessionHome]string{
		"global": filepath.Join(d, "files", "global", "sessions", "s"),
		"vault": filepath.Join(d, "files", "vaults", "v", "sessions", "s"),
		"project": filepath.Join(d, "files", "global", "projects", "p", "sessions", "s"),
	}
	for home, want := range cases {
		if got := SessionWorkspace(d, home, "v", "p", "s"); got != want { t.Errorf("%s: %q want %q", home, got, want) }
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/layout -run 'TestProjectLayout|TestSessionWorkspace' -v`
Expected: FAIL because package `internal/layout` and its functions do not exist.

- [ ] **Step 3: Write minimal implementation**

```go
package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

type SessionHome string

func ProjectRoot(dataDir, vaultID, projectID string) string {
	if vaultID == "" { return filepath.Join(dataDir, "files", "global", "projects", projectID) }
	return filepath.Join(dataDir, "files", "vaults", vaultID, "projects", projectID)
}

func SourceDir(projectRoot string) string { return filepath.Join(projectRoot, "source") }

func SessionWorkspace(dataDir string, home SessionHome, vaultID, projectID, sessionID string) string {
	switch home {
	case "global": return filepath.Join(dataDir, "files", "global", "sessions", sessionID)
	case "vault": return filepath.Join(dataDir, "files", "vaults", vaultID, "sessions", sessionID)
	case "project": return filepath.Join(ProjectRoot(dataDir, vaultID, projectID), "sessions", sessionID)
	default: panic(fmt.Sprintf("invalid session home %q", home))
	}
}

func EnsureProjectDirs(dataDir, vaultID, projectID string) error {
	root := ProjectRoot(dataDir, vaultID, projectID)
	for _, name := range []string{"source", "memory", "soul"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil { return fmt.Errorf("create project %s: %w", name, err) }
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/layout -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/layout/layout.go internal/layout/layout_test.go
git commit -m "feat: add rooted project layout"
```

### Task 10: Persist vaults and projects with immutable placement

**Files:**
- Create: `internal/store/vaults.go`
- Create: `internal/store/projects.go`
- Test: `internal/store/projects_test.go`

**Interfaces:**
- Consumes: `*sql.DB`, `ids.NewID()`, `clock.Clock`, the Phase 1 `vaults`/`projects` schema, and `layout.EnsureProjectDirs`.
- Produces: `NewVaultStore(db, clock) *VaultStore`, `Create(ctx, name) (domain.Vault, error)`, `List(ctx) ([]domain.Vault, error)`, `NewProjectStore(db, dataDir, clock) *ProjectStore`, `Create(ctx, name, vaultID) (domain.Project, error)`, `List(ctx) ([]domain.Project, error)`, and `Get(ctx, id) (domain.Project, error)`.

- [ ] **Step 1: Write the failing test**

```go
package store_test

import (
	"context"
	"testing"
	"time"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	dbtest "github.com/rigasyahrul/personal-agent/internal/db"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

func TestVaultAndProjectCRUD(t *testing.T) {
	d := t.TempDir(); database, err := dbtest.Open(d); if err != nil { t.Fatal(err) }; defer database.Close()
	c := &clock.FakeClock{T: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)}
	vs := store.NewVaultStore(database, c); v, err := vs.Create(context.Background(), "Learning"); if err != nil { t.Fatal(err) }
	ps := store.NewProjectStore(database, d, c); p, err := ps.Create(context.Background(), "Go", v.ID); if err != nil { t.Fatal(err) }
	if p.VaultID != v.ID || p.Name != "Go" { t.Fatalf("%+v", p) }
	if _, err := ps.Create(context.Background(), "Bad", "missing"); err == nil { t.Fatal("expected unknown vault failure") }
	got, err := ps.Get(context.Background(), p.ID); if err != nil || got.VaultID != v.ID { t.Fatalf("%+v %v", got, err) }
	list, err := ps.List(context.Background()); if err != nil || len(list) != 1 { t.Fatalf("%+v %v", list, err) }
	if err := layout.EnsureProjectDirs(d, "", p.ID); err != nil { t.Fatal(err) }
	if _, err := database.Exec(`UPDATE projects SET vault_id=NULL WHERE id=?`, p.ID); err == nil { t.Fatal("vault placement must be immutable") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run TestVaultAndProjectCRUD -v`
Expected: FAIL because the stores and constructors are undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/store/vaults.go
package store

import (
	"context"
	"database/sql"
	"strings"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
)

type VaultStore struct { db *sql.DB; clock clock.Clock }
func NewVaultStore(db *sql.DB, c clock.Clock) *VaultStore { return &VaultStore{db: db, clock: c} }
func (s *VaultStore) Create(ctx context.Context, name string) (domain.Vault, error) {
	name = strings.TrimSpace(name); v := domain.Vault{ID: ids.NewID(), Name: name, CreatedAt: s.clock.Now().UTC(), UpdatedAt: s.clock.Now().UTC()}
	if name == "" { return domain.Vault{}, ErrInvalid }
	_, err := s.db.ExecContext(ctx, `INSERT INTO vaults(id,name,created_at,updated_at) VALUES(?,?,?,?)`, v.ID,v.Name,v.CreatedAt,v.UpdatedAt); return v, err
}
func (s *VaultStore) List(ctx context.Context) ([]domain.Vault, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,created_at,updated_at FROM vaults ORDER BY name,id`); if err != nil { return nil, err }; defer rows.Close()
	out := []domain.Vault{}; for rows.Next() { var v domain.Vault; if err := rows.Scan(&v.ID,&v.Name,&v.CreatedAt,&v.UpdatedAt); err != nil { return nil,err }; out=append(out,v) }; return out,rows.Err()
}
```

```go
// internal/store/projects.go
package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

var ErrInvalid = errors.New("invalid input")
type ProjectStore struct { db *sql.DB; dataDir string; clock clock.Clock }
func NewProjectStore(db *sql.DB, dataDir string, c clock.Clock) *ProjectStore { return &ProjectStore{db:db,dataDir:dataDir,clock:c} }
func (s *ProjectStore) Create(ctx context.Context, name, vaultID string) (domain.Project, error) {
	name=strings.TrimSpace(name); if name=="" { return domain.Project{},ErrInvalid }
	if vaultID!="" { var n int; if err:=s.db.QueryRowContext(ctx,`SELECT count(*) FROM vaults WHERE id=?`,vaultID).Scan(&n); err!=nil||n!=1 { return domain.Project{},ErrInvalid } }
	now:=s.clock.Now().UTC(); p:=domain.Project{ID:ids.NewID(),VaultID:vaultID,Name:name,CreatedAt:now,UpdatedAt:now}
	tx,err:=s.db.BeginTx(ctx,nil); if err!=nil{return domain.Project{},err}; defer tx.Rollback()
	var nullable any; if vaultID!="" { nullable=vaultID }
	if _,err=tx.ExecContext(ctx,`INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES(?,?,?,?,?)`,p.ID,nullable,p.Name,p.CreatedAt,p.UpdatedAt);err!=nil{return domain.Project{},err}
	if err=layout.EnsureProjectDirs(s.dataDir,vaultID,p.ID);err!=nil{return domain.Project{},err}; if err=tx.Commit();err!=nil{return domain.Project{},err}; return p,nil
}
func scanProject(row interface{ Scan(...any) error }) (domain.Project,error) { var p domain.Project; var v sql.NullString; err:=row.Scan(&p.ID,&v,&p.Name,&p.CreatedAt,&p.UpdatedAt); if v.Valid { p.VaultID=v.String }; return p,err }
func (s *ProjectStore) Get(ctx context.Context,id string)(domain.Project,error){return scanProject(s.db.QueryRowContext(ctx,`SELECT id,vault_id,name,created_at,updated_at FROM projects WHERE id=?`,id))}
func (s *ProjectStore) List(ctx context.Context)([]domain.Project,error){rows,err:=s.db.QueryContext(ctx,`SELECT id,vault_id,name,created_at,updated_at FROM projects ORDER BY updated_at DESC,id`);if err!=nil{return nil,err};defer rows.Close();out:=[]domain.Project{};for rows.Next(){p,e:=scanProject(rows);if e!=nil{return nil,e};out=append(out,p)};return out,rows.Err()}
```

Add `Vault` and `Project` structs with the fields used above to `internal/domain/models.go`, and add an immutable-placement trigger to `001_init.sql`:

```sql
CREATE TRIGGER projects_vault_immutable BEFORE UPDATE OF vault_id ON projects
WHEN NEW.vault_id IS NOT OLD.vault_id BEGIN SELECT RAISE(ABORT, 'project vault_id is immutable'); END;
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store -run TestVaultAndProjectCRUD -v`
Expected: PASS with one persisted project and rejected placement update.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/models.go internal/db/migrations/001_init.sql internal/store/vaults.go internal/store/projects.go internal/store/projects_test.go
git commit -m "feat: persist vaults and projects"
```

### Task 11: Serve vault, project, overview, and home data

**Files:**
- Create: `internal/httpapi/project_handlers.go`
- Modify: `internal/httpapi/server.go`
- Test: `internal/httpapi/project_handlers_test.go`

**Interfaces:**
- Consumes: authenticated/CSRF-protected Phase 1 mux, `VaultStore`, `ProjectStore`, and SQLite aggregate tables.
- Produces: `GET/POST /api/v1/vaults`, `GET/POST /api/v1/projects`, `GET /api/v1/projects/{id}`, and `GET /api/v1/home`; project DTO fields are `id`, `vault_id`, `vault_name`, `name`, `note_count`, `session_count`, and `due_count`.

- [ ] **Step 1: Write the failing test**

```go
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"github.com/rigasyahrul/personal-agent/internal/httpapi"
)

func TestProjectAPI(t *testing.T) {
	s := newAuthenticatedTestServer(t)
	w := httptest.NewRecorder(); r := httptest.NewRequest(http.MethodPost,"/api/v1/projects",strings.NewReader(`{"name":"Go","vault_id":null}`)); r.Header.Set("Content-Type","application/json"); addAuthAndCSRF(r)
	s.ServeHTTP(w,r); if w.Code!=http.StatusCreated { t.Fatalf("%d %s",w.Code,w.Body.String()) }
	w=httptest.NewRecorder(); r=httptest.NewRequest(http.MethodGet,"/api/v1/home",nil); addAuth(r); s.ServeHTTP(w,r)
	if w.Code!=http.StatusOK || !strings.Contains(w.Body.String(),`"note_count":0`) || !strings.Contains(w.Body.String(),`"session_count":0`) || !strings.Contains(w.Body.String(),`"due_count":0`) { t.Fatalf("%d %s",w.Code,w.Body.String()) }
	_ = httpapi.ProjectDTO{}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestProjectAPI -v`
Expected: FAIL because the routes and `ProjectDTO` are absent.

- [ ] **Step 3: Write minimal implementation**

```go
package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type ProjectDTO struct { ID string `json:"id"`; VaultID string `json:"vault_id,omitempty"`; VaultName string `json:"vault_name,omitempty"`; Name string `json:"name"`; NoteCount int `json:"note_count"`; SessionCount int `json:"session_count"`; DueCount int `json:"due_count"` }
type ProjectHandlers struct { Vaults *store.VaultStore; Projects *store.ProjectStore }
func jsonOut(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func (h ProjectHandlers) vaults(w http.ResponseWriter,r *http.Request){if r.Method==http.MethodGet{v,e:=h.Vaults.List(r.Context());if e!=nil{http.Error(w,e.Error(),500);return};jsonOut(w,200,v);return};var in struct{Name string `json:"name"`};if json.NewDecoder(r.Body).Decode(&in)!=nil{http.Error(w,"invalid json",400);return};v,e:=h.Vaults.Create(r.Context(),in.Name);if e!=nil{http.Error(w,e.Error(),400);return};jsonOut(w,201,v)}
func (h ProjectHandlers) projects(w http.ResponseWriter,r *http.Request){if r.Method==http.MethodGet{ps,e:=h.Projects.List(r.Context());if e!=nil{http.Error(w,e.Error(),500);return};out:=make([]ProjectDTO,len(ps));for i,p:=range ps{out[i]=ProjectDTO{ID:p.ID,VaultID:p.VaultID,Name:p.Name}};jsonOut(w,200,out);return};var in struct{Name string `json:"name"`;VaultID *string `json:"vault_id"`};if json.NewDecoder(r.Body).Decode(&in)!=nil{http.Error(w,"invalid json",400);return};v:="";if in.VaultID!=nil{v=*in.VaultID};p,e:=h.Projects.Create(r.Context(),in.Name,v);if e!=nil{http.Error(w,e.Error(),400);return};jsonOut(w,201,ProjectDTO{ID:p.ID,VaultID:p.VaultID,Name:p.Name})}
func (h ProjectHandlers) project(w http.ResponseWriter,r *http.Request){p,e:=h.Projects.Get(r.Context(),r.PathValue("id"));if e!=nil{http.Error(w,"project not found",404);return};jsonOut(w,200,ProjectDTO{ID:p.ID,VaultID:p.VaultID,Name:p.Name})}
func (h ProjectHandlers) home(w http.ResponseWriter,r *http.Request){ps,e:=h.Projects.List(r.Context());if e!=nil{http.Error(w,e.Error(),500);return};out:=make([]ProjectDTO,len(ps));for i,p:=range ps{out[i]=ProjectDTO{ID:p.ID,VaultID:p.VaultID,Name:p.Name}};jsonOut(w,200,map[string]any{"projects":out,"due_count":0,"generated_at":time.Now().UTC()})}
func (h ProjectHandlers) Register(mux *http.ServeMux){mux.HandleFunc("GET /api/v1/vaults",h.vaults);mux.HandleFunc("POST /api/v1/vaults",h.vaults);mux.HandleFunc("GET /api/v1/projects",h.projects);mux.HandleFunc("POST /api/v1/projects",h.projects);mux.HandleFunc("GET /api/v1/projects/{id}",h.project);mux.HandleFunc("GET /api/v1/home",h.home)}
```

Wire `ProjectHandlers.Register(mux)` in `server.go` inside the existing auth/CSRF middleware. Keep aggregate values zero until their tables gain rows; Task 12 supplies real note counts and Phase 3 supplies sessions.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/httpapi -run TestProjectAPI -v`
Expected: PASS and the home DTO includes all three count fields.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/server.go internal/httpapi/project_handlers.go internal/httpapi/project_handlers_test.go
git commit -m "feat: expose project and home APIs"
```

### Task 12: Index and integrity-check the source tree

**Files:**
- Create: `internal/store/notes.go`
- Create: `internal/httpapi/note_handlers.go`
- Modify: `internal/httpapi/project_handlers.go`
- Test: `internal/store/notes_test.go`
- Test: `internal/httpapi/note_handlers_test.go`

**Interfaces:**
- Consumes: `paths.ValidateRelPath`, `layout.SourceDir`, ready Note rows, project placement, and rooted filesystem reads.
- Produces: `NewNoteStore(db, dataDir) *NoteStore`, `Tree(ctx, projectID) ([]TreeEntry, error)`, `Get(ctx, noteID) (NoteDocument, error)`, `POST /api/v1/projects/{id}/folders`, `GET /api/v1/projects/{id}/tree`, and `GET /api/v1/notes/{id}`; URLs accept note IDs only.

- [ ] **Step 1: Write the failing test**

```go
package store_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

func TestNoteTreeAndIntegrity(t *testing.T) {
	d,db,p := projectFixture(t); body:=[]byte("# Safe\n"); path:=filepath.Join(projectSource(t,d,p),"guide","safe.md")
	if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil{t.Fatal(err)};if err:=os.WriteFile(path,body,0600);err!=nil{t.Fatal(err)};sum:=sha256.Sum256(body)
	if _,err:=db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1',?,?,?,?,'ready',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,p.ID,"guide/safe.md",fmt.Sprintf("%x",sum),len(body));err!=nil{t.Fatal(err)}
	s:=store.NewNoteStore(db,d); tree,err:=s.Tree(context.Background(),p.ID);if err!=nil||len(tree)!=2||tree[1].NoteID!="n1"{t.Fatalf("%+v %v",tree,err)}
	doc,err:=s.Get(context.Background(),"n1");if err!=nil||string(doc.Body)!=string(body){t.Fatalf("%+v %v",doc,err)}
	if err:=os.WriteFile(path,[]byte("changed"),0600);err!=nil{t.Fatal(err)};_,err=s.Get(context.Background(),"n1");if !errors.Is(err,store.ErrIntegrity){t.Fatalf("got %v",err)}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run TestNoteTreeAndIntegrity -v`
Expected: FAIL because `NoteStore`, `TreeEntry`, and `ErrIntegrity` are undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

var ErrIntegrity=errors.New("note integrity check failed")
type TreeEntry struct{Path string `json:"path"`;Kind string `json:"kind"`;NoteID string `json:"note_id,omitempty"`}
type NoteDocument struct{ID string `json:"id"`;ProjectID string `json:"project_id"`;RelativePath string `json:"relative_path"`;ContentSHA256 string `json:"content_sha256"`;ByteSize int64 `json:"byte_size"`;Revision int `json:"revision"`;Body []byte `json:"body"`}
type NoteStore struct{db *sql.DB;dataDir string}
func NewNoteStore(db *sql.DB,dataDir string)*NoteStore{return &NoteStore{db:db,dataDir:dataDir}}
func (s *NoteStore) projectRoot(ctx context.Context,id string)(string,error){var v sql.NullString;if err:=s.db.QueryRowContext(ctx,`SELECT vault_id FROM projects WHERE id=?`,id).Scan(&v);err!=nil{return "",err};return layout.ProjectRoot(s.dataDir,v.String,id),nil}
func (s *NoteStore) Tree(ctx context.Context,projectID string)([]TreeEntry,error){root,e:=s.projectRoot(ctx,projectID);if e!=nil{return nil,e};ids:=map[string]string{};rows,e:=s.db.QueryContext(ctx,`SELECT relative_path,id FROM notes WHERE project_id=? AND status='ready'`,projectID);if e!=nil{return nil,e};defer rows.Close();for rows.Next(){var p,id string;if e=rows.Scan(&p,&id);e!=nil{return nil,e};ids[p]=id};out:=[]TreeEntry{};e=filepath.WalkDir(layout.SourceDir(root),func(p string,d os.DirEntry,e error)error{if e!=nil{return e};if p==layout.SourceDir(root){return nil};if d.Type()&os.ModeSymlink!=0{return ErrIntegrity};rel,e:=filepath.Rel(layout.SourceDir(root),p);if e!=nil{return e};rel=filepath.ToSlash(rel);if d.IsDir(){out=append(out,TreeEntry{Path:rel,Kind:"folder"});return nil};if filepath.Ext(rel)!=".md"||!d.Type().IsRegular(){return ErrIntegrity};id,ok:=ids[rel];if !ok{return ErrIntegrity};out=append(out,TreeEntry{Path:rel,Kind:"note",NoteID:id});return nil});sort.Slice(out,func(i,j int)bool{return out[i].Path<out[j].Path});return out,e}
func (s *NoteStore) Get(ctx context.Context,id string)(NoteDocument,error){var n NoteDocument;var vault sql.NullString;e:=s.db.QueryRowContext(ctx,`SELECT n.id,n.project_id,n.relative_path,n.content_sha256,n.byte_size,n.revision,p.vault_id FROM notes n JOIN projects p ON p.id=n.project_id WHERE n.id=? AND n.status='ready'`,id).Scan(&n.ID,&n.ProjectID,&n.RelativePath,&n.ContentSHA256,&n.ByteSize,&n.Revision,&vault);if e!=nil{return n,e};if strings.Contains(n.RelativePath,"\\"){return n,ErrIntegrity};b,e:=os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(s.dataDir,vault.String,n.ProjectID)),filepath.FromSlash(n.RelativePath)));if e!=nil{return n,ErrIntegrity};sum:=fmt.Sprintf("%x",sha256.Sum256(b));if sum!=n.ContentSHA256||int64(len(b))!=n.ByteSize{return n,ErrIntegrity};n.Body=b;return n,nil}
```

Implement handlers using the existing JSON helper. Folder creation must call `paths.ValidateRelPath`, reject a final `.md` component, and call a rooted `Mkdir` under the selected project's `source`; return `409` when it exists and never follow a symlink. Tree returns `[]TreeEntry`. Note read maps `ErrIntegrity` to `409 {"code":"integrity_error"}` and encodes `body` as a JSON string (not base64). Add note counts with `COUNT(*) FILTER (WHERE status='ready')` to project/home DTO queries.

```go
func (h NoteHandlers) get(w http.ResponseWriter,r *http.Request){n,e:=h.Notes.Get(r.Context(),r.PathValue("id"));if errors.Is(e,store.ErrIntegrity){jsonOut(w,409,map[string]string{"code":"integrity_error"});return};if e!=nil{http.Error(w,"note not found",404);return};jsonOut(w,200,map[string]any{"id":n.ID,"project_id":n.ProjectID,"relative_path":n.RelativePath,"content_sha256":n.ContentSHA256,"byte_size":n.ByteSize,"revision":n.Revision,"body":string(n.Body)})}
func (h NoteHandlers) tree(w http.ResponseWriter,r *http.Request){v,e:=h.Notes.Tree(r.Context(),r.PathValue("id"));if e!=nil{http.Error(w,e.Error(),409);return};jsonOut(w,200,v)}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/store ./internal/httpapi -run 'TestNoteTreeAndIntegrity|TestNoteHandlers' -v`
Expected: PASS; changing the file after indexing produces HTTP 409 and no metadata rewrite.

- [ ] **Step 5: Commit**

```bash
git add internal/store/notes.go internal/store/notes_test.go internal/httpapi/note_handlers.go internal/httpapi/note_handlers_test.go internal/httpapi/project_handlers.go internal/httpapi/server.go
git commit -m "feat: browse integrity-checked source notes"
```

### Task 13: Publish direct Markdown through the shared machine

**Files:**
- Create: `internal/store/direct.go`
- Create: `internal/publish/machine.go`
- Create: `internal/publish/recover.go`
- Create: `internal/httpapi/note_handlers.go`
- Test: `internal/publish/machine_test.go`

**Interfaces:**
- Consumes: `PublishInput` exactly as locked, `paths.ValidateRelPath`, `paths.MaxMarkdownBytes`, rooted no-clobber filesystem operations, Note/direct-operation schema, and `clock.Clock`.
- Produces: `type Machine struct { DB *sql.DB; DataDir string; Clock clock.Clock }`, `Run(ctx, in PublishInput) (opStatus string, noteID string, err error)`, `RecoverAll(ctx) error`, and `POST /api/v1/projects/{id}/direct-notes`. Promote kind remains rejected until Phase 5 rather than pretending success.

- [ ] **Step 1: Write the failing test**

```go
package publish_test

func TestDirectCreateIsIdempotentAndNeverOverwrites(t *testing.T) {
	d,db,p,c:=publishFixture(t);m:=publish.Machine{DB:db,DataDir:d,Clock:c}
	in:=publish.PublishInput{OpID:"op1",RequestKey:"key1",RequestFingerprint:"fp1",Kind:"direct",Body:[]byte("# One\n"),TargetProjectID:p.ID,TargetRelPath:"guide/one.md",ReviewMode:domain.ReviewMode("none"),NoteID:"n1"}
	status,noteID,err:=m.Run(context.Background(),in);if err!=nil||status!="completed"||noteID!="n1"{t.Fatalf("%s %s %v",status,noteID,err)}
	status,noteID,err=m.Run(context.Background(),in);if err!=nil||status!="completed"||noteID!="n1"{t.Fatalf("retry: %s %s %v",status,noteID,err)}
	other:=in;other.OpID="op2";other.RequestKey="key2";other.RequestFingerprint="fp2";other.NoteID="n2";other.Body=[]byte("overwrite")
	if _,_,err=m.Run(context.Background(),other);!errors.Is(err,publish.ErrConflict){t.Fatalf("got %v",err)}
	b,err:=os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(d,p.VaultID,p.ID)),"guide","one.md"));if err!=nil||string(b)!="# One\n"{t.Fatalf("%q %v",b,err)}
}

func TestDirectCreateValidation(t *testing.T){d,db,p,c:=publishFixture(t);m:=publish.Machine{DB:db,DataDir:d,Clock:c};for _,path:=range []string{"../x.md","x.txt","memory/x.md"}{_,_,err:=m.Run(context.Background(),publish.PublishInput{OpID:"o"+path,RequestKey:"k"+path,RequestFingerprint:"f"+path,Kind:"direct",Body:[]byte("x"),TargetProjectID:p.ID,TargetRelPath:path,ReviewMode:"none",NoteID:"n"+path});if err==nil{t.Fatalf("accepted %q",path)}}}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/publish -run 'TestDirectCreate' -v`
Expected: FAIL because the publish machine does not exist.

- [ ] **Step 3: Write minimal implementation**

```go
package publish

import("context";"crypto/sha256";"database/sql";"errors";"fmt";"os";"path/filepath";"strings";"github.com/rigasyahrul/personal-agent/internal/clock";"github.com/rigasyahrul/personal-agent/internal/domain";"github.com/rigasyahrul/personal-agent/internal/layout";pathcheck "github.com/rigasyahrul/personal-agent/internal/paths")
var ErrConflict=errors.New("publication conflict")
type PublishInput struct{OpID,RequestKey,RequestFingerprint string;Kind string;SessionID string;WorkspacePath string;Body []byte;TargetProjectID,TargetRelPath string;ReviewMode domain.ReviewMode;NoteID string}
type Machine struct{DB *sql.DB;DataDir string;Clock clock.Clock}
func (m *Machine) Run(ctx context.Context,in PublishInput)(string,string,error){
	if in.Kind!="direct"{return "",in.NoteID,errors.New("promote is not enabled")};clean,e:=pathcheck.ValidateRelPath(in.TargetRelPath);if e!=nil||!strings.HasSuffix(clean,".md")||len(in.Body)>pathcheck.MaxMarkdownBytes||clean=="memory"||strings.HasPrefix(clean,"memory/")||clean=="soul"||strings.HasPrefix(clean,"soul/"){return "",in.NoteID,ErrInvalid}
	var oldFP,status,note string;e=m.DB.QueryRowContext(ctx,`SELECT request_fingerprint,status,note_id FROM direct_create_operations WHERE request_key=?`,in.RequestKey).Scan(&oldFP,&status,&note);if e==nil{if oldFP!=in.RequestFingerprint{return status,note,ErrConflict};return status,note,nil};if !errors.Is(e,sql.ErrNoRows){return "",in.NoteID,e}
	var vault sql.NullString;if e=m.DB.QueryRowContext(ctx,`SELECT vault_id FROM projects WHERE id=?`,in.TargetProjectID).Scan(&vault);e!=nil{return "",in.NoteID,e};now:=m.Clock.Now().UTC();_,e=m.DB.ExecContext(ctx,`INSERT INTO direct_create_operations(id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,'accepted',?,?)`,in.OpID,in.RequestKey,in.RequestFingerprint,in.TargetProjectID,clean,in.ReviewMode,in.NoteID,now,now);if e!=nil{return "",in.NoteID,e}
	stage:=filepath.Join(m.DataDir,"staging","direct",in.OpID,"body.md");if e=os.MkdirAll(filepath.Dir(stage),0700);e==nil{e=os.WriteFile(stage,in.Body,0600)};if e!=nil{return "",in.NoteID,e};sum:=fmt.Sprintf("%x",sha256.Sum256(in.Body));_,e=m.DB.ExecContext(ctx,`UPDATE direct_create_operations SET status='frozen',frozen_sha256=?,frozen_size=?,updated_at=? WHERE id=?`,sum,len(in.Body),now,in.OpID);if e!=nil{return "",in.NoteID,e}
	tx,e:=m.DB.BeginTx(ctx,nil);if e!=nil{return "",in.NoteID,e};defer tx.Rollback();_,e=tx.ExecContext(ctx,`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES(?,?,?,'',0,'pending',0,?,?)`,in.NoteID,in.TargetProjectID,clean,now,now);if e!=nil{return "",in.NoteID,ErrConflict};if _,e=tx.ExecContext(ctx,`UPDATE direct_create_operations SET status='path_reserved',updated_at=? WHERE id=?`,now,in.OpID);e!=nil{return "",in.NoteID,e};if e=tx.Commit();e!=nil{return "",in.NoteID,e}
	dst:=filepath.Join(layout.SourceDir(layout.ProjectRoot(m.DataDir,vault.String,in.TargetProjectID)),filepath.FromSlash(clean));if e=os.MkdirAll(filepath.Dir(dst),0700);e!=nil{return "",in.NoteID,e};f,e:=os.OpenFile(dst,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if errors.Is(e,os.ErrExist){return "",in.NoteID,ErrConflict};if e!=nil{return "",in.NoteID,e};if _,e=f.Write(in.Body);e==nil{e=f.Sync()};closeErr:=f.Close();if e==nil{e=closeErr};if e!=nil{return "",in.NoteID,e};_,e=m.DB.ExecContext(ctx,`UPDATE direct_create_operations SET status='published_fs',updated_at=? WHERE id=?`,now,in.OpID);if e!=nil{return "",in.NoteID,e}
	tx,e=m.DB.BeginTx(ctx,nil);if e!=nil{return "",in.NoteID,e};defer tx.Rollback();if _,e=tx.ExecContext(ctx,`UPDATE notes SET content_sha256=?,byte_size=?,status='ready',revision=1,updated_at=? WHERE id=?`,sum,len(in.Body),now,in.NoteID);e!=nil{return "",in.NoteID,e};if in.ReviewMode=="whole"{_,e=tx.ExecContext(ctx,`INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,stage,due_at,interval_days,ease_factor,reps,lapses,row_version,status,scheduler_version) VALUES(lower(hex(randomblob(16))),?,?, 'whole',?,1,'Review this note',0,?,0,2.5,0,0,1,'active','sm2-lite-v1')`,in.TargetProjectID,in.NoteID,sum,now)}else if in.ReviewMode=="bites"{_,e=tx.ExecContext(ctx,`INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts) VALUES(lower(hex(randomblob(16))),?,?,'bites-v1','pending',0)`,in.NoteID,sum)};if e!=nil{return "",in.NoteID,e};if _,e=tx.ExecContext(ctx,`UPDATE direct_create_operations SET status='completed',updated_at=? WHERE id=?`,now,in.OpID);e!=nil{return "",in.NoteID,e};if e=tx.Commit();e!=nil{return "",in.NoteID,e};_ = os.RemoveAll(filepath.Dir(stage));return "completed",in.NoteID,nil
}
```

`RecoverAll` queries direct operations not in `completed,failed`, reconstructs immutable input from staging plus operation columns, and resumes from the recorded status; each transition must first reconcile the corresponding DB/FS artifact. The direct HTTP handler requires `Idempotency-Key`, computes `request_fingerprint = sha256(project_id + NUL + path + NUL + review_mode + NUL + body)`, preallocates UUID operation/note IDs, and maps `ErrConflict` to 409. Empty keys, invalid review modes, non-`.md`, bodies over 1 MiB, and unsafe paths return 400. Never overwrite an existing destination.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/publish ./internal/httpapi -run 'TestDirectCreate|TestDirectNoteHandler' -v`
Expected: PASS; retry returns the same note, conflicting fingerprints/paths return 409, and original bytes remain unchanged.

- [ ] **Step 5: Commit**

```bash
git add internal/store/direct.go internal/publish/machine.go internal/publish/recover.go internal/publish/machine_test.go internal/httpapi/note_handlers.go internal/httpapi/note_handlers_test.go internal/httpapi/server.go
git commit -m "feat: publish direct source notes"
```

### Task 14: Add projects and notes screens

**Files:**
- Modify: `web/index.html`
- Modify: `web/css/app.css`
- Modify: `web/js/api.js`
- Modify: `web/js/router.js`
- Modify: `web/js/app.js`
- Create: `web/js/pages/home.js`
- Create: `web/js/pages/project.js`
- Create: `web/js/pages/notes.js`
- Test: `web/js/pages/projects.test.mjs`

**Interfaces:**
- Consumes: project/home/tree/note/folder/direct-note endpoints from Tasks 11–13 and CSRF support from Phase 1.
- Produces: project cards and new-project form, project overview, source tree, note-by-ID viewer, and new Markdown file/folder forms; no edit/delete UI.

- [ ] **Step 1: Write the failing test**

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import { projectCard } from './home.js';
import { treeRows, directPayload } from './notes.js';

test('project card includes counts and vault badge', () => {
  const html = projectCard({id:'p1',name:'Go',vault_name:'Learning',note_count:2,session_count:0,due_count:1});
  assert.match(html, /Go/); assert.match(html, /Learning/); assert.match(html, /2 notes/); assert.match(html, /1 due/);
});
test('tree links notes by id, never by path', () => {
  const html = treeRows('p1', [{kind:'note',path:'guide/a.md',note_id:'n1'}]);
  assert.match(html, /notes\/n1/); assert.doesNotMatch(html, /notes\/guide/);
});
test('direct create preserves locked request fields', () => {
  assert.deepEqual(directPayload('guide/a.md','# A','whole'), {relative_path:'guide/a.md',body:'# A',review_mode:'whole'});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test web/js/pages/projects.test.mjs`
Expected: FAIL because the page modules and exports do not exist.

- [ ] **Step 3: Write minimal implementation**

```js
// web/js/api.js
export async function api(path, options={}) {
  const headers = {'Accept':'application/json', ...(options.headers||{})};
  if (options.body && typeof options.body !== 'string') { headers['Content-Type']='application/json'; options.body=JSON.stringify(options.body); }
  if (!['GET','HEAD'].includes(options.method||'GET')) headers['X-CSRF-Token']=document.cookie.split('; ').find(v=>v.startsWith('pa_csrf='))?.split('=')[1]||'';
  const response=await fetch(`/api/v1${path}`,{...options,headers}); const data=await response.json().catch(()=>({}));
  if(!response.ok) throw new Error(data.message||data.code||`HTTP ${response.status}`); return data;
}
```

```js
// web/js/pages/home.js
import {api} from '../api.js';
const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export const projectCard=p=>`<article class="card"><a href="#/projects/${encodeURIComponent(p.id)}"><h2>${esc(p.name)}</h2></a>${p.vault_name?`<span class="badge">${esc(p.vault_name)}</span>`:''}<p>${p.note_count||0} notes · ${p.session_count||0} sessions · ${p.due_count||0} due</p></article>`;
export async function renderHome(root){const data=await api('/home');root.innerHTML=`<header><h1>Projects</h1><button id="new-project">New project</button></header><form id="project-form" hidden><label>Name <input name="name" required></label><label>Vault ID (optional) <input name="vault_id"></label><button>Create</button></form><section class="cards">${data.projects.map(projectCard).join('')||'<p>Create your first project.</p>'}</section>`;root.querySelector('#new-project').onclick=()=>root.querySelector('#project-form').hidden=false;root.querySelector('#project-form').onsubmit=async e=>{e.preventDefault();const f=new FormData(e.currentTarget);const vault=f.get('vault_id').trim();const p=await api('/projects',{method:'POST',body:{name:f.get('name'),vault_id:vault||null}});location.hash=`#/projects/${p.id}`}}
```

```js
// web/js/pages/notes.js
import {api} from '../api.js';
const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export const treeRows=(projectID,entries)=>entries.map(e=>e.kind==='folder'?`<li>📁 ${esc(e.path)}</li>`:`<li>📄 <a href="#/projects/${encodeURIComponent(projectID)}/notes/${encodeURIComponent(e.note_id)}">${esc(e.path)}</a></li>`).join('');
export const directPayload=(relative_path,body,review_mode)=>({relative_path,body,review_mode});
export async function renderNotes(root,projectID,noteID){const tree=await api(`/projects/${encodeURIComponent(projectID)}/tree`);let viewer='';if(noteID){const n=await api(`/notes/${encodeURIComponent(noteID)}`);viewer=`<article><h2>${esc(n.relative_path)}</h2><pre>${esc(n.body)}</pre></article>`}root.innerHTML=`<nav><a href="#/projects/${projectID}">Overview</a></nav><h1>Notes</h1><ul>${treeRows(projectID,tree)}</ul>${viewer}<details><summary>New folder</summary><form id="folder"><input name="path" maxlength="512" required><button>Create folder</button></form></details><details><summary>New file</summary><form id="file"><input name="path" maxlength="512" pattern=".*\\.md$" required><textarea name="body" maxlength="1048576" required></textarea><select name="review"><option value="none">No review</option><option value="whole">Whole note</option><option value="bites">Bites</option></select><button>Create file</button></form></details>`;root.querySelector('#folder').onsubmit=async e=>{e.preventDefault();await api(`/projects/${projectID}/folders`,{method:'POST',body:{relative_path:new FormData(e.currentTarget).get('path')}});return renderNotes(root,projectID)};root.querySelector('#file').onsubmit=async e=>{e.preventDefault();const f=new FormData(e.currentTarget);const result=await api(`/projects/${projectID}/direct-notes`,{method:'POST',headers:{'Idempotency-Key':crypto.randomUUID()},body:directPayload(f.get('path'),f.get('body'),f.get('review'))});location.hash=`#/projects/${projectID}/notes/${result.note_id}`}}
```

```js
// web/js/pages/project.js
import {api} from '../api.js';
export async function renderProject(root,id){const p=await api(`/projects/${encodeURIComponent(id)}`);root.innerHTML=`<h1>${p.name}</h1><nav><a href="#/projects/${id}/notes">Notes</a></nav><section class="cards"><p>${p.note_count||0} notes</p><p>${p.session_count||0} sessions</p><p>${p.due_count||0} due</p></section><a class="button" href="#/projects/${id}/notes">New source file</a>`}
```

Update `router.js` to match `#/projects/:id/notes/:noteID`, `#/projects/:id/notes`, and `#/projects/:id` in that order, calling the exports above; default to `renderHome`. Update `app.js` to rerender on `hashchange`, add `<main id="app">` in `index.html`, and add responsive `.cards`, `.card`, `.badge`, form, tree, and `pre { white-space: pre-wrap }` rules to `app.css`. Render Markdown as escaped text in this phase; a vetted Markdown renderer arrives only through `components/markdown.js`, never through raw `innerHTML` from note bodies.

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test web/js/pages/projects.test.mjs && go test ./...`
Expected: PASS; tree links contain `note_id`, direct file input is `.md`-constrained and 1 MiB-limited, and all Go tests remain green.

- [ ] **Step 5: Commit**

```bash
git add web/index.html web/css/app.css web/js/api.js web/js/router.js web/js/app.js web/js/pages/home.js web/js/pages/project.js web/js/pages/notes.js web/js/pages/projects.test.mjs
git commit -m "feat: add projects and source notes UI"
```

## Phase self-check

- Covers spec §4 project layout (`source/`, reserved `memory/`/`soul/`) and rooted project placement.
- Covers §5 Vault, immutable Project placement, ready Note metadata, stable note identity, and integrity checking.
- Covers §6 direct publication through `publish.Machine`, `.md`/1 MiB/path limits, idempotency, and 409 no-clobber behavior.
- Covers §8 Home, Projects, overview, Notes tree/view, and new file/folder UI without edit/delete.
- Covers §9 F1 create project, F5 direct source file, and F6 browse by `note_id` with integrity errors.
