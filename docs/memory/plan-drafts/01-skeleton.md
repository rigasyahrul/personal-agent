## Phase 1: Skeleton

### Task 1: Go module, configuration, IDs, clock, and Makefile

**Files:**
- Create: `go.mod`, `Makefile`, `internal/config/config.go`, `internal/config/config_test.go`, `internal/ids/ids.go`, `internal/ids/ids_test.go`, `internal/clock/clock.go`, `internal/clock/clock_test.go`

**Interfaces:**
- Consumes: environment variables `PA_DATA_DIR`, `PA_ADDR`, `BOOTSTRAP_TOKEN`, `PA_SECURE_COOKIES`
- Produces: `func config.Load() (config.Config, error)`; `func ids.NewID() string`; `type clock.Clock interface{ Now() time.Time }`; `type clock.RealClock struct{}`; `func (clock.RealClock) Now() time.Time`; `type clock.FakeClock struct{ T time.Time }`; `func (*clock.FakeClock) Now() time.Time`; `func (*clock.FakeClock) Advance(time.Duration)`

- [ ] **Step 1: Write the failing tests**

```go
// internal/config/config_test.go
package config
import "testing"
func TestLoadDefaults(t *testing.T) { t.Setenv("PA_DATA_DIR", ""); t.Setenv("PA_ADDR", ""); c, err := Load(); if err != nil || c.DataDir != "./data" || c.Addr != ":8080" { t.Fatalf("%+v %v", c, err) } }

// internal/ids/ids_test.go
package ids
import ("testing"; "github.com/google/uuid")
func TestNewIDIsUUID4(t *testing.T) { u, err := uuid.Parse(NewID()); if err != nil || u.Version() != 4 { t.Fatalf("%v %v", u, err) } }

// internal/clock/clock_test.go
package clock
import ("testing"; "time")
func TestFakeClockAdvance(t *testing.T) { f := &FakeClock{T: time.Unix(0, 0)}; f.Advance(time.Minute); if !f.Now().Equal(time.Unix(60, 0)) { t.Fatal(f.Now()) } }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config ./internal/ids ./internal/clock -v`
Expected: FAIL because the module and packages do not exist.

- [ ] **Step 3: Write the minimal implementation**

```go
// go.mod
module github.com/rigasyahrul/personal-agent

go 1.24

require github.com/google/uuid v1.6.0

// internal/config/config.go
package config
import ("errors"; "os")
type Config struct { DataDir, Addr, BootstrapToken string; SecureCookies bool }
func Load() (Config, error) { c := Config{DataDir: os.Getenv("PA_DATA_DIR"), Addr: os.Getenv("PA_ADDR"), BootstrapToken: os.Getenv("BOOTSTRAP_TOKEN"), SecureCookies: os.Getenv("PA_SECURE_COOKIES") != "false"}; if c.DataDir == "" { c.DataDir = "./data" }; if c.Addr == "" { c.Addr = ":8080" }; if c.Addr[0] != ':' { return c, errors.New("PA_ADDR must begin with ':'") }; return c, nil }

// internal/ids/ids.go
package ids
import "github.com/google/uuid"
func NewID() string { return uuid.NewString() }

// internal/clock/clock.go
package clock
import "time"
type Clock interface{ Now() time.Time }
type RealClock struct{}
func (RealClock) Now() time.Time { return time.Now().UTC() }
type FakeClock struct{ T time.Time }
func (f *FakeClock) Now() time.Time { return f.T }
func (f *FakeClock) Advance(d time.Duration) { f.T = f.T.Add(d) }

// Makefile
.PHONY: test run build
test:
	go test ./...
run:
	go run ./cmd/personal-agent
build:
	go build ./cmd/personal-agent
```

Run: `go mod tidy`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config ./internal/ids ./internal/clock -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum Makefile internal/config internal/ids internal/clock
git commit -m "chore: initialize Go skeleton"
```

### Task 2: Validate relative paths

**Files:**
- Create: `internal/paths/paths.go`, `internal/paths/paths_test.go`

**Interfaces:**
- Consumes: logical POSIX UTF-8 relative paths
- Produces: `type paths.PathError struct{ Code, Message string }`; `func paths.ValidateRelPath(string) (string, error)`; constants `MaxPathBytes = 512`, `MaxDepth = 16`, `MaxComponentBytes = 255`, `MaxMarkdownBytes = 1 << 20`

- [ ] **Step 1: Write the failing test**

```go
package paths
import "testing"
func TestValidateRelPath(t *testing.T) {
	valid := map[string]string{"notes/a.md":"notes/a.md", "a//b":"a/b"}
	for in, want := range valid { got, err := ValidateRelPath(in); if err != nil || got != want { t.Errorf("%q: %q %v", in, got, err) } }
	bad := []string{"", ".", "..", "../a", "/a", "a/../b", "a/./b", "a\x00b", "a\nb"}
	for _, in := range bad { if _, err := ValidateRelPath(in); err == nil { t.Errorf("accepted %q", in) } }
	if _, err := ValidateRelPath(string(make([]byte, MaxPathBytes+1))); err == nil { t.Fatal("accepted long path") }
	deep := "a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a/a"; if _, err := ValidateRelPath(deep); err == nil { t.Fatal("accepted deep path") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/paths -run TestValidateRelPath -v`
Expected: FAIL with `undefined: ValidateRelPath`.

- [ ] **Step 3: Write minimal implementation**

```go
package paths
import ("fmt"; "path"; "strings"; "unicode/utf8")
const ( MaxPathBytes = 512; MaxDepth = 16; MaxComponentBytes = 255; MaxMarkdownBytes = 1 << 20 )
type PathError struct{ Code, Message string }
func (e *PathError) Error() string { return e.Code + ": " + e.Message }
func reject(code, message string) (string, error) { return "", &PathError{Code: code, Message: message} }
func ValidateRelPath(p string) (string, error) {
	if p == "" || !utf8.ValidString(p) { return reject("invalid_path", "path must be non-empty UTF-8") }
	if len(p) > MaxPathBytes || strings.HasPrefix(p, "/") { return reject("invalid_path", "path is absolute or too long") }
	for _, r := range p { if r < 0x20 || r == 0x7f { return reject("invalid_path", "control characters are forbidden") } }
	for _, c := range strings.Split(p, "/") { if c == "." || c == ".." { return reject("invalid_path", "dot components are forbidden") } }
	clean := path.Clean(p); parts := strings.Split(clean, "/")
	if clean == "." || len(parts) > MaxDepth { return reject("invalid_path", "path is empty or too deep") }
	for _, c := range parts { if len(c) == 0 || len(c) > MaxComponentBytes { return reject("invalid_path", fmt.Sprintf("invalid component %q", c)) } }
	return clean, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/paths -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/paths
git commit -m "feat: validate relative paths"
```

### Task 3: SQLite WAL and complete initial migration

**Files:**
- Create: `internal/db/db.go`, `internal/db/migrations/001_init.sql`, `internal/db/migrate_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `context.Context`, database file path
- Produces: `func db.Open(context.Context, string) (*sql.DB, error)`; embedded, idempotently recorded migration `001_init.sql`

- [ ] **Step 1: Write the failing test**

```go
package db
import ("context"; "path/filepath"; "testing")
func TestOpenMigratesAllTablesAndWAL(t *testing.T) { d, err := Open(context.Background(), filepath.Join(t.TempDir(), "db", "app.sqlite")); if err != nil { t.Fatal(err) }; defer d.Close(); var mode string; if err := d.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" { t.Fatalf("%q %v", mode, err) }; tables := []string{"owner","settings","auth_sessions","vaults","projects","sessions","agent_runs","messages","notes","promote_ops","direct_ops","review_pending","review_items","review_events","backup_runs"}; for _, name := range tables { var n int; if err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&n); err != nil || n != 1 { t.Fatalf("table %s: %d %v", name, n, err) } }; if _, err := Open(context.Background(), filepath.Join(t.TempDir(), "other.sqlite")); err != nil { t.Fatal(err) } }
func TestSessionScopeCheck(t *testing.T) { d, _ := Open(context.Background(), filepath.Join(t.TempDir(), "x.sqlite")); defer d.Close(); _, err := d.Exec(`INSERT INTO sessions(id,home,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s','project','active','p','m','{}','{}','t','x','x')`); if err == nil { t.Fatal("invalid project scope accepted") } }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db -v`
Expected: FAIL because `Open` is undefined.

- [ ] **Step 3: Write minimal implementation and the full schema**

```go
// internal/db/db.go
package db
import ("context"; "database/sql"; "embed"; "fmt"; "os"; "path/filepath"; _ "modernc.org/sqlite")
//go:embed migrations/*.sql
var migrations embed.FS
func Open(ctx context.Context, file string) (*sql.DB, error) { if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil { return nil, err }; d, err := sql.Open("sqlite", file); if err != nil { return nil, err }; d.SetMaxOpenConns(1); for _, q := range []string{"PRAGMA foreign_keys=ON", "PRAGMA journal_mode=WAL", "PRAGMA busy_timeout=5000"} { if _, err = d.ExecContext(ctx, q); err != nil { d.Close(); return nil, err } }; if _, err = d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil { d.Close(); return nil, err }; var n int; if err = d.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='001'`).Scan(&n); err != nil { d.Close(); return nil, err }; if n == 0 { b, e := migrations.ReadFile("migrations/001_init.sql"); if e != nil { d.Close(); return nil, e }; tx, e := d.BeginTx(ctx, nil); if e == nil { _, e = tx.ExecContext(ctx, string(b)) }; if e == nil { _, e = tx.ExecContext(ctx, `INSERT INTO schema_migrations VALUES('001',strftime('%Y-%m-%dT%H:%M:%fZ','now'))`) }; if e == nil { e = tx.Commit() } else { tx.Rollback() }; if e != nil { d.Close(); return nil, fmt.Errorf("migration 001: %w", e) } }; return d, nil }
```

```sql
-- internal/db/migrations/001_init.sql
CREATE TABLE owner (id INTEGER PRIMARY KEY CHECK(id=1), password_hash TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE settings (id INTEGER PRIMARY KEY CHECK(id=1), timezone TEXT NOT NULL DEFAULT 'UTC', default_provider TEXT, default_model_id TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
INSERT INTO settings(id,timezone,created_at,updated_at) VALUES(1,'UTC',strftime('%Y-%m-%dT%H:%M:%fZ','now'),strftime('%Y-%m-%dT%H:%M:%fZ','now'));
CREATE TABLE auth_sessions (token_hash TEXT PRIMARY KEY, owner_id INTEGER NOT NULL DEFAULT 1 REFERENCES owner(id) ON DELETE CASCADE, csrf_token TEXT NOT NULL, expires_at TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE vaults (id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE projects (id TEXT PRIMARY KEY, vault_id TEXT REFERENCES vaults(id) ON DELETE RESTRICT, name TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE sessions (id TEXT PRIMARY KEY, home TEXT NOT NULL CHECK(home IN ('global','vault','project')), vault_id TEXT REFERENCES vaults(id) ON DELETE RESTRICT, project_id TEXT REFERENCES projects(id) ON DELETE RESTRICT, status TEXT NOT NULL CHECK(status IN ('active','terminal')), provider TEXT NOT NULL, model_id TEXT NOT NULL, model_parameters_json TEXT NOT NULL CHECK(json_valid(model_parameters_json)), tool_grants_json TEXT NOT NULL CHECK(json_valid(tool_grants_json)), title TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT, CHECK((home='global' AND vault_id IS NULL AND project_id IS NULL) OR (home='vault' AND vault_id IS NOT NULL AND project_id IS NULL) OR (home='project' AND project_id IS NOT NULL)));
CREATE TRIGGER sessions_project_vault_insert BEFORE INSERT ON sessions WHEN NEW.home='project' AND NOT EXISTS(SELECT 1 FROM projects p WHERE p.id=NEW.project_id AND p.vault_id IS NEW.vault_id) BEGIN SELECT RAISE(ABORT,'session project vault mismatch'); END;
CREATE TRIGGER sessions_immutable_update BEFORE UPDATE OF home,vault_id,project_id,provider,model_id,model_parameters_json ON sessions BEGIN SELECT RAISE(ABORT,'immutable session fields'); END;
CREATE TABLE agent_runs (id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id), request_key TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('queued','running','completed','failed','cancelled')), started_at TEXT, completed_at TEXT, error TEXT, created_at TEXT NOT NULL, UNIQUE(session_id,request_key));
CREATE UNIQUE INDEX one_active_run ON agent_runs(session_id) WHERE status IN ('queued','running');
CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL REFERENCES sessions(id), run_id TEXT REFERENCES agent_runs(id), sequence INTEGER NOT NULL, role TEXT NOT NULL CHECK(role IN ('system','user','assistant','tool')), content TEXT NOT NULL, tool_calls_json TEXT CHECK(tool_calls_json IS NULL OR json_valid(tool_calls_json)), tool_call_id TEXT, status TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(session_id,sequence));
CREATE TABLE notes (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), relative_path TEXT NOT NULL, content_sha256 TEXT, byte_size INTEGER CHECK(byte_size IS NULL OR byte_size>=0), status TEXT NOT NULL CHECK(status IN ('pending','ready','failed')), origin_session_id TEXT REFERENCES sessions(id), origin_workspace_path TEXT, revision INTEGER NOT NULL DEFAULT 0 CHECK(revision>=0), created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(project_id,relative_path));
CREATE TABLE promote_ops (id TEXT PRIMARY KEY, request_key TEXT NOT NULL UNIQUE, request_fingerprint TEXT NOT NULL, session_id TEXT NOT NULL REFERENCES sessions(id), workspace_path TEXT NOT NULL, target_project_id TEXT NOT NULL REFERENCES projects(id), target_relative_path TEXT NOT NULL, review_mode TEXT NOT NULL CHECK(review_mode IN ('none','whole','bites')), note_id TEXT NOT NULL, frozen_sha256 TEXT, frozen_size INTEGER, status TEXT NOT NULL CHECK(status IN ('accepted','frozen','path_reserved','published_fs','finalized','review_enqueued','completed','failed')), error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE direct_ops (id TEXT PRIMARY KEY, request_key TEXT NOT NULL UNIQUE, request_fingerprint TEXT NOT NULL, target_project_id TEXT NOT NULL REFERENCES projects(id), target_relative_path TEXT NOT NULL, review_mode TEXT NOT NULL CHECK(review_mode IN ('none','whole','bites')), note_id TEXT NOT NULL, frozen_sha256 TEXT, frozen_size INTEGER, status TEXT NOT NULL CHECK(status IN ('accepted','frozen','path_reserved','published_fs','finalized','review_enqueued','completed','failed')), error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE review_pending (id TEXT PRIMARY KEY, note_id TEXT NOT NULL REFERENCES notes(id), source_sha256 TEXT NOT NULL, generator_version TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('pending','leased','completed','failed')), attempts INTEGER NOT NULL DEFAULT 0, lease_until TEXT, last_error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE UNIQUE INDEX review_pending_generation ON review_pending(note_id,source_sha256,generator_version) WHERE status IN ('pending','leased');
CREATE TABLE review_items (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), note_id TEXT NOT NULL REFERENCES notes(id), kind TEXT NOT NULL CHECK(kind IN ('whole','bite')), source_sha256 TEXT NOT NULL, source_revision INTEGER NOT NULL, prompt TEXT NOT NULL, answer TEXT, generation_id TEXT REFERENCES review_pending(id), ordinal INTEGER, stage INTEGER NOT NULL DEFAULT 0, due_at TEXT NOT NULL, interval_days REAL NOT NULL DEFAULT 0, ease_factor REAL NOT NULL DEFAULT 2.5 CHECK(ease_factor>=1.3), reps INTEGER NOT NULL DEFAULT 0, lapses INTEGER NOT NULL DEFAULT 0, last_reviewed_at TEXT, row_version INTEGER NOT NULL DEFAULT 1, status TEXT NOT NULL CHECK(status IN ('active','suspended','retired')), scheduler_version TEXT NOT NULL, CHECK((kind='whole' AND generation_id IS NULL AND ordinal IS NULL) OR (kind='bite' AND answer IS NOT NULL AND generation_id IS NOT NULL AND ordinal IS NOT NULL)), UNIQUE(generation_id,ordinal));
CREATE UNIQUE INDEX review_whole_active ON review_items(note_id,source_revision) WHERE kind='whole' AND status='active';
CREATE TABLE review_events (id TEXT PRIMARY KEY, review_item_id TEXT NOT NULL REFERENCES review_items(id), request_key TEXT NOT NULL UNIQUE, rating TEXT NOT NULL CHECK(rating IN ('again','hard','good','easy')), previous_state_json TEXT NOT NULL CHECK(json_valid(previous_state_json)), resulting_state_json TEXT NOT NULL CHECK(json_valid(resulting_state_json)), scheduler_version TEXT NOT NULL, reviewed_at TEXT NOT NULL, duration_ms INTEGER CHECK(duration_ms IS NULL OR duration_ms>=0));
CREATE TABLE backup_runs (id TEXT PRIMARY KEY, status TEXT NOT NULL CHECK(status IN ('running','succeeded','failed')), cutoff_at TEXT NOT NULL, local_path TEXT, object_key TEXT, manifest_hash TEXT, started_at TEXT NOT NULL, completed_at TEXT, error TEXT);
```

Run: `go get modernc.org/sqlite && go mod tidy`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/db -v`
Expected: PASS with WAL enabled, all 15 domain tables present, and invalid scope rejected.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/db
git commit -m "feat: add SQLite schema and migrations"
```

### Task 4: Argon2id passwords and session tokens

**Files:**
- Create: `internal/auth/password.go`, `internal/auth/password_test.go`, `internal/auth/session.go`, `internal/auth/session_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: plaintext password, PHC hash string, cryptographic random source
- Produces: `func auth.HashPassword(string) (string, error)`; `func auth.CheckPassword(string, string) bool`; `func auth.NewSessionToken() string`; `func auth.TokenHash(string) string`

- [ ] **Step 1: Write the failing tests**

```go
package auth
import ("strings"; "testing")
func TestPasswordRoundTrip(t *testing.T) { h, err := HashPassword("correct horse battery staple"); if err != nil { t.Fatal(err) }; if !strings.HasPrefix(h,"$argon2id$") || !CheckPassword(h,"correct horse battery staple") || CheckPassword(h,"wrong") { t.Fatal("argon2id verification failed") } }
func TestSessionTokens(t *testing.T) { a, b := NewSessionToken(), NewSessionToken(); if a == "" || a == b || TokenHash(a) == a || TokenHash(a) != TokenHash(a) { t.Fatal("unsafe token") } }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/auth -run 'TestPasswordRoundTrip|TestSessionTokens' -v`
Expected: FAIL with undefined functions.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/auth/password.go
package auth
import ("crypto/rand"; "crypto/subtle"; "encoding/base64"; "fmt"; "strings"; "golang.org/x/crypto/argon2")
func HashPassword(pw string) (string,error) { salt := make([]byte,16); if _,err:=rand.Read(salt); err!=nil{return "",err}; sum:=argon2.IDKey([]byte(pw),salt,3,64*1024,2,32); return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s",base64.RawStdEncoding.EncodeToString(salt),base64.RawStdEncoding.EncodeToString(sum)),nil }
func CheckPassword(encoded,pw string) bool { p:=strings.Split(encoded,"$"); if len(p)!=6 || p[1]!="argon2id" || p[2]!="v=19" || p[3]!="m=65536,t=3,p=2" {return false}; salt,e1:=base64.RawStdEncoding.DecodeString(p[4]); want,e2:=base64.RawStdEncoding.DecodeString(p[5]); if e1!=nil||e2!=nil{return false}; got:=argon2.IDKey([]byte(pw),salt,3,64*1024,2,32); return subtle.ConstantTimeCompare(got,want)==1 }

// internal/auth/session.go
package auth
import ("crypto/rand"; "crypto/sha256"; "encoding/base64"; "encoding/hex")
func NewSessionToken() string { b:=make([]byte,32); if _,err:=rand.Read(b); err!=nil { panic(err) }; return base64.RawURLEncoding.EncodeToString(b) }
func TokenHash(token string) string { h:=sha256.Sum256([]byte(token)); return hex.EncodeToString(h[:]) }
```

Run: `go get golang.org/x/crypto && go mod tidy`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/auth
git commit -m "feat: add argon2id authentication primitives"
```

### Task 5: Owner bootstrap, login, logout, me, and CSRF

**Files:**
- Create: `internal/auth/bootstrap.go`, `internal/auth/bootstrap_test.go`, `internal/auth/csrf.go`, `internal/httpapi/auth_handlers.go`, `internal/httpapi/middleware.go`, `internal/httpapi/auth_handlers_test.go`

**Interfaces:**
- Consumes: `*sql.DB`, `clock.Clock`, bootstrap token, `pa_session` and `pa_csrf` cookies, `X-CSRF-Token`
- Produces: `func auth.Bootstrap(context.Context, *sql.DB, string, string, string, time.Time) error`; `func httpapi.AuthRoutes(*http.ServeMux, AuthDeps)`; `func httpapi.RequireAuth(*sql.DB, http.Handler) http.Handler`; `func httpapi.RequireCSRF(http.Handler) http.Handler`; endpoints `GET /api/v1/setup/status`, `POST /api/v1/setup/bootstrap`, `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`

- [ ] **Step 1: Write the failing integration test**

```go
package httpapi
import ("bytes"; "context"; "encoding/json"; "net/http"; "net/http/httptest"; "path/filepath"; "testing"; "time"; "github.com/rigasyahrul/personal-agent/internal/clock"; database "github.com/rigasyahrul/personal-agent/internal/db")
func TestBootstrapLoginMeLogoutAndCSRF(t *testing.T) { d,err:=database.Open(context.Background(),filepath.Join(t.TempDir(),"a.sqlite")); if err!=nil{t.Fatal(err)}; defer d.Close(); mux:=http.NewServeMux(); AuthRoutes(mux,AuthDeps{DB:d,Clock:&clock.FakeClock{T:time.Unix(1000,0).UTC()},BootstrapToken:"secret",SecureCookies:false}); post:=func(path,body string,cookies []*http.Cookie,csrf string)*httptest.ResponseRecorder{ r:=httptest.NewRequest(http.MethodPost,path,bytes.NewBufferString(body)); r.Header.Set("Content-Type","application/json"); if csrf!=""{r.Header.Set("X-CSRF-Token",csrf)}; for _,c:=range cookies{r.AddCookie(c)}; w:=httptest.NewRecorder(); mux.ServeHTTP(w,r); return w }; w:=post("/api/v1/setup/bootstrap",`{"token":"secret","password":"long-enough-password"}`,nil,""); if w.Code!=201{t.Fatal(w.Code,w.Body.String())}; if post("/api/v1/setup/bootstrap",`{"token":"secret","password":"another-long-password"}`,nil,"").Code!=409{t.Fatal("second bootstrap accepted")}; w=post("/api/v1/auth/login",`{"password":"long-enough-password"}`,nil,""); if w.Code!=204{t.Fatal(w.Code,w.Body.String())}; cookies:=w.Result().Cookies(); var csrf string; for _,c:=range cookies{if c.Name=="pa_csrf"{csrf=c.Value}}; r:=httptest.NewRequest(http.MethodGet,"/api/v1/auth/me",nil); for _,c:=range cookies{r.AddCookie(c)}; me:=httptest.NewRecorder(); mux.ServeHTTP(me,r); var out map[string]any; json.NewDecoder(me.Body).Decode(&out); if me.Code!=200 || out["owner"]!=true{t.Fatal(me.Code,out)}; if post("/api/v1/auth/logout",`{}`,cookies,"wrong").Code!=403{t.Fatal("csrf bypass")}; if post("/api/v1/auth/logout",`{}`,cookies,csrf).Code!=204{t.Fatal("logout failed")} }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestBootstrapLoginMeLogoutAndCSRF -v`
Expected: FAIL because `AuthRoutes` and `AuthDeps` are undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/auth/bootstrap.go
package auth
import ("context"; "database/sql"; "errors"; "time")
var ErrBootstrapped=errors.New("owner already bootstrapped"); var ErrBootstrapToken=errors.New("invalid bootstrap token")
func Bootstrap(ctx context.Context,d *sql.DB,configured,provided,pw string,now time.Time) error { var n int; if err:=d.QueryRowContext(ctx,"SELECT count(*) FROM owner").Scan(&n);err!=nil{return err}; if n!=0{return ErrBootstrapped}; if configured==""||provided!=configured{return ErrBootstrapToken}; h,err:=HashPassword(pw);if err!=nil{return err}; _,err=d.ExecContext(ctx,"INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,?,?,?)",h,now.Format(time.RFC3339Nano),now.Format(time.RFC3339Nano));return err }

// internal/auth/csrf.go
package auth
import "crypto/subtle"
func ValidCSRF(cookie,header string) bool { return cookie!=""&&len(cookie)==len(header)&&subtle.ConstantTimeCompare([]byte(cookie),[]byte(header))==1 }

// internal/httpapi/middleware.go
package httpapi
import ("context"; "database/sql"; "net/http"; "time"; "github.com/rigasyahrul/personal-agent/internal/auth")
type ownerKey struct{}
func RequireAuth(d *sql.DB,next http.Handler) http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){c,e:=r.Cookie("pa_session");if e!=nil{http.Error(w,"unauthorized",401);return};var expiry string;if d.QueryRowContext(r.Context(),"SELECT expires_at FROM auth_sessions WHERE token_hash=?",auth.TokenHash(c.Value)).Scan(&expiry)!=nil{http.Error(w,"unauthorized",401);return};t,e:=time.Parse(time.RFC3339Nano,expiry);if e!=nil||!t.After(time.Now().UTC()){http.Error(w,"unauthorized",401);return};next.ServeHTTP(w,r.WithContext(context.WithValue(r.Context(),ownerKey{},true)))})}
func RequireCSRF(next http.Handler) http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){c,e:=r.Cookie("pa_csrf");if e!=nil||!auth.ValidCSRF(c.Value,r.Header.Get("X-CSRF-Token")){http.Error(w,"csrf",403);return};next.ServeHTTP(w,r)})}
```

```go
// internal/httpapi/auth_handlers.go
package httpapi
import ("database/sql"; "encoding/json"; "errors"; "net/http"; "time"; "github.com/rigasyahrul/personal-agent/internal/auth"; "github.com/rigasyahrul/personal-agent/internal/clock")
type AuthDeps struct{DB *sql.DB;Clock clock.Clock;BootstrapToken string;SecureCookies bool}
func AuthRoutes(m *http.ServeMux,d AuthDeps){ write:=func(w http.ResponseWriter,v any){w.Header().Set("Content-Type","application/json");json.NewEncoder(w).Encode(v)}; m.HandleFunc("GET /api/v1/setup/status",func(w http.ResponseWriter,r *http.Request){var n int;d.DB.QueryRowContext(r.Context(),"SELECT count(*) FROM owner").Scan(&n);write(w,map[string]bool{"bootstrapped":n==1})});m.HandleFunc("POST /api/v1/setup/bootstrap",func(w http.ResponseWriter,r *http.Request){var in struct{Token,Password string};if json.NewDecoder(r.Body).Decode(&in)!=nil||len(in.Password)<12{http.Error(w,"invalid request",400);return};err:=auth.Bootstrap(r.Context(),d.DB,d.BootstrapToken,in.Token,in.Password,d.Clock.Now());if errors.Is(err,auth.ErrBootstrapped){http.Error(w,"already bootstrapped",409);return};if err!=nil{http.Error(w,"forbidden",403);return};w.WriteHeader(201)});m.HandleFunc("POST /api/v1/auth/login",func(w http.ResponseWriter,r *http.Request){var in struct{Password string};json.NewDecoder(r.Body).Decode(&in);var h string;if d.DB.QueryRowContext(r.Context(),"SELECT password_hash FROM owner WHERE id=1").Scan(&h)!=nil||!auth.CheckPassword(h,in.Password){http.Error(w,"unauthorized",401);return};token,csrf:=auth.NewSessionToken(),auth.NewSessionToken();now:=d.Clock.Now();_,e:=d.DB.ExecContext(r.Context(),"INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)",auth.TokenHash(token),csrf,now.Add(30*24*time.Hour).Format(time.RFC3339Nano),now.Format(time.RFC3339Nano));if e!=nil{http.Error(w,"session",500);return};http.SetCookie(w,&http.Cookie{Name:"pa_session",Value:token,Path:"/",HttpOnly:true,Secure:d.SecureCookies,SameSite:http.SameSiteLaxMode});http.SetCookie(w,&http.Cookie{Name:"pa_csrf",Value:csrf,Path:"/",Secure:d.SecureCookies,SameSite:http.SameSiteLaxMode});w.WriteHeader(204)});m.Handle("GET /api/v1/auth/me",RequireAuth(d.DB,http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){write(w,map[string]bool{"owner":true})})));m.Handle("POST /api/v1/auth/logout",RequireAuth(d.DB,RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){c,_:=r.Cookie("pa_session");d.DB.ExecContext(r.Context(),"DELETE FROM auth_sessions WHERE token_hash=?",auth.TokenHash(c.Value));http.SetCookie(w,&http.Cookie{Name:"pa_session",Path:"/",MaxAge:-1,HttpOnly:true,Secure:d.SecureCookies});w.WriteHeader(204)})))}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth ./internal/httpapi -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth internal/httpapi
git commit -m "feat: add owner bootstrap and browser auth"
```

### Task 6: Health, setup status, empty Home, and server wiring

**Files:**
- Create: `internal/httpapi/health.go`, `internal/httpapi/server.go`, `internal/httpapi/server_test.go`, `internal/app/app.go`, `cmd/personal-agent/main.go`

**Interfaces:**
- Consumes: `config.Config`, `*sql.DB`, `clock.Clock`, static `http.FileSystem`
- Produces: `func httpapi.New(ServerDeps) http.Handler`; `func app.New(context.Context, config.Config) (*app.App, error)`; `func (*app.App) Handler() http.Handler`; endpoints `GET /health`, `GET /api/v1/home`

- [ ] **Step 1: Write the failing test**

```go
package httpapi
import ("context"; "encoding/json"; "net/http"; "net/http/httptest"; "path/filepath"; "testing"; "time"; "github.com/rigasyahrul/personal-agent/internal/clock"; database "github.com/rigasyahrul/personal-agent/internal/db")
func TestHealthSetupAndEmptyHome(t *testing.T){dir:=t.TempDir();d,err:=database.Open(context.Background(),filepath.Join(dir,"db","a.sqlite"));if err!=nil{t.Fatal(err)};defer d.Close();h:=New(ServerDeps{DB:d,DataDir:dir,Clock:&clock.FakeClock{T:time.Unix(0,0)},BootstrapToken:"x"});for _,path:=range []string{"/health","/api/v1/setup/status","/api/v1/home"}{r:=httptest.NewRequest(http.MethodGet,path,nil);w:=httptest.NewRecorder();h.ServeHTTP(w,r);if w.Code!=200{t.Fatalf("%s: %d %s",path,w.Code,w.Body.String())};var v map[string]any;if json.NewDecoder(w.Body).Decode(&v)!=nil{t.Fatalf("%s not JSON",path)}}}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestHealthSetupAndEmptyHome -v`
Expected: FAIL because `New` and `ServerDeps` are undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/httpapi/health.go
package httpapi
import ("encoding/json"; "net/http"; "os"; "path/filepath")
func healthHandler(dataDir string)http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){p:=filepath.Join(dataDir,".health-write");err:=os.WriteFile(p,[]byte("ok"),0600);if err==nil{err=os.Remove(p)};w.Header().Set("Content-Type","application/json");if err!=nil{w.WriteHeader(503)};json.NewEncoder(w).Encode(map[string]any{"ok":err==nil,"storage_writable":err==nil})}}

// internal/httpapi/server.go
package httpapi
import ("database/sql"; "encoding/json"; "net/http"; "github.com/rigasyahrul/personal-agent/internal/clock")
type ServerDeps struct{DB *sql.DB;DataDir string;Clock clock.Clock;BootstrapToken string;SecureCookies bool;Static http.FileSystem}
func New(d ServerDeps)http.Handler{m:=http.NewServeMux();AuthRoutes(m,AuthDeps{DB:d.DB,Clock:d.Clock,BootstrapToken:d.BootstrapToken,SecureCookies:d.SecureCookies});m.Handle("GET /health",healthHandler(d.DataDir));m.HandleFunc("GET /api/v1/home",func(w http.ResponseWriter,r *http.Request){w.Header().Set("Content-Type","application/json");json.NewEncoder(w).Encode(map[string]any{"projects":[]any{},"due_count":0,"last_project":nil})});if d.Static!=nil{m.Handle("GET /",http.FileServer(d.Static))};return m}

// internal/app/app.go
package app
import ("context"; "net/http"; "path/filepath"; "github.com/rigasyahrul/personal-agent/internal/clock"; "github.com/rigasyahrul/personal-agent/internal/config"; database "github.com/rigasyahrul/personal-agent/internal/db"; "github.com/rigasyahrul/personal-agent/internal/httpapi")
type App struct{handler http.Handler}
func New(ctx context.Context,c config.Config)(*App,error){d,e:=database.Open(ctx,filepath.Join(c.DataDir,"db","personal-agent.sqlite"));if e!=nil{return nil,e};return &App{handler:httpapi.New(httpapi.ServerDeps{DB:d,DataDir:c.DataDir,Clock:clock.RealClock{},BootstrapToken:c.BootstrapToken,SecureCookies:c.SecureCookies,Static:http.Dir("web")})},nil}
func(a *App)Handler()http.Handler{return a.handler}

// cmd/personal-agent/main.go
package main
import("context";"log";"net/http";"github.com/rigasyahrul/personal-agent/internal/app";"github.com/rigasyahrul/personal-agent/internal/config")
func main(){c,e:=config.Load();if e!=nil{log.Fatal(e)};a,e:=app.New(context.Background(),c);if e!=nil{log.Fatal(e)};log.Printf("listening on %s",c.Addr);log.Fatal(http.ListenAndServe(c.Addr,a.Handler()))}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi internal/app cmd/personal-agent
git commit -m "feat: serve health setup and empty home APIs"
```

### Task 7: Static empty Home shell

**Files:**
- Create: `web/index.html`, `web/css/app.css`, `web/js/api.js`, `web/js/router.js`, `web/js/app.js`, `web/js/pages/home.js`, `internal/httpapi/static_test.go`

**Interfaces:**
- Consumes: `GET /api/v1/home`, `GET /health`, `GET /api/v1/setup/status`
- Produces: static SPA at `GET /`; `api.get(path)`; `home.render(element)`

- [ ] **Step 1: Write the failing test**

```go
package httpapi
import("net/http";"net/http/httptest";"os";"strings";"testing")
func TestStaticShell(t *testing.T){if _,err:=os.Stat("../../web/index.html");err!=nil{t.Fatal(err)};h:=http.FileServer(http.Dir("../../web"));r:=httptest.NewRequest("GET","/",nil);w:=httptest.NewRecorder();h.ServeHTTP(w,r);if w.Code!=200||!strings.Contains(w.Body.String(),"Personal Agent")||!strings.Contains(w.Body.String(),`type="module"`){t.Fatalf("%d %s",w.Code,w.Body.String())}}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestStaticShell -v`
Expected: FAIL because `web/index.html` does not exist.

- [ ] **Step 3: Write minimal implementation**

```html
<!-- web/index.html -->
<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Personal Agent</title><link rel="stylesheet" href="/css/app.css"></head><body><header><h1>Personal Agent</h1><span id="health">Checking storage…</span></header><main id="app" aria-live="polite">Loading…</main><script type="module" src="/js/app.js"></script></body></html>
```

```css
/* web/css/app.css */
:root{font-family:system-ui,sans-serif;color:#172033;background:#f6f7fb}body{max-width:64rem;margin:auto;padding:1rem}header{display:flex;justify-content:space-between;align-items:center}main{background:white;border-radius:.75rem;padding:2rem;box-shadow:0 2px 12px #0001}.muted{color:#667085}button{padding:.65rem 1rem;border:0;border-radius:.5rem;background:#2457d6;color:white}
```

```js
// web/js/api.js
export async function get(path){const response=await fetch(path,{headers:{Accept:'application/json'}});if(!response.ok)throw new Error(`${response.status} ${await response.text()}`);return response.json()}

// web/js/router.js
export function route(){return location.pathname==='/'?'home':'not-found'}

// web/js/pages/home.js
import{get}from'../api.js';
export async function render(root){const home=await get('/api/v1/home');root.innerHTML=home.projects.length?'<h2>Projects</h2>':`<h2>Your learning home</h2><p class="muted">No projects yet. Create your first project to begin collecting notes and sessions.</p><button disabled>Create project (next phase)</button>`}

// web/js/app.js
import{get}from'./api.js';import{route}from'./router.js';import{render as home}from'./pages/home.js';
const root=document.querySelector('#app');Promise.all([get('/health'),get('/api/v1/setup/status')]).then(([health,setup])=>{document.querySelector('#health').textContent=`Storage ${health.storage_writable?'ready':'unavailable'} · ${setup.bootstrapped?'Owner ready':'Setup required'}`}).catch(e=>document.querySelector('#health').textContent=e.message);if(route()==='home')home(root).catch(e=>root.textContent=e.message);else root.textContent='Not found';
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... && test "$(grep -c '/api/v1/home' web/js/pages/home.js)" -eq 1`
Expected: PASS and the Home module contains exactly one Home API reference.

- [ ] **Step 5: Commit**

```bash
git add web internal/httpapi/static_test.go
git commit -m "feat: add empty Home web shell"
```

### Task 8: Container deployment and Go 1.24 orb setup

**Files:**
- Create: `deploy/Dockerfile`, `deploy/docker-compose.yml`, `deploy/Caddyfile`, `deploy/.env.example`, `README.md`, `deploy/deploy_test.go`
- Modify: `.agents/setup`

**Interfaces:**
- Consumes: module build, port `8080`, `PA_DATA_DIR`, `BOOTSTRAP_TOKEN`, `PA_DOMAIN`
- Produces: `personal-agent` container, persistent `pa-data` volume, optional-domain Caddy reverse proxy, documented local startup, orb installation of Go 1.24+

- [ ] **Step 1: Write the failing deployment test**

```go
package deploy_test
import("os";"strings";"testing")
func TestDeploymentFiles(t *testing.T){checks:=map[string][]string{"Dockerfile":{"golang:1.24","CMD"},"docker-compose.yml":{"personal-agent:","caddy:","pa-data:"},"Caddyfile":{"reverse_proxy personal-agent:8080"},".env.example":{"BOOTSTRAP_TOKEN=","PA_DOMAIN="},"../README.md":{"docker compose"},"../.agents/setup":{"1.24"}};for file,need:=range checks{b,err:=os.ReadFile(file);if err!=nil{t.Fatal(file,err)};for _,s:=range need{if !strings.Contains(string(b),s){t.Errorf("%s missing %q",file,s)}}}}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./deploy -run TestDeploymentFiles -v`
Expected: FAIL because deployment files do not exist.

- [ ] **Step 3: Write minimal implementation**

```dockerfile
# deploy/Dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /personal-agent ./cmd/personal-agent
FROM alpine:3.22
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=build /personal-agent /usr/local/bin/personal-agent
COPY web ./web
EXPOSE 8080
CMD ["personal-agent"]
```

```yaml
# deploy/docker-compose.yml
services:
  personal-agent:
    build: {context: .., dockerfile: deploy/Dockerfile}
    environment:
      PA_DATA_DIR: /data
      PA_ADDR: :8080
      PA_SECURE_COOKIES: ${PA_SECURE_COOKIES:-true}
      BOOTSTRAP_TOKEN: ${BOOTSTRAP_TOKEN:?set BOOTSTRAP_TOKEN}
    volumes: [pa-data:/data]
    ports: ["8080:8080"]
    restart: unless-stopped
  caddy:
    image: caddy:2-alpine
    profiles: [domain]
    environment: [PA_DOMAIN]
    ports: ["80:80", "443:443"]
    volumes: ["./Caddyfile:/etc/caddy/Caddyfile:ro", "caddy-data:/data"]
    depends_on: [personal-agent]
    restart: unless-stopped
volumes: {pa-data: {}, caddy-data: {}}
```

```caddyfile
# deploy/Caddyfile
{$PA_DOMAIN:localhost} {
    reverse_proxy personal-agent:8080
}
```

```dotenv
# deploy/.env.example
BOOTSTRAP_TOKEN=replace-with-at-least-32-random-characters
PA_DOMAIN=agent.example.com
PA_SECURE_COOKIES=true
```

```markdown
<!-- README.md -->
# Personal Agent

Self-hosted, single-owner learning dashboard. Requires Go 1.24+ or Docker.

## Development

Run `make test`, then set `BOOTSTRAP_TOKEN` and run `make run`. Open port 8080, bootstrap the owner once, and log in. Runtime data defaults to `./data`.

## Docker Compose

Copy `deploy/.env.example` to `deploy/.env`, replace the bootstrap token, then run `docker compose -f deploy/docker-compose.yml up --build`. For a real domain, set `PA_DOMAIN`, keep secure cookies enabled, and add `--profile domain`; Caddy terminates HTTPS. Model and backup credentials belong in environment variables, never the database.
```

```bash
# .agents/setup
#!/usr/bin/env bash
set -euo pipefail
required=1.24.0
current=$(go version 2>/dev/null | sed -n 's/.*go\([0-9.]*\).*/\1/p' || true)
if [ -z "$current" ] || [ "$(printf '%s\n' "$required" "$current" | sort -V | head -n1)" != "$required" ]; then
  curl -fsSLo /tmp/go.tgz https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
fi
export PATH=/usr/local/go/bin:$PATH
go mod download
```

- [ ] **Step 4: Run focused and phase verification**

Run: `go test ./deploy -v && go test ./... && docker compose -f deploy/docker-compose.yml config >/dev/null`
Expected: all Go tests PASS and Compose configuration validates. Then run `BOOTSTRAP_TOKEN=01234567890123456789012345678901 PA_SECURE_COOKIES=false timeout 10s go run ./cmd/personal-agent` and expect the log to contain `listening on :8080` before timeout.

- [ ] **Step 5: Commit**

```bash
git add deploy README.md .agents/setup
git commit -m "chore: add single-host deployment skeleton"
```

## Phase self-check

- Spec §3: single Go API, local SQLite/files data directory, static browser client, and Compose/Caddy topology are established.
- Spec §5 and §11: all entities are represented in migration 001; owner bootstrap, Argon2id, hashed sessions, secure cookie settings, and CSRF are test-covered; session scope has a shape CHECK and project-vault trigger.
- Spec §8 Home and §9 F0: writable storage, bootstrap state, empty dashboard DTO, and first-run shell are served.
- Spec §14 phase 1: `go test ./...` is green; `go run ./cmd/personal-agent` serves health, setup, auth, empty Home, and static shell; SQLite WAL migration and Compose deployment exist.
