## Phase 6: Backup

### Task 33: Create a mutation-safe local backup bundle

**Files:**
- Create: `internal/backup/backup.go`
- Create: `internal/backup/backup_test.go`
- Create: `internal/store/backup.go`
- Modify: `internal/domain/models.go`
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: the application `*sql.DB`, `PA_DATA_DIR`, `clock.Clock`, `ids.NewID()`, and the existing application composition root.
- Produces: `backup.Barrier`, `backup.Service.Run(context.Context) (domain.BackupRun, error)`, `backup.Service.List(context.Context) ([]domain.BackupRun, error)`, and immutable `.tar.gz` bundles containing `database.sqlite`, `files/`, and a final `manifest.json` with SHA-256 checksums.

- [ ] **Step 1: Write the failing local-backup tests**

Create `internal/backup/backup_test.go`:

```go
package backup_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/backup"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	dbopen "github.com/rigasyahrul/personal-agent/internal/db"
)

func TestRunCreatesConsistentLocalBundleAndSucceededRun(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	db, err := dbopen.Open(filepath.Join(dataDir, "personal-agent.sqlite"))
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','Known','2026-08-12T10:00:00Z','2026-08-12T10:00:00Z')`); err != nil { t.Fatal(err) }
	source := filepath.Join(dataDir, "global", "projects", "p1", "source")
	if err := os.MkdirAll(source, 0o700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(source, "known.md"), []byte("# known note\n"), 0o600); err != nil { t.Fatal(err) }

	barrier := &backup.Barrier{}
	svc := backup.NewService(db, dataDir, barrier, &clock.FakeClock{T: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)}, nil)
	run, err := svc.Run(ctx)
	if err != nil { t.Fatal(err) }
	if run.Status != "succeeded" || run.LocalPath == "" || run.ManifestHash == "" { t.Fatalf("run=%+v", run) }
	manifest, files := readBundle(t, run.LocalPath)
	if manifest.CutoffAt != "2026-08-12T10:00:00Z" { t.Fatalf("cutoff=%q", manifest.CutoffAt) }
	for name, want := range manifest.Files {
		sum := sha256.Sum256(files[name])
		if hex.EncodeToString(sum[:]) != want { t.Fatalf("checksum mismatch for %s", name) }
	}
	snapshot := filepath.Join(t.TempDir(), "snapshot.sqlite")
	if err := os.WriteFile(snapshot, files["database.sqlite"], 0o600); err != nil { t.Fatal(err) }
	restored, err := sql.Open("sqlite", snapshot)
	if err != nil { t.Fatal(err) }
	defer restored.Close()
	var name string
	if err := restored.QueryRow(`SELECT name FROM projects WHERE id='p1'`).Scan(&name); err != nil { t.Fatal(err) }
	if name != "Known" || string(files["files/global/projects/p1/source/known.md"]) != "# known note\n" { t.Fatal("bundle omitted known data") }
}

func TestBarrierBlocksMutationDuringSnapshot(t *testing.T) {
	b := &backup.Barrier{}
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() { _ = b.Snapshot(func() error { close(entered); <-release; return nil }) }()
	<-entered
	go func() { _ = b.Mutate(func() error { close(done); return nil }) }()
	select { case <-done: t.Fatal("mutation crossed snapshot barrier"); case <-time.After(30 * time.Millisecond): }
	close(release)
	select { case <-done: case <-time.After(time.Second): t.Fatal("mutation remained blocked") }
}

type testManifest struct { CutoffAt string `json:"cutoff_at"`; Files map[string]string `json:"files"` }

func readBundle(t *testing.T, path string) (testManifest, map[string][]byte) {
	t.Helper()
	f, err := os.Open(path); if err != nil { t.Fatal(err) }; defer f.Close()
	gz, err := gzip.NewReader(f); if err != nil { t.Fatal(err) }; defer gz.Close()
	tr := tar.NewReader(gz); files := map[string][]byte{}
	for { h, err := tr.Next(); if err == io.EOF { break }; if err != nil { t.Fatal(err) }; b, err := io.ReadAll(tr); if err != nil { t.Fatal(err) }; files[h.Name] = b }
	var m testManifest
	if err := json.Unmarshal(files["manifest.json"], &m); err != nil { t.Fatal(err) }
	return m, files
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/backup -run 'TestRunCreates|TestBarrier' -v`

Expected: FAIL because package `internal/backup` and its service do not exist.

- [ ] **Step 3: Implement the local snapshot, manifest, and run store**

Add this model to `internal/domain/models.go`:

```go
type BackupRun struct {
	ID, Status, CutoffAt, StartedAt string
	LocalPath, ObjectKey, ManifestHash, CompletedAt, Error string
}
```

Create `internal/store/backup.go` with `CreateBackupRun`, `CompleteBackupRun`, and `ListBackupRuns`; use the migration's `backup_runs` columns, order lists by `started_at DESC`, and translate nullable columns with `COALESCE(column,'')`:

```go
package store

import (
	"context"
	"database/sql"
	"github.com/rigasyahrul/personal-agent/internal/domain"
)

func CreateBackupRun(ctx context.Context, db *sql.DB, r domain.BackupRun) error {
	_, err := db.ExecContext(ctx, `INSERT INTO backup_runs(id,status,cutoff_at,started_at) VALUES(?,?,?,?)`, r.ID, r.Status, r.CutoffAt, r.StartedAt)
	return err
}

func CompleteBackupRun(ctx context.Context, db *sql.DB, r domain.BackupRun) error {
	_, err := db.ExecContext(ctx, `UPDATE backup_runs SET status=?,local_path=NULLIF(?,''),object_key=NULLIF(?,''),manifest_hash=NULLIF(?,''),completed_at=NULLIF(?,''),error=NULLIF(?,'') WHERE id=?`, r.Status, r.LocalPath, r.ObjectKey, r.ManifestHash, r.CompletedAt, r.Error, r.ID)
	return err
}

func ListBackupRuns(ctx context.Context, db *sql.DB) ([]domain.BackupRun, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,status,cutoff_at,COALESCE(local_path,''),COALESCE(object_key,''),COALESCE(manifest_hash,''),started_at,COALESCE(completed_at,''),COALESCE(error,'') FROM backup_runs ORDER BY started_at DESC`)
	if err != nil { return nil, err }; defer rows.Close()
	var out []domain.BackupRun
	for rows.Next() { var r domain.BackupRun; if err := rows.Scan(&r.ID,&r.Status,&r.CutoffAt,&r.LocalPath,&r.ObjectKey,&r.ManifestHash,&r.StartedAt,&r.CompletedAt,&r.Error); err != nil { return nil, err }; out = append(out,r) }
	return out, rows.Err()
}
```

Create `internal/backup/backup.go`. The SQLite driver connection exposes the online backup API through `NewBackup`; write the database snapshot before walking data files, skip the live DB/WAL/SHM and `backups/`, sort archive names, write `manifest.json` after every payload member, rename the temporary archive atomically, and record failures:

```go
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"modernc.org/sqlite"
)

type Barrier struct{ mu sync.RWMutex }
func (b *Barrier) Mutate(fn func() error) error { b.mu.RLock(); defer b.mu.RUnlock(); return fn() }
func (b *Barrier) Snapshot(fn func() error) error { b.mu.Lock(); defer b.mu.Unlock(); return fn() }

type Uploader interface { Upload(context.Context, string, string) error }
type Service struct { DB *sql.DB; DataDir string; Barrier *Barrier; Clock clock.Clock; Uploader Uploader; Bucket string }
func NewService(db *sql.DB, dataDir string, barrier *Barrier, c clock.Clock, uploader Uploader) *Service { return &Service{DB:db,DataDir:dataDir,Barrier:barrier,Clock:c,Uploader:uploader} }
func (s *Service) List(ctx context.Context) ([]domain.BackupRun,error) { return store.ListBackupRuns(ctx,s.DB) }

type manifest struct { Version int `json:"version"`; CutoffAt string `json:"cutoff_at"`; Files map[string]string `json:"files"` }

func (s *Service) Run(ctx context.Context) (run domain.BackupRun, err error) {
	now := s.Clock.Now().UTC(); run = domain.BackupRun{ID:ids.NewID(),Status:"running",CutoffAt:now.Format(time.RFC3339),StartedAt:now.Format(time.RFC3339)}
	if err = store.CreateBackupRun(ctx,s.DB,run); err != nil { return run,err }
	defer func(){ if err != nil { run.Status="failed"; run.Error=err.Error(); run.CompletedAt=s.Clock.Now().UTC().Format(time.RFC3339); _=store.CompleteBackupRun(context.Background(),s.DB,run) } }()
	err = s.Barrier.Snapshot(func() error { var e error; run.LocalPath,run.ManifestHash,e=s.local(ctx,run); return e })
	if err != nil { return run,err }
	run.Status="succeeded"; run.CompletedAt=s.Clock.Now().UTC().Format(time.RFC3339)
	if err=store.CompleteBackupRun(ctx,s.DB,run); err!=nil{return run,err}
	return run,nil
}

func (s *Service) local(ctx context.Context, run domain.BackupRun) (string,string,error) {
	dir:=filepath.Join(s.DataDir,"backups"); if err:=os.MkdirAll(dir,0o700); err!=nil{return "","",err}
	work,err:=os.MkdirTemp(dir,"snapshot-"); if err!=nil{return "","",err}; defer os.RemoveAll(work)
	dbPath:=filepath.Join(work,"database.sqlite")
	conn,err:=s.DB.Conn(ctx); if err!=nil{return "","",err}
	err=conn.Raw(func(raw any) error { c,ok:=raw.(interface{ NewBackup(string)(*sqlite.Backup,error) }); if !ok{return fmt.Errorf("sqlite connection lacks backup API")}; b,e:=c.NewBackup("file:"+dbPath); if e!=nil{return e}; defer b.Finish(); for { more,e:=b.Step(128); if e!=nil{return e}; if !more{return nil} } }); conn.Close()
	if err!=nil{return "","",err}
	entries:=map[string]string{"database.sqlite":dbPath}
	err=filepath.WalkDir(s.DataDir,func(path string,d os.DirEntry,e error) error { if e!=nil{return e}; rel,e:=filepath.Rel(s.DataDir,path); if e!=nil{return e}; if rel=="."{return nil}; first:=strings.Split(filepath.ToSlash(rel),"/")[0]; if d.IsDir() && first=="backups"{return filepath.SkipDir}; if d.IsDir(){return nil}; if first=="personal-agent.sqlite" || strings.HasSuffix(rel,"-wal") || strings.HasSuffix(rel,"-shm"){return nil}; entries["files/"+filepath.ToSlash(rel)]=path; return nil }); if err!=nil{return "","",err}
	final:=filepath.Join(dir,run.ID+".tar.gz"); tmp:=final+".tmp"; f,err:=os.OpenFile(tmp,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0o600); if err!=nil{return "","",err}
	gz:=gzip.NewWriter(f); tw:=tar.NewWriter(gz); sums:=map[string]string{}; names:=make([]string,0,len(entries)); for n:=range entries{names=append(names,n)}; sort.Strings(names)
	write:=func(name string,b []byte) error { if err:=tw.WriteHeader(&tar.Header{Name:name,Mode:0o600,Size:int64(len(b)),ModTime:time.Unix(0,0)});err!=nil{return err}; _,err:=tw.Write(b); return err }
	for _,name:=range names { b,e:=os.ReadFile(entries[name]); if e!=nil{err=e;break}; sum:=sha256.Sum256(b); sums[name]=hex.EncodeToString(sum[:]); if e=write(name,b);e!=nil{err=e;break} }
	m:=manifest{Version:1,CutoffAt:run.CutoffAt,Files:sums}; mb,_:=json.Marshal(m); if err==nil{err=write("manifest.json",mb)}; if e:=tw.Close();err==nil{err=e}; if e:=gz.Close();err==nil{err=e}; if e:=f.Close();err==nil{err=e}; if err!=nil{os.Remove(tmp);return "","",err}; if err=os.Rename(tmp,final);err!=nil{return "","",err}; mh:=sha256.Sum256(mb); return final,hex.EncodeToString(mh[:]),nil
}

var _ = io.EOF
```

In `internal/app/app.go`, construct exactly one `barrier := &backup.Barrier{}` and pass it both to `backup.NewService(...)` and every mutation entry point. Wrap database-plus-filesystem mutation bodies (project/session creation and deletion, workspace writes, publication/recovery, bite generation, and review rating) with `barrier.Mutate`; do not lock read-only paths. This ensures the exclusive snapshot waits for in-flight file operations to reach their durable operation state and prevents new ones until the manifest is complete.

- [ ] **Step 4: Run focused and regression tests**

Run: `gofmt -w internal/backup internal/store/backup.go internal/domain/models.go internal/app/app.go && go test ./internal/backup ./internal/store ./internal/app`

Expected: PASS; the bundle test proves both the online database snapshot and source file are checksummed and readable.

- [ ] **Step 5: Commit**

```bash
git add internal/backup/backup.go internal/backup/backup_test.go internal/store/backup.go internal/domain/models.go internal/app/app.go
git commit -m "feat: create mutation-safe local backups"
```

### Task 34: Upload backups to optional S3 storage

**Files:**
- Create: `internal/backup/s3.go`
- Modify: `internal/backup/backup.go`
- Modify: `internal/backup/backup_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: Task 33's local immutable bundle, `backup.Uploader`, and environment configuration.
- Produces: `backup.NewS3Uploader(client S3PutObjectAPI, bucket string) Uploader`; unset `PA_BACKUP_S3_BUCKET` requires no AWS credentials and local completion is sufficient, while a configured bucket requires successful upload before `BackupRun.status=succeeded`.

- [ ] **Step 1: Write failing optional-upload tests**

Append to `internal/backup/backup_test.go`:

```go
type mockUploader struct { calls int; key string; err error }
func (m *mockUploader) Upload(_ context.Context, path,key string) error { m.calls++; m.key=key; if _,err:=os.Stat(path);err!=nil{return err}; return m.err }

func TestRunWithoutBucketSucceedsLocallyWithoutUpload(t *testing.T) {
	svc, db := testService(t); up := &mockUploader{}; svc.Uploader=up
	run, err := svc.Run(context.Background())
	if err != nil || run.Status != "succeeded" || up.calls != 0 { t.Fatalf("run=%+v calls=%d err=%v",run,up.calls,err) }
	assertStoredStatus(t,db,run.ID,"succeeded")
}

func TestConfiguredUploadControlsFinalStatus(t *testing.T) {
	svc, db := testService(t); up := &mockUploader{}; svc.Uploader=up; svc.Bucket="archive"
	run, err := svc.Run(context.Background())
	if err != nil || run.Status != "succeeded" || up.calls != 1 || run.ObjectKey == "" { t.Fatalf("run=%+v calls=%d err=%v",run,up.calls,err) }
	assertStoredStatus(t,db,run.ID,"succeeded")

	svc, db = testService(t); svc.Bucket="archive"; svc.Uploader=&mockUploader{err:fmt.Errorf("S3 unavailable")}
	run, err = svc.Run(context.Background())
	if err == nil || run.Status != "failed" || run.LocalPath == "" { t.Fatalf("run=%+v err=%v",run,err) }
	assertStoredStatus(t,db,run.ID,"failed")
}
```

Add helpers using `dbopen.Open(filepath.Join(t.TempDir(), "personal-agent.sqlite"))`, a fake clock, a new barrier, and a query of `backup_runs.status`. Each helper must register `db.Close` with `t.Cleanup`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/backup -run 'TestRunWithoutBucket|TestConfiguredUpload' -v`

Expected: `TestRunWithoutBucketSucceedsLocallyWithoutUpload` passes and `TestConfiguredUploadControlsFinalStatus` fails because a configured upload is not yet invoked.

- [ ] **Step 3: Implement optional upload and final status transition**

Replace the tail of `Service.Run` after `s.Barrier.Snapshot(...)` with:

```go
	if err != nil { return run, err }
	if s.Bucket != "" {
		if s.Uploader == nil { err = fmt.Errorf("backup bucket configured without uploader"); return run, err }
		run.ObjectKey = "backups/" + filepath.Base(run.LocalPath)
		if err = s.Uploader.Upload(ctx, run.LocalPath, run.ObjectKey); err != nil { return run, err }
	}
	run.Status = "succeeded"
	run.CompletedAt = s.Clock.Now().UTC().Format(time.RFC3339)
	if err = store.CompleteBackupRun(ctx, s.DB, run); err != nil { return run, err }
	return run, nil
```

Create `internal/backup/s3.go` using the AWS SDK's narrow mockable interface:

```go
package backup

import (
	"context"
	"os"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3PutObjectAPI interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}
type s3Uploader struct { client S3PutObjectAPI; bucket string }
func NewS3Uploader(client S3PutObjectAPI, bucket string) Uploader { return &s3Uploader{client:client,bucket:bucket} }
func (u *s3Uploader) Upload(ctx context.Context,path,key string) error {
	f,err:=os.Open(path); if err!=nil{return err}; defer f.Close()
	_,err=u.client.PutObject(ctx,&s3.PutObjectInput{Bucket:&u.bucket,Key:&key,Body:f})
	return err
}
```

Add `BackupS3Bucket string` to config and load it from `PA_BACKUP_S3_BUCKET`. In `internal/app/app.go`, only when the bucket is non-empty, load AWS default config, construct `s3.NewFromConfig`, and assign `service.Bucket` and `service.Uploader`; with an empty bucket, do not load AWS config. Add `github.com/aws/aws-sdk-go-v2/config` and `github.com/aws/aws-sdk-go-v2/service/s3` with `go get`.

- [ ] **Step 4: Run focused and configuration tests**

Run: `gofmt -w internal/backup internal/config internal/app && go test ./internal/backup ./internal/config ./internal/app`

Expected: PASS, including local-only success, upload success, and upload-failure persistence.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/backup internal/config/config.go internal/app/app.go
git commit -m "feat: optionally upload backup bundles to S3"
```

### Task 35: Expose backup controls and status in Settings

**Files:**
- Create: `internal/httpapi/backup_handlers.go`
- Create: `internal/httpapi/backup_handlers_test.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/settings_handlers.go`
- Modify: `web/js/api.js`
- Modify: `web/js/pages/settings.js`

**Interfaces:**
- Consumes: `backup.Service.List`, `backup.Service.Run`, existing authenticated/CSRF middleware, and the existing settings DTO.
- Produces: authenticated `GET /api/v1/backups`, CSRF-protected `POST /api/v1/backups`, and settings fields `backup.last_success` and `backup.last_failure` plus a “Backup now” control.

- [ ] **Step 1: Write failing HTTP contract tests**

Create `internal/httpapi/backup_handlers_test.go` following the package's existing authenticated test-server helper:

```go
package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBackupsRequireAuthAndPostRequiresCSRF(t *testing.T) {
	s := newTestServer(t)
	for _,tc:=range []struct{method string; csrf bool; want int}{{"GET",false,http.StatusUnauthorized},{"POST",false,http.StatusUnauthorized}} {
		r:=httptest.NewRequest(tc.method,"/api/v1/backups",nil); w:=httptest.NewRecorder(); s.ServeHTTP(w,r)
		if w.Code!=tc.want { t.Fatalf("%s got %d",tc.method,w.Code) }
	}
	r:=authenticatedRequest(t,"POST","/api/v1/backups",strings.NewReader(`{}`),false); w:=httptest.NewRecorder(); s.ServeHTTP(w,r)
	if w.Code!=http.StatusForbidden { t.Fatalf("got %d",w.Code) }
}

func TestBackupNowThenListAndSettingsStatus(t *testing.T) {
	s:=newTestServer(t)
	r:=authenticatedRequest(t,"POST","/api/v1/backups",strings.NewReader(`{}`),true); w:=httptest.NewRecorder(); s.ServeHTTP(w,r)
	if w.Code!=http.StatusCreated || !strings.Contains(w.Body.String(),`"status":"succeeded"`) { t.Fatalf("%d %s",w.Code,w.Body.String()) }
	for _,path:=range []string{"/api/v1/backups","/api/v1/settings"} {
		r=authenticatedRequest(t,"GET",path,nil,false); w=httptest.NewRecorder(); s.ServeHTTP(w,r)
		if w.Code!=http.StatusOK || !strings.Contains(w.Body.String(),`"last_success"`) { t.Fatalf("%s: %d %s",path,w.Code,w.Body.String()) }
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/httpapi -run 'TestBackups|TestBackupNow' -v`

Expected: FAIL with `404` because the backup routes are not registered.

- [ ] **Step 3: Implement handlers, settings summary, and UI**

Create `internal/httpapi/backup_handlers.go`:

```go
package httpapi

import (
	"net/http"
	"github.com/rigasyahrul/personal-agent/internal/backup"
)

type backupHandlers struct{ service *backup.Service }
func (h backupHandlers) list(w http.ResponseWriter,r *http.Request) { runs,err:=h.service.List(r.Context()); if err!=nil{writeError(w,http.StatusInternalServerError,"backup_list_failed",err.Error());return}; writeJSON(w,http.StatusOK,map[string]any{"backups":runs}) }
func (h backupHandlers) create(w http.ResponseWriter,r *http.Request) { run,err:=h.service.Run(r.Context()); if err!=nil{writeJSON(w,http.StatusInternalServerError,run);return}; writeJSON(w,http.StatusCreated,run) }
```

Register `GET /api/v1/backups` behind authentication and `POST /api/v1/backups` behind authentication plus CSRF in `server.go`. Ensure the generic mutation-barrier middleware excludes this POST: `Service.Run` itself acquires the exclusive lock, so wrapping it in `Barrier.Mutate` would deadlock.

In `settings_handlers.go`, list runs and add:

```go
type backupSummary struct { LastSuccess *domain.BackupRun `json:"last_success"`; LastFailure *domain.BackupRun `json:"last_failure"` }
func summarizeBackups(runs []domain.BackupRun) backupSummary {
	var s backupSummary
	for i:=range runs { r:=runs[i]; if r.Status=="succeeded" && s.LastSuccess==nil{s.LastSuccess=&r}; if r.Status=="failed" && s.LastFailure==nil{s.LastFailure=&r} }
	return s
}
```

Expose it as the `backup` property of the settings response. In `web/js/api.js`, add `listBackups: () => request('/api/v1/backups')` and `backupNow: () => request('/api/v1/backups', {method:'POST', body:'{}'})`. In `web/js/pages/settings.js`, render “Never backed up” when `last_success` is null, otherwise “Last successful backup: <completed_at>”; render “Last attempt failed: <error>” when the newest failure is newer than the last success; and wire a disabled-while-running “Backup now” button that refreshes settings and displays the API error.

- [ ] **Step 4: Run API tests and static checks**

Run: `gofmt -w internal/httpapi && go test ./internal/httpapi && node --check web/js/api.js && node --check web/js/pages/settings.js`

Expected: PASS; unauthenticated and missing-CSRF requests remain rejected, while backup creation/list/settings status work.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/backup_handlers.go internal/httpapi/backup_handlers_test.go internal/httpapi/server.go internal/httpapi/settings_handlers.go web/js/api.js web/js/pages/settings.js
git commit -m "feat: expose backup controls and status"
```

### Task 36: Document and automate the restore drill

**Files:**
- Create: `docs/ops/backup-restore.md`
- Modify: `internal/backup/backup_test.go`

**Interfaces:**
- Consumes: Task 33's bundle format and manifest checksum contract.
- Produces: an operator stop/verify/restore/start drill and acceptance coverage proving a restored bundle opens as SQLite and contains a known indexed note and source body.

- [ ] **Step 1: Write the failing restore acceptance test**

Append a `TestRestoreDrillFindsKnownNote` test to `internal/backup/backup_test.go`. Seed project `p1`, write `global/projects/p1/source/known.md`, calculate its SHA-256, insert a ready `notes` row for that path, run a backup, extract only names listed in the manifest into a fresh restore directory using `filepath.Clean` plus a prefix check, open `database.sqlite`, query `notes` by ID, and assert the restored source bytes and metadata hash agree. Initially call a not-yet-created test helper `restoreBundle(t, run.LocalPath, restoreDir)` so the test is red.

```go
func TestRestoreDrillFindsKnownNote(t *testing.T) {
	ctx:=context.Background(); dataDir:=t.TempDir(); db,err:=dbopen.Open(filepath.Join(dataDir,"personal-agent.sqlite")); if err!=nil{t.Fatal(err)}; t.Cleanup(func(){db.Close()})
	body:=[]byte("# restore me\n"); sum:=sha256.Sum256(body); source:=filepath.Join(dataDir,"global","projects","p1","source"); if err:=os.MkdirAll(source,0o700);err!=nil{t.Fatal(err)}; if err:=os.WriteFile(filepath.Join(source,"known.md"),body,0o600);err!=nil{t.Fatal(err)}
	_,err=db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','Known','2026-08-12T10:00:00Z','2026-08-12T10:00:00Z'); INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1','p1','known.md',?,?, 'ready',1,'2026-08-12T10:00:00Z','2026-08-12T10:00:00Z')`,hex.EncodeToString(sum[:]),len(body)); if err!=nil{t.Fatal(err)}
	svc:=backup.NewService(db,dataDir,&backup.Barrier{},&clock.FakeClock{T:time.Date(2026,8,12,10,0,0,0,time.UTC)},nil); run,err:=svc.Run(ctx); if err!=nil{t.Fatal(err)}
	restoreDir:=t.TempDir(); restoreBundle(t,run.LocalPath,restoreDir)
	restored,err:=sql.Open("sqlite",filepath.Join(restoreDir,"database.sqlite")); if err!=nil{t.Fatal(err)}; defer restored.Close()
	var path,hash string; if err:=restored.QueryRow(`SELECT relative_path,content_sha256 FROM notes WHERE id='n1' AND status='ready'`).Scan(&path,&hash);err!=nil{t.Fatal(err)}
	restoredBody,err:=os.ReadFile(filepath.Join(restoreDir,"global","projects","p1","source",filepath.FromSlash(path)));if err!=nil{t.Fatal(err)}; got:=sha256.Sum256(restoredBody)
	if hash!=hex.EncodeToString(got[:]) || string(restoredBody)!=string(body){t.Fatal("restored note failed integrity check")}
}
```

- [ ] **Step 2: Run the restore test to verify it fails**

Run: `go test ./internal/backup -run TestRestoreDrillFindsKnownNote -v`

Expected: FAIL to compile with `undefined: restoreBundle`.

- [ ] **Step 3: Add the safe extraction helper and operator procedure**

Implement `restoreBundle` in the test file by reading and validating `manifest.json` first, rejecting absolute names or names whose cleaned form starts with `..`, verifying every payload checksum before writing it beneath `restoreDir`, mapping `files/<relative>` to `<restoreDir>/<relative>`, and writing `database.sqlite` at the restore root. This test-only extractor intentionally mirrors the documented manual verification without adding a v1 restore API.

Create `docs/ops/backup-restore.md` with these exact operational sections and commands:

```markdown
# Backup and restore

Backups are immutable `.tar.gz` bundles in `$PA_DATA_DIR/backups`. With no `PA_BACKUP_S3_BUCKET`, a verified local bundle is a successful backup. With a bucket configured, success means the same bundle was uploaded. RPO is the time since the last successful run; for a daily run, worst-case RPO is 24 hours. RTO depends on bundle size and operator download/verification time.

## Restore drill

1. In Settings, run **Backup now** and confirm its status is `succeeded`. Record its bundle path or S3 object key and `manifest_hash`.
2. Stop writes and the application: `docker compose -f deploy/docker-compose.yml stop personal-agent` (use the actual application service name from the Compose file if it differs).
3. Preserve the current volume: `cp -a "$PA_DATA_DIR" "${PA_DATA_DIR}.before-restore"`.
4. Download the selected S3 object when applicable. Extract the bundle into an empty temporary directory, never over the live volume: `mkdir -p /tmp/pa-restore && tar -xzf BACKUP.tar.gz -C /tmp/pa-restore`.
5. Recompute `sha256sum` for every file named by `manifest.json`; compare each result to `files[name]`. Recompute SHA-256 over the exact `manifest.json` bytes and compare it with the recorded `manifest_hash`. Abort on a missing, extra, or mismatched payload.
6. Verify the database before replacement: `sqlite3 /tmp/pa-restore/database.sqlite 'PRAGMA integrity_check;'`; the only output must be `ok`.
7. Build a fresh data directory by placing `database.sqlite` at `$PA_DATA_DIR/personal-agent.sqlite` and copying the contents under `/tmp/pa-restore/files/` to the same relative paths beneath `$PA_DATA_DIR`. Do not restore the bundle's `backups/` directory and remove stale `personal-agent.sqlite-wal` or `personal-agent.sqlite-shm` files.
8. Start the application: `docker compose -f deploy/docker-compose.yml start personal-agent`.
9. Verify `/health`, sign in, open a known note, and confirm its body renders without an integrity error. Confirm projects, review queue, and the latest durable operation states are present.
10. Record drill date, backup run ID, cutoff, manifest hash, elapsed restore time, and verification result. If any check fails, stop the app, restore `${PA_DATA_DIR}.before-restore`, and investigate before deleting either copy.

## Automated acceptance drill

`go test ./internal/backup -run TestRestoreDrillFindsKnownNote -v` creates a bundle, restores it into a fresh directory, opens the restored SQLite database, finds a known ready Note, and verifies its source body's SHA-256. Run it after every bundle-format change.
```

- [ ] **Step 4: Run restore and full backup verification**

Run: `gofmt -w internal/backup/backup_test.go && go test ./internal/backup -v && go test ./...`

Expected: PASS, including `TestRestoreDrillFindsKnownNote`; the full suite confirms the shared mutation barrier did not regress publication, review, session, or HTTP mutations.

- [ ] **Step 5: Commit**

```bash
git add docs/ops/backup-restore.md internal/backup/backup_test.go
git commit -m "docs: add verified backup restore drill"
```

### Phase self-check

- Spec §5 `BackupRun`: running/succeeded/failed lifecycle and all local/upload/manifest/error fields are persisted.
- Spec §9 F8 and §12: the shared mutation barrier drains durable file operations, SQLite uses its online backup API, files and operation state are bundled, the manifest is written last, and upload gates success only when configured.
- The application remains functional with `PA_BACKUP_S3_BUCKET` unset; local verified completion is sufficient and AWS configuration is not loaded.
- HTTP `GET/POST /api/v1/backups`, Settings never/success/failure states, authentication, and CSRF are covered.
- Spec §13 #10: the automated restore drill opens the restored database, finds a known Note, and verifies the restored source body; the operator stop/restore/start procedure is documented.
