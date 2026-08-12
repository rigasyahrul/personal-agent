## Phase 5: Promote + review

### Task 25: Recoverable promote publication machine

**Files:**
- Create: `internal/store/promote.go`
- Create: `internal/publish/machine.go`
- Create: `internal/publish/machine_test.go`

**Interfaces:**
- Consumes: `publish.PublishInput`, `layout.SessionWorkspace`, `layout.ProjectRoot`, `paths.ValidateRelPath`, `fsroot` rooted file operations, `ids.NewID`, and the Phase 1 schema.
- Produces: `(*publish.Machine).Run(context.Context, publish.PublishInput) (string, string, error)`, durable exact transitions `accepted → frozen → path_reserved → published_fs → finalized → review_enqueued → completed`, `publish.ConflictError`, and idempotent operation lookup by request key and fingerprint.

- [ ] **Step 1: Write the failing publication tests**

```go
package publish_test

func TestPromotePublishesOnceAndRejectsChangedFingerprint(t *testing.T) {
	ctx := context.Background()
	db, dataDir, projectID, sessionID := newPublishFixture(t)
	workspace := layout.SessionWorkspace(dataDir, layout.SessionHome("project"), "", projectID, sessionID)
	require.NoError(t, os.MkdirAll(workspace, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "draft.md"), []byte("# frozen\n"), 0o644))
	m := &publish.Machine{DB: db, DataDir: dataDir, Clock: &clock.FakeClock{T: time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)}}
	in := publish.PublishInput{OpID: ids.NewID(), RequestKey: "promote-1", RequestFingerprint: "fp-1", Kind: "promote", SessionID: sessionID, WorkspacePath: "draft.md", TargetProjectID: projectID, TargetRelPath: "guides/frozen.md", ReviewMode: domain.ReviewWhole, NoteID: ids.NewID()}

	status, noteID, err := m.Run(ctx, in)
	require.NoError(t, err)
	require.Equal(t, "completed", status)
	require.Equal(t, in.NoteID, noteID)
	body, err := os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, "", projectID)), "guides/frozen.md"))
	require.NoError(t, err)
	require.Equal(t, "# frozen\n", string(body))

	status, gotNoteID, err := m.Run(ctx, in)
	require.NoError(t, err)
	require.Equal(t, "completed", status)
	require.Equal(t, noteID, gotNoteID)
	var notes, items int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM notes WHERE id=?`, noteID).Scan(&notes))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM review_items WHERE note_id=?`, noteID).Scan(&items))
	require.Equal(t, 1, notes)
	require.Equal(t, 1, items)

	changed := in
	changed.RequestFingerprint = "fp-changed"
	_, _, err = m.Run(ctx, changed)
	var conflict *publish.ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, "idempotency_key_reused", conflict.Code)
}

func TestPromoteRejectsAnotherProjectAndExistingDestination(t *testing.T) {
	ctx := context.Background()
	db, dataDir, projectID, sessionID := newPublishFixture(t)
	m := &publish.Machine{DB: db, DataDir: dataDir, Clock: clock.RealClock{}}
	otherProject := insertProject(t, db, "other")
	in := validPromoteInput(t, dataDir, projectID, sessionID)
	in.TargetProjectID = otherProject
	_, _, err := m.Run(ctx, in)
	require.ErrorContains(t, err, "session project is the only promote target")

	in = validPromoteInput(t, dataDir, projectID, sessionID)
	destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, "", projectID)), filepath.FromSlash(in.TargetRelPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))
	require.NoError(t, os.WriteFile(destination, []byte("keep me"), 0o644))
	_, _, err = m.Run(ctx, in)
	var conflict *publish.ConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, "destination_exists", conflict.Code)
	require.Equal(t, "keep me", string(mustRead(t, destination)))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/publish -run 'TestPromote(PublishesOnce|RejectsAnother)' -v`
Expected: FAIL because `Machine.Run` and `ConflictError` do not exist.

- [ ] **Step 3: Implement the minimal durable state machine**

```go
package publish

type ConflictError struct{ Code string }
func (e *ConflictError) Error() string { return e.Code }

func (m *Machine) Run(ctx context.Context, in PublishInput) (string, string, error) {
	if in.Kind != "promote" { return "", "", fmt.Errorf("unsupported publication kind %q", in.Kind) }
	workspacePath, err := paths.ValidateRelPath(in.WorkspacePath); if err != nil { return "", "", err }
	targetPath, err := paths.ValidateRelPath(in.TargetRelPath); if err != nil { return "", "", err }
	if path.Ext(targetPath) != ".md" { return "", "", fmt.Errorf("promotion requires a .md destination") }
	if in.ReviewMode != domain.ReviewNone && in.ReviewMode != domain.ReviewWhole && in.ReviewMode != domain.ReviewBites { return "", "", fmt.Errorf("invalid review mode") }

	tx, err := m.DB.BeginTx(ctx, nil); if err != nil { return "", "", err }
	defer tx.Rollback()
	var existingID, fingerprint, status, noteID string
	err = tx.QueryRowContext(ctx, `SELECT id,request_fingerprint,status,note_id FROM promote_operations WHERE request_key=?`, in.RequestKey).Scan(&existingID, &fingerprint, &status, &noteID)
	if err == nil {
		if fingerprint != in.RequestFingerprint { return "", "", &ConflictError{Code: "idempotency_key_reused"} }
		tx.Commit()
		if status == "completed" || status == "failed" { return status, noteID, nil }
		in.OpID, in.NoteID = existingID, noteID
	} else if !errors.Is(err, sql.ErrNoRows) { return "", "", err
	} else {
		var home, sessionProject, vaultID, sessionStatus string
		if err := tx.QueryRowContext(ctx, `SELECT home,project_id,coalesce(vault_id,''),status FROM sessions WHERE id=?`, in.SessionID).Scan(&home, &sessionProject, &vaultID, &sessionStatus); err != nil { return "", "", err }
		if sessionStatus != "active" { return "", "", fmt.Errorf("session is terminal") }
		if home != "project" || sessionProject != in.TargetProjectID { return "", "", fmt.Errorf("session project is the only promote target") }
		_, err = tx.ExecContext(ctx, `INSERT INTO promote_operations(id,request_key,request_fingerprint,session_id,workspace_path,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,'accepted',?,?)`, in.OpID,in.RequestKey,in.RequestFingerprint,in.SessionID,workspacePath,in.TargetProjectID,targetPath,in.ReviewMode,in.NoteID,m.Clock.Now(),m.Clock.Now())
		if err != nil { return "", "", err }
		if err := tx.Commit(); err != nil { return "", "", err }
	}

	stagingDir := filepath.Join(m.DataDir, "staging", in.OpID)
	stagingFile := filepath.Join(stagingDir, "frozen.md")
	if err := os.MkdirAll(stagingDir, 0o700); err != nil { return m.fail(ctx, in, err) }
	if statusBefore(ctx,m.DB,in.OpID) == "accepted" {
		source := layout.SessionWorkspace(m.DataDir, layout.SessionHome("project"), "", in.TargetProjectID, in.SessionID)
		body, err := readRootedRegular(source, workspacePath, paths.MaxMarkdownBytes); if err != nil { return m.fail(ctx,in,err) }
		if err := writeSync(stagingFile, body); err != nil { return m.fail(ctx,in,err) }
		hash := sha256.Sum256(body)
		if err := transition(ctx,m.DB,in.OpID,"accepted","frozen",hex.EncodeToString(hash[:]),int64(len(body))); err != nil { return "", "", err }
	}
	if statusBefore(ctx,m.DB,in.OpID) == "frozen" {
		_, err := m.DB.ExecContext(ctx, `INSERT INTO notes(id,project_id,relative_path,status,origin_session_id,origin_workspace_path,revision,created_at,updated_at) SELECT note_id,target_project_id,target_relative_path,'pending',session_id,workspace_path,0,?,? FROM promote_operations WHERE id=?`,m.Clock.Now(),m.Clock.Now(),in.OpID)
		if isUnique(err) { return m.failConflict(ctx,in,"destination_exists") }; if err != nil { return m.fail(ctx,in,err) }
		if err := setStatus(ctx,m.DB,in.OpID,"frozen","path_reserved",m.Clock.Now()); err != nil { return "", "", err }
	}
	if statusBefore(ctx,m.DB,in.OpID) == "path_reserved" {
		destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(m.DataDir,"",in.TargetProjectID)),filepath.FromSlash(targetPath))
		if err := atomicNoClobber(stagingFile,destination); err != nil { if errors.Is(err,fs.ErrExist) { return m.failConflict(ctx,in,"destination_exists") }; return m.fail(ctx,in,err) }
		if err := setStatus(ctx,m.DB,in.OpID,"path_reserved","published_fs",m.Clock.Now()); err != nil { return "", "", err }
	}
	return m.finalize(ctx,in)
}
```

In `finalize`, use one transaction to copy frozen hash/size into the pending Note, set `status='ready', revision=1`, move the operation `published_fs → finalized`, insert the whole `ReviewItem` or bite `ReviewPending` with uniqueness constraints, move `finalized → review_enqueued` whenever review mode is not `none`, and finally move to `completed`. `atomicNoClobber` must create and fsync a temporary file in the destination directory, use an OS no-replace operation, fsync that directory, and never remove an existing destination. Any pre-publication conflict marks the operation `failed` with its error while preserving existing bytes.

- [ ] **Step 4: Run the publication package tests**

Run: `go test ./internal/publish -v`
Expected: PASS; the retry has one Note/review set, changed fingerprints conflict, cross-project promotion is rejected, and existing bytes remain unchanged.

- [ ] **Step 5: Commit**

```bash
git add internal/store/promote.go internal/publish/machine.go internal/publish/machine_test.go
git commit -m "feat: add recoverable promote publication machine"
```

### Task 26: Startup recovery of non-terminal publications

**Files:**
- Create: `internal/publish/recover.go`
- Modify: `internal/publish/machine_test.go`
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: Task 25 operation rows, staging files, source files, hashes, and `Machine.Run` resumable transitions.
- Produces: `(*publish.Machine).RecoverAll(context.Context) error`, reconciliation after a crash at every non-terminal status, and startup recovery before HTTP serving.

- [ ] **Step 1: Write the failing crash-recovery test**

```go
func TestRecoverAllConvergesAfterFilesystemPublish(t *testing.T) {
	ctx := context.Background()
	db, dataDir, projectID, sessionID := newPublishFixture(t)
	in := validPromoteInput(t, dataDir, projectID, sessionID)
	insertOperationAndPendingNote(t, db, in, "published_fs", "abc", 3)
	destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir,"",projectID)),filepath.FromSlash(in.TargetRelPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(destination),0o755))
	require.NoError(t, os.WriteFile(destination,[]byte("abc"),0o644))
	m := &publish.Machine{DB:db,DataDir:dataDir,Clock:&clock.FakeClock{T:time.Date(2026,8,12,12,0,0,0,time.UTC)}}

	require.NoError(t,m.RecoverAll(ctx))
	var opStatus, noteStatus string
	require.NoError(t,db.QueryRow(`SELECT status FROM promote_operations WHERE id=?`,in.OpID).Scan(&opStatus))
	require.NoError(t,db.QueryRow(`SELECT status FROM notes WHERE id=?`,in.NoteID).Scan(&noteStatus))
	require.Equal(t,"completed",opStatus)
	require.Equal(t,"ready",noteStatus)
	require.NoError(t,m.RecoverAll(ctx))
	var count int
	require.NoError(t,db.QueryRow(`SELECT count(*) FROM review_items WHERE note_id=?`,in.NoteID).Scan(&count))
	require.Equal(t,1,count)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/publish -run TestRecoverAllConvergesAfterFilesystemPublish -v`
Expected: FAIL because `RecoverAll` is undefined.

- [ ] **Step 3: Implement recovery and startup wiring**

```go
func (m *Machine) RecoverAll(ctx context.Context) error {
	rows, err := m.DB.QueryContext(ctx, `SELECT id,request_key,request_fingerprint,session_id,workspace_path,target_project_id,target_relative_path,review_mode,note_id FROM promote_operations WHERE status NOT IN ('completed','failed') ORDER BY created_at,id`)
	if err != nil { return err }
	defer rows.Close()
	var inputs []PublishInput
	for rows.Next() {
		var in PublishInput
		if err := rows.Scan(&in.OpID,&in.RequestKey,&in.RequestFingerprint,&in.SessionID,&in.WorkspacePath,&in.TargetProjectID,&in.TargetRelPath,&in.ReviewMode,&in.NoteID); err != nil { return err }
		in.Kind = "promote"
		inputs = append(inputs,in)
	}
	if err := rows.Err(); err != nil { return err }
	for _, in := range inputs {
		if err := m.reconcilePublishedFile(ctx,in); err != nil { return fmt.Errorf("recover %s: %w",in.OpID,err) }
		if _,_,err := m.Run(ctx,in); err != nil { return fmt.Errorf("recover %s: %w",in.OpID,err) }
	}
	return nil
}

func (m *Machine) reconcilePublishedFile(ctx context.Context, in PublishInput) error {
	status := statusBefore(ctx,m.DB,in.OpID)
	if status != "published_fs" && status != "finalized" && status != "review_enqueued" { return nil }
	var want string
	if err := m.DB.QueryRowContext(ctx,`SELECT frozen_sha256 FROM promote_operations WHERE id=?`,in.OpID).Scan(&want); err != nil { return err }
	destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(m.DataDir,"",in.TargetProjectID)),filepath.FromSlash(in.TargetRelPath))
	body, err := os.ReadFile(destination); if err != nil { return err }
	got := sha256.Sum256(body)
	if hex.EncodeToString(got[:]) != want { return fmt.Errorf("published file hash mismatch") }
	return nil
}
```

In `internal/app/app.go`, construct the machine after migrations and call `machine.RecoverAll(ctx)` before constructing or serving the HTTP mux; return the wrapped recovery error so startup fails safely rather than serving inconsistent state.

- [ ] **Step 4: Run focused and application tests**

Run: `go test ./internal/publish ./internal/app -v`
Expected: PASS, including repeated recovery producing exactly one review item.

- [ ] **Step 5: Commit**

```bash
git add internal/publish/recover.go internal/publish/machine_test.go internal/app/app.go
git commit -m "feat: recover unfinished publications at startup"
```

### Task 27: Exact sm2-lite-v1 scheduler

**Files:**
- Create: `internal/review/scheduler.go`
- Create: `internal/review/scheduler_test.go`

**Interfaces:**
- Consumes: `domain.Rating` values `again | hard | good | easy` and UTC review time.
- Produces: pure `review.ApplyRating(ReviewItemState, domain.Rating, time.Time) ReviewItemState` implementing the exact locked table.

- [ ] **Step 1: Write table-driven failing tests for every branch**

```go
package review_test

func TestApplyRatingExactTable(t *testing.T) {
	now := time.Date(2026,8,12,9,0,0,0,time.UTC)
	tests := []struct{name string; in review.ReviewItemState; rating domain.Rating; stage,reps,lapses int; interval,ease float64; due time.Time}{
		{"again", review.ReviewItemState{Stage:3,IntervalDays:10,EaseFactor:1.4,Reps:8,Lapses:2},domain.RatingAgain,0,0,3,0,1.3,now.Add(10*time.Minute)},
		{"hard-new",review.ReviewItemState{EaseFactor:2.5},domain.RatingHard,1,1,0,.5,2.35,now.Add(12*time.Hour)},
		{"hard-later",review.ReviewItemState{Stage:2,IntervalDays:10,EaseFactor:1.4,Reps:2},domain.RatingHard,2,3,0,12,1.3,now.Add(12*24*time.Hour)},
		{"good-new",review.ReviewItemState{EaseFactor:2.5},domain.RatingGood,1,1,0,1,2.5,now.Add(24*time.Hour)},
		{"good-stage-one",review.ReviewItemState{Stage:1,IntervalDays:1,EaseFactor:2.5,Reps:1},domain.RatingGood,2,2,0,3,2.5,now.Add(72*time.Hour)},
		{"good-later",review.ReviewItemState{Stage:2,IntervalDays:4,EaseFactor:2.5,Reps:2},domain.RatingGood,2,3,0,10,2.5,now.Add(10*24*time.Hour)},
		{"easy-new",review.ReviewItemState{EaseFactor:2.5},domain.RatingEasy,2,1,0,4,2.65,now.Add(4*24*time.Hour)},
		{"easy-later",review.ReviewItemState{Stage:2,IntervalDays:4,EaseFactor:2.5,Reps:2},domain.RatingEasy,2,3,0,13.78,2.65,now.Add(time.Duration(13.78*24*float64(time.Hour)))},
	}
	for _,tt := range tests { t.Run(tt.name,func(t *testing.T){
		got := review.ApplyRating(tt.in,tt.rating,now)
		require.Equal(t,tt.stage,got.Stage); require.Equal(t,tt.reps,got.Reps); require.Equal(t,tt.lapses,got.Lapses)
		require.InDelta(t,tt.interval,got.IntervalDays,1e-9); require.InDelta(t,tt.ease,got.EaseFactor,1e-9); require.Equal(t,tt.due,got.DueAt)
	}) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/review -run TestApplyRatingExactTable -v`
Expected: FAIL because `ApplyRating` is undefined.

- [ ] **Step 3: Implement the pure scheduler**

```go
package review

type ReviewItemState struct { Stage int; IntervalDays, EaseFactor float64; Reps, Lapses int; DueAt time.Time }

func ApplyRating(item ReviewItemState, rating domain.Rating, now time.Time) ReviewItemState {
	switch rating {
	case domain.RatingAgain:
		item.Lapses++; item.Reps=0; item.Stage=0; item.IntervalDays=0
		item.EaseFactor=math.Max(1.3,item.EaseFactor-.2); item.DueAt=now.Add(10*time.Minute)
	case domain.RatingHard:
		item.Reps++; item.EaseFactor=math.Max(1.3,item.EaseFactor-.15)
		if item.Stage==0 { item.IntervalDays=.5 } else { item.IntervalDays*=1.2 }
		if item.Stage<1 { item.Stage=1 }; item.DueAt=addDays(now,item.IntervalDays)
	case domain.RatingGood:
		item.Reps++
		if item.Stage==0 { item.IntervalDays=1; item.Stage=1 } else if item.Stage==1 { item.IntervalDays=3; item.Stage=2 } else { item.IntervalDays*=item.EaseFactor }
		item.DueAt=addDays(now,item.IntervalDays)
	case domain.RatingEasy:
		item.Reps++; item.EaseFactor+=.15
		if item.Stage<2 { item.IntervalDays=4; item.Stage=2 } else { item.IntervalDays=item.IntervalDays*item.EaseFactor*1.3 }
		item.DueAt=addDays(now,item.IntervalDays)
	default: panic("invalid rating")
	}
	return item
}
func addDays(t time.Time, days float64) time.Time { return t.Add(time.Duration(days*24*float64(time.Hour))) }
```

- [ ] **Step 4: Run scheduler tests**

Run: `go test ./internal/review -run TestApplyRatingExactTable -v`
Expected: PASS for Again +10m, Hard, Good, Easy, and the 1.3 ease floor.

- [ ] **Step 5: Commit**

```bash
git add internal/review/scheduler.go internal/review/scheduler_test.go
git commit -m "feat: implement exact sm2-lite-v1 scheduler"
```

### Task 28: Whole-note finalization and idempotent ratings

**Files:**
- Create: `internal/store/review.go`
- Modify: `internal/publish/machine.go`
- Create: `internal/store/review_test.go`

**Interfaces:**
- Consumes: Task 27 `ApplyRating`, finalized Note metadata, request key, expected `row_version`, and `clock.Clock`.
- Produces: immediately due whole-note items (`stage=0`, `interval_days=0`, `ease_factor=2.5`, `reps=0`, `lapses=0`), `store.ReviewStore.Rate`, one append-only `ReviewEvent`, idempotent retries, and optimistic concurrency conflicts.

- [ ] **Step 1: Write failing store tests**

```go
func TestRateIsAtomicIdempotentAndVersioned(t *testing.T) {
	db,itemID := reviewStoreFixture(t)
	s := store.ReviewStore{DB:db,Clock:&clock.FakeClock{T:time.Date(2026,8,12,9,0,0,0,time.UTC)}}
	got,err := s.Rate(context.Background(),itemID,"rate-1",0,domain.RatingGood,1250)
	require.NoError(t,err); require.Equal(t,int64(1),got.RowVersion); require.Equal(t,3.0,got.IntervalDays)

	again,err := s.Rate(context.Background(),itemID,"rate-1",0,domain.RatingGood,1250)
	require.NoError(t,err); require.Equal(t,got,again)
	var events int; require.NoError(t,db.QueryRow(`SELECT count(*) FROM review_events WHERE request_key='rate-1'`).Scan(&events)); require.Equal(t,1,events)

	_,err=s.Rate(context.Background(),itemID,"rate-2",0,domain.RatingEasy,20)
	var conflict *store.RowVersionConflict; require.ErrorAs(t,err,&conflict)
}

func TestWholeReviewItemStartsImmediatelyDue(t *testing.T) {
	db,dataDir,projectID,sessionID:=newPublishFixture(t); now:=time.Date(2026,8,12,9,0,0,0,time.UTC)
	in:=validPromoteInput(t,dataDir,projectID,sessionID); in.ReviewMode=domain.ReviewWhole
	_,_,err:=(&publish.Machine{DB:db,DataDir:dataDir,Clock:&clock.FakeClock{T:now}}).Run(context.Background(),in); require.NoError(t,err)
	var stage,reps,lapses int; var interval,ease float64; var due time.Time
	require.NoError(t,db.QueryRow(`SELECT stage,interval_days,ease_factor,reps,lapses,due_at FROM review_items WHERE note_id=?`,in.NoteID).Scan(&stage,&interval,&ease,&reps,&lapses,&due))
	require.Equal(t,[]any{0,0.0,2.5,0,0,now},[]any{stage,interval,ease,reps,lapses,due})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/store ./internal/publish -run 'Test(RateIsAtomic|WholeReview)' -v`
Expected: FAIL because `ReviewStore.Rate` and atomic whole-item finalization are absent.

- [ ] **Step 3: Implement atomic rating and whole-item insertion**

```go
func (s ReviewStore) Rate(ctx context.Context,itemID,key string,version int64,rating domain.Rating,durationMS int64)(RatedItem,error){
	tx,err:=s.DB.BeginTx(ctx,nil); if err!=nil{return RatedItem{},err}; defer tx.Rollback()
	if prior,ok,err:=eventResult(ctx,tx,key); err!=nil{return RatedItem{},err}else if ok{return prior,nil}
	var current RatedItem
	err=tx.QueryRowContext(ctx,`SELECT stage,interval_days,ease_factor,reps,lapses,due_at,row_version FROM review_items WHERE id=? AND status='active'`,itemID).Scan(&current.Stage,&current.IntervalDays,&current.EaseFactor,&current.Reps,&current.Lapses,&current.DueAt,&current.RowVersion)
	if err!=nil{return RatedItem{},err}; if current.RowVersion!=version{return RatedItem{},&RowVersionConflict{Current:current.RowVersion}}
	next:=review.ApplyRating(current.State(),rating,s.Clock.Now())
	previousJSON,_:=json.Marshal(current); resulting:=current.WithState(next); resulting.RowVersion++; resulting.LastReviewedAt=s.Clock.Now(); resultingJSON,_:=json.Marshal(resulting)
	res,err:=tx.ExecContext(ctx,`UPDATE review_items SET stage=?,interval_days=?,ease_factor=?,reps=?,lapses=?,due_at=?,last_reviewed_at=?,row_version=row_version+1 WHERE id=? AND row_version=?`,next.Stage,next.IntervalDays,next.EaseFactor,next.Reps,next.Lapses,next.DueAt,s.Clock.Now(),itemID,version)
	if err!=nil{return RatedItem{},err}; if n,_:=res.RowsAffected();n!=1{return RatedItem{},&RowVersionConflict{Current:version+1}}
	_,err=tx.ExecContext(ctx,`INSERT INTO review_events(id,review_item_id,request_key,rating,previous_state_json,resulting_state_json,scheduler_version,reviewed_at,duration_ms) VALUES(?,?,?,?,?,?,?,?,?)`,ids.NewID(),itemID,key,rating,previousJSON,resultingJSON,"sm2-lite-v1",s.Clock.Now(),durationMS)
	if err!=nil{return RatedItem{},err}; if err:=tx.Commit();err!=nil{return RatedItem{},err}; return resulting,nil
}
```

In the Task 25 finalize transaction, insert a `whole` item with `source_sha256`, `source_revision=1`, prompt `Review this note`, `due_at=now`, `status='active'`, `row_version=0`, and `scheduler_version='sm2-lite-v1'`; use a uniqueness-preserving insert so recovery cannot duplicate it.

- [ ] **Step 4: Run store and publication tests**

Run: `go test ./internal/store ./internal/publish -v`
Expected: PASS; same-key rating retries return the first result with one event, stale versions conflict, and whole items are immediately due.

- [ ] **Step 5: Commit**

```bash
git add internal/store/review.go internal/store/review_test.go internal/publish/machine.go
git commit -m "feat: finalize whole reviews and rate idempotently"
```

### Task 29: Bite generation lease worker and retry

**Files:**
- Create: `internal/review/bites.go`
- Create: `internal/review/bites_test.go`
- Modify: `internal/publish/machine.go`

**Interfaces:**
- Consumes: `agent.Provider`, finalized note bytes, `ReviewPending` rows, generator version `bites-v1`, and schema `{ "bites": [{"prompt": string, "answer": string}] }` limited to 8.
- Produces: `review.BiteWorker.LeaseAndRun(context.Context) (bool, error)`, transactional bite item creation, expired-lease recovery, failed retry state, and no effect on ready Notes.

- [ ] **Step 1: Write failing worker tests with a fake provider**

```go
type fakeProvider struct{ response string; err error }
func(f fakeProvider)Chat(context.Context,agent.ChatRequest)(agent.ChatResponse,error){return agent.ChatResponse{Content:f.response},f.err}

func TestBiteWorkerFailureKeepsNoteAndRetryCreatesItemsOnce(t *testing.T){
	db,dataDir,pendingID,noteID:=biteFixture(t)
	now:=time.Date(2026,8,12,9,0,0,0,time.UTC); c:=&clock.FakeClock{T:now}
	w:=review.BiteWorker{DB:db,DataDir:dataDir,Clock:c,Provider:fakeProvider{err:errors.New("provider down")},Lease:time.Minute}
	didWork,err:=w.LeaseAndRun(context.Background()); require.True(t,didWork); require.ErrorContains(t,err,"provider down")
	var noteStatus,pendingStatus string; require.NoError(t,db.QueryRow(`SELECT status FROM notes WHERE id=?`,noteID).Scan(&noteStatus)); require.Equal(t,"ready",noteStatus)
	require.NoError(t,db.QueryRow(`SELECT status FROM review_pending WHERE id=?`,pendingID).Scan(&pendingStatus)); require.Equal(t,"failed",pendingStatus)

	require.NoError(t,store.RetryReviewPending(context.Background(),db,pendingID))
	w.Provider=fakeProvider{response:`{"bites":[{"prompt":"What is A?","answer":"B"},{"prompt":"What is C?","answer":"D"}]}`}
	didWork,err=w.LeaseAndRun(context.Background()); require.True(t,didWork); require.NoError(t,err)
	var count int; require.NoError(t,db.QueryRow(`SELECT count(*) FROM review_items WHERE generation_id=?`,pendingID).Scan(&count)); require.Equal(t,2,count)
	didWork,err=w.LeaseAndRun(context.Background()); require.False(t,didWork); require.NoError(t,err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/review -run TestBiteWorkerFailureKeepsNoteAndRetryCreatesItemsOnce -v`
Expected: FAIL because `BiteWorker` does not exist.

- [ ] **Step 3: Implement leasing, strict decoding, and transactional insertion**

```go
type BiteWorker struct{DB *sql.DB; DataDir string; Clock clock.Clock; Provider agent.Provider; Lease time.Duration}
type biteOutput struct{Bites []struct{Prompt string `json:"prompt"`; Answer string `json:"answer"`} `json:"bites"`}

func(w *BiteWorker)LeaseAndRun(ctx context.Context)(bool,error){
	tx,err:=w.DB.BeginTx(ctx,nil);if err!=nil{return false,err};defer tx.Rollback()
	row:=tx.QueryRowContext(ctx,`SELECT id,note_id,source_sha256 FROM review_pending WHERE status='pending' OR (status='leased' AND lease_until<=?) ORDER BY id LIMIT 1`,w.Clock.Now())
	var jobID,noteID,sourceHash string;if err:=row.Scan(&jobID,&noteID,&sourceHash);errors.Is(err,sql.ErrNoRows){return false,nil}else if err!=nil{return false,err}
	res,err:=tx.ExecContext(ctx,`UPDATE review_pending SET status='leased',attempts=attempts+1,lease_until=? WHERE id=? AND (status='pending' OR lease_until<=?)`,w.Clock.Now().Add(w.Lease),jobID,w.Clock.Now());if err!=nil{return false,err};if n,_:=res.RowsAffected();n!=1{return false,nil};if err:=tx.Commit();err!=nil{return false,err}
	body,projectID,revision,err:=w.readReadyNote(ctx,noteID,sourceHash);if err!=nil{return true,w.fail(ctx,jobID,err)}
	resp,err:=w.Provider.Chat(ctx,agent.ChatRequest{Messages:[]agent.ChatMessage{{Role:"system",Content:"Return JSON only: {\"bites\":[{\"prompt\":string,\"answer\":string}]}, with 1 to 8 non-empty bites."},{Role:"user",Content:string(body)}}});if err!=nil{return true,w.fail(ctx,jobID,err)}
	var out biteOutput;dec:=json.NewDecoder(strings.NewReader(resp.Content));dec.DisallowUnknownFields();if err:=dec.Decode(&out);err!=nil{return true,w.fail(ctx,jobID,err)}
	if len(out.Bites)<1||len(out.Bites)>8{return true,w.fail(ctx,jobID,fmt.Errorf("generator returned %d bites",len(out.Bites)))}
	for _,b:=range out.Bites{if strings.TrimSpace(b.Prompt)==""||strings.TrimSpace(b.Answer)==""{return true,w.fail(ctx,jobID,fmt.Errorf("bite prompt and answer must be non-empty"))}}
	return true,w.complete(ctx,jobID,noteID,projectID,sourceHash,revision,out)
}
```

`complete` must use one transaction, verify the row is still `leased`, insert each bite with `(generation_id, ordinal)` uniqueness, initial whole-note scheduling defaults, and then set `ReviewPending.status='completed'`. `fail` sets only the pending row to `failed`, clears `lease_until`, records `last_error`, and never updates or deletes the Note. Finalization inserts one active `ReviewPending` for `(note_id, source_sha256, 'bites-v1')` and moves the operation through `review_enqueued` to `completed`.

- [ ] **Step 4: Run bite and publication tests**

Run: `go test ./internal/review ./internal/publish -v`
Expected: PASS; provider failure leaves the Note ready, retry creates exactly two items, and no duplicate job is processed.

- [ ] **Step 5: Commit**

```bash
git add internal/review/bites.go internal/review/bites_test.go internal/publish/machine.go
git commit -m "feat: generate retryable review bites"
```

### Task 30: Explicitly scoped review queue API

**Files:**
- Create: `internal/review/queue.go`
- Create: `internal/httpapi/review_handlers.go`
- Create: `internal/httpapi/review_handlers_test.go`
- Modify: `internal/httpapi/server.go`

**Interfaces:**
- Consumes: active due review items, owner time, Task 28 rating store, Task 29 retry, auth/CSRF middleware, and query scope `all | project:{id}`.
- Produces: `GET /api/v1/review/queue`, rate/suspend/retry mutations, explicit scope echo, and `caught_up` computed only in the selected scope.

- [ ] **Step 1: Write failing HTTP scope and mutation tests**

```go
func TestReviewQueueNeverWidensProjectScope(t *testing.T){
	srv,csrf,p1,p2,item1,item2:=reviewHTTPFixture(t);_ = item2
	r:=authedRequest(t,http.MethodGet,"/api/v1/review/queue?scope=project:"+p1,nil,csrf);w:=httptest.NewRecorder();srv.ServeHTTP(w,r)
	require.Equal(t,http.StatusOK,w.Code)
	var got struct{Scope string `json:"scope"`;CaughtUp bool `json:"caught_up"`;Items []struct{ID string `json:"id"`} `json:"items"`}
	require.NoError(t,json.NewDecoder(w.Body).Decode(&got));require.Equal(t,"project:"+p1,got.Scope);require.False(t,got.CaughtUp);require.Equal(t,item1,got.Items[0].ID);require.Len(t,got.Items,1)

	r=authedRequest(t,http.MethodGet,"/api/v1/review/queue?scope=project:"+p2,nil,csrf);w=httptest.NewRecorder();srv.ServeHTTP(w,r)
	require.Equal(t,http.StatusOK,w.Code);require.NoError(t,json.NewDecoder(w.Body).Decode(&got));require.Equal(t,"project:"+p2,got.Scope)
}

func TestReviewRateRetryAndSuspend(t *testing.T){
	srv,csrf,p1,_,item,_:=reviewHTTPFixture(t)
	body:=strings.NewReader(`{"rating":"good","request_key":"rating-1","row_version":0,"duration_ms":50}`)
	r:=authedRequest(t,http.MethodPost,"/api/v1/review/items/"+item+"/rate",body,csrf);w:=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusOK,w.Code)
	body=strings.NewReader(`{"rating":"good","request_key":"rating-1","row_version":0,"duration_ms":50}`)
	r=authedRequest(t,http.MethodPost,"/api/v1/review/items/"+item+"/rate",body,csrf);w=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusOK,w.Code)
	r=authedRequest(t,http.MethodPost,"/api/v1/review/items/"+item+"/suspend",strings.NewReader(`{}`),csrf);w=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusNoContent,w.Code)
	r=authedRequest(t,http.MethodGet,"/api/v1/review/queue?scope=project:"+p1,nil,csrf);w=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Contains(t,w.Body.String(),`"caught_up":true`)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/httpapi -run 'TestReview(Queue|Rate)' -v`
Expected: FAIL with 404 because review routes are not registered.

- [ ] **Step 3: Implement exact scope parsing and handlers**

```go
func ParseScope(raw string)(Scope,error){
	if raw=="all"{return Scope{Raw:"all"},nil}
	if strings.HasPrefix(raw,"project:")&&len(strings.TrimPrefix(raw,"project:"))>0{return Scope{Raw:raw,ProjectID:strings.TrimPrefix(raw,"project:")},nil}
	return Scope{},fmt.Errorf("scope must be all or project:{id}")
}
func(q Queue)Due(ctx context.Context,scope Scope)(QueueDTO,error){
	query:=`SELECT id,project_id,note_id,kind,prompt,answer,row_version,due_at FROM review_items WHERE status='active' AND due_at<=?`
	args:=[]any{q.Clock.Now()};if scope.ProjectID!=""{query+=` AND project_id=?`;args=append(args,scope.ProjectID)};query+=` ORDER BY due_at,id LIMIT 50`
	items,err:=scanQueue(ctx,q.DB,query,args...);if err!=nil{return QueueDTO{},err}
	return QueueDTO{Scope:scope.Raw,CaughtUp:len(items)==0,Items:items},nil
}
```

Register all four locked routes. Reject missing/invalid scope with 400 rather than defaulting. Rate validates the four exact ratings and maps `RowVersionConflict` to 409. Suspend updates only the named active item to `suspended` and is idempotent. Pending retry changes `failed → pending`, clears lease/error, and returns 409 for a row not in `failed`; all three mutations remain behind auth and CSRF.

- [ ] **Step 4: Run HTTP API tests**

Run: `go test ./internal/httpapi -run 'TestReview(Queue|Rate)' -v`
Expected: PASS; project responses contain no other-project item and caught-up changes after suspension.

- [ ] **Step 5: Commit**

```bash
git add internal/review/queue.go internal/httpapi/review_handlers.go internal/httpapi/review_handlers_test.go internal/httpapi/server.go
git commit -m "feat: expose explicitly scoped review queue"
```

### Task 31: Promote and operation-status HTTP APIs

**Files:**
- Create: `internal/httpapi/promote_handlers.go`
- Create: `internal/httpapi/promote_handlers_test.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/store/promote.go`

**Interfaces:**
- Consumes: Task 25 `Machine.Run`, session/project lookup, request `Idempotency-Key`, immutable canonical request fingerprint, and durable operation rows.
- Produces: `POST /api/v1/sessions/{id}/promote`, `GET /api/v1/operations/{id}`, 409 conflict responses, and badge DTO fields for promotion and card-generation retry states.

- [ ] **Step 1: Write failing endpoint tests**

```go
func TestPromoteEndpointIsIdempotentAndReportsBadges(t *testing.T){
	srv,csrf,sessionID:=promoteHTTPFixture(t)
	payload:=`{"workspace_path":"draft.md","target_relative_path":"saved/draft.md","review_mode":"bites"}`
	request:=func(key string) *httptest.ResponseRecorder { r:=authedRequest(t,http.MethodPost,"/api/v1/sessions/"+sessionID+"/promote",strings.NewReader(payload),csrf);r.Header.Set("Idempotency-Key",key);w:=httptest.NewRecorder();srv.ServeHTTP(w,r);return w }
	w1:=request("promote-http-1");require.Equal(t,http.StatusAccepted,w1.Code)
	w2:=request("promote-http-1");require.Equal(t,http.StatusAccepted,w2.Code);require.JSONEq(t,w1.Body.String(),w2.Body.String())
	var accepted struct{OperationID string `json:"operation_id"`};require.NoError(t,json.Unmarshal(w1.Body.Bytes(),&accepted))
	r:=authedRequest(t,http.MethodGet,"/api/v1/operations/"+accepted.OperationID,nil,csrf);w:=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusOK,w.Code)
	require.Contains(t,w.Body.String(),`"publication_status":"completed"`);require.Contains(t,w.Body.String(),`"badge":"Note saved; cards pending…"`);require.Contains(t,w.Body.String(),`"retry_cards":false`)
}

func TestPromoteEndpointMapsConflictsTo409(t *testing.T){
	srv,csrf,sessionID:=promoteHTTPFixture(t)
	r:=authedRequest(t,http.MethodPost,"/api/v1/sessions/"+sessionID+"/promote",strings.NewReader(`{"workspace_path":"draft.md","target_relative_path":"a.md","review_mode":"none"}`),csrf);r.Header.Set("Idempotency-Key","same");w:=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusAccepted,w.Code)
	r=authedRequest(t,http.MethodPost,"/api/v1/sessions/"+sessionID+"/promote",strings.NewReader(`{"workspace_path":"draft.md","target_relative_path":"b.md","review_mode":"none"}`),csrf);r.Header.Set("Idempotency-Key","same");w=httptest.NewRecorder();srv.ServeHTTP(w,r);require.Equal(t,http.StatusConflict,w.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/httpapi -run TestPromoteEndpoint -v`
Expected: FAIL with 404 because promote routes do not exist.

- [ ] **Step 3: Implement promote submission, fingerprinting, and status DTOs**

```go
type promoteRequest struct{WorkspacePath string `json:"workspace_path"`;TargetRelativePath string `json:"target_relative_path"`;ReviewMode domain.ReviewMode `json:"review_mode"`}
func(h PromoteHandler)Create(w http.ResponseWriter,r *http.Request){
	key:=strings.TrimSpace(r.Header.Get("Idempotency-Key"));if key==""{writeError(w,400,"idempotency_key_required");return}
	var req promoteRequest;if err:=decodeStrictJSON(r,&req);err!=nil{writeError(w,400,"invalid_request");return}
	projectID,err:=h.Store.SessionProject(r.Context(),r.PathValue("id"));if err!=nil{writeStoreError(w,err);return}
	canonical,_:=json.Marshal(struct{SessionID,WorkspacePath,TargetProjectID,TargetRelativePath string;ReviewMode domain.ReviewMode}{r.PathValue("id"),req.WorkspacePath,projectID,req.TargetRelativePath,req.ReviewMode})
	fingerprint:=sha256.Sum256(canonical)
	in:=publish.PublishInput{OpID:ids.NewID(),RequestKey:key,RequestFingerprint:hex.EncodeToString(fingerprint[:]),Kind:"promote",SessionID:r.PathValue("id"),WorkspacePath:req.WorkspacePath,TargetProjectID:projectID,TargetRelPath:req.TargetRelativePath,ReviewMode:req.ReviewMode,NoteID:ids.NewID()}
	status,noteID,err:=h.Machine.Run(r.Context(),in);if err!=nil{if _,ok:=err.(*publish.ConflictError);ok{writeError(w,409,err.Error());return};writeError(w,422,err.Error());return}
	writeJSON(w,202,map[string]any{"operation_id":in.OpID,"note_id":noteID,"status":status})
}
```

When an existing idempotency key is returned, expose its stored operation ID rather than the newly allocated ID. The status handler returns exact `publication_status`, Note status, pending ID/status, and derives only these copies: non-terminal publication=`Promoting…`; failed publication=`Promote failed — Retry`; ready note plus pending/leased cards=`Note saved; cards pending…`; ready note plus failed cards=`Cards failed — Retry cards`; otherwise ready=`Ready`. Set `retry_cards=true` only for failed `ReviewPending`; do not retry publication with a new destination or silently overwrite.

- [ ] **Step 4: Run HTTP tests**

Run: `go test ./internal/httpapi -run TestPromoteEndpoint -v`
Expected: PASS for same-key replay, changed-fingerprint 409, operation status, and card badge data.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/promote_handlers.go internal/httpapi/promote_handlers_test.go internal/httpapi/server.go internal/store/promote.go
git commit -m "feat: expose promote operation APIs"
```

### Task 32: Save-to-source and review web UI

**Files:**
- Modify: `web/js/api.js`
- Modify: `web/js/pages/sessions.js`
- Create: `web/js/pages/review.js`
- Create: `web/js/components/status-badges.js`
- Modify: `web/js/router.js`
- Modify: `web/css/app.css`
- Test: `internal/httpapi/web_test.go`

**Interfaces:**
- Consumes: Tasks 30–31 review/promote APIs, session project ID, selected workspace `.md`, operation badge DTOs, and URL query scope.
- Produces: Save to source modal, project-only target, review cards for whole/bite items, explicit scope chip, rating/suspend actions, caught-up UI, and durable operation/card badges with retry-cards action.

- [ ] **Step 1: Write a failing embedded-web contract test**

```go
func TestWebContainsPromoteAndReviewContracts(t *testing.T){
	tests:=map[string][]string{
		"../../../web/js/pages/sessions.js":{"Save to source","target_relative_path","review_mode","operation_id"},
		"../../../web/js/pages/review.js":{"project:","scope=","caught_up","row_version","duration_ms"},
		"../../../web/js/components/status-badges.js":{"Promoting…","Promote failed — Retry","Note saved; cards pending…","Cards failed — Retry cards","Ready"},
	}
	for file,wants:=range tests{t.Run(filepath.Base(file),func(t *testing.T){body,err:=os.ReadFile(file);require.NoError(t,err);for _,want:=range wants{require.Contains(t,string(body),want)}})}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpapi -run TestWebContainsPromoteAndReviewContracts -v`
Expected: FAIL because the review page and badge component do not exist.

- [ ] **Step 3: Implement the modal, cards, scope state, and badges**

```js
// web/js/components/status-badges.js
export function operationBadge(operation, onRetryCards) {
  const el = document.createElement('div'); el.className = `status-badge status-${operation.publication_status}`;
  el.textContent = operation.badge || 'Ready';
  if (operation.retry_cards) { const button=document.createElement('button'); button.textContent='Retry cards'; button.onclick=onRetryCards; el.append(' ',button); }
  return el;
}

// web/js/pages/review.js
export async function renderReview(root,{projectId}) {
  const params=new URLSearchParams(location.search); const fallback=projectId?`project:${projectId}`:'all'; const scope=params.get('scope')||fallback;
  if(scope!=='all'&&!scope.startsWith('project:')) throw new Error('Invalid review scope');
  const data=await api.get(`/api/v1/review/queue?scope=${encodeURIComponent(scope)}`);
  const chip=document.createElement('nav'); chip.className='scope-chip'; chip.innerHTML=`<button data-scope="project:${projectId}">This project</button><button data-scope="all">All projects</button>`;
  chip.onclick=e=>{const next=e.target.dataset.scope;if(!next)return;params.set('scope',next);history.pushState({},'',`${location.pathname}?${params}`);renderReview(root,{projectId});};
  root.replaceChildren(chip);
  if(data.caught_up){const empty=document.createElement('p');empty.className='caught-up';empty.textContent=`Caught up in ${data.scope==='all'?'all projects':'this project'}.`;root.append(empty);return;}
  for(const item of data.items){const started=performance.now();const card=document.createElement('article');card.className='review-card';card.innerHTML=`<h2>${escapeHTML(item.prompt)}</h2>${item.kind==='whole'?'<button class="open-note">Open current note</button>':`<button class="reveal">Reveal answer</button><p class="answer" hidden>${escapeHTML(item.answer)}</p>`}<div class="ratings"></div>`;
    card.querySelector('.reveal')?.addEventListener('click',()=>card.querySelector('.answer').hidden=false);
    for(const rating of ['again','hard','good','easy']){const b=document.createElement('button');b.textContent=rating;b.onclick=async()=>{await api.post(`/api/v1/review/items/${item.id}/rate`,{rating,request_key:crypto.randomUUID(),row_version:item.row_version,duration_ms:Math.round(performance.now()-started)});renderReview(root,{projectId});};card.querySelector('.ratings').append(b);}
    const suspend=document.createElement('button');suspend.textContent='Suspend';suspend.onclick=async()=>{await api.post(`/api/v1/review/items/${item.id}/suspend`,{});renderReview(root,{projectId});};card.append(suspend);root.append(card);}
}
```

In `sessions.js`, show **Save to source** only for a selected regular `.md` workspace file. The modal collects `target_relative_path` and radio values `none | whole | bites`; it displays the immutable session project name without a target-project picker, posts with a fresh `Idempotency-Key`, stores `operation_id`, polls `/api/v1/operations/{id}` across navigation/reload, renders `operationBadge`, and invokes `/api/v1/review/pending/{id}/retry` only when `retry_cards` is true. In `router.js`, project Review links set `?scope=project:{projectId}` and Home review sets `?scope=all`. Add focused modal/card/scope/badge styles without introducing a bundler.

- [ ] **Step 4: Run web contract and full tests**

Run: `go test ./internal/httpapi -run TestWebContainsPromoteAndReviewContracts -v && go test ./...`
Expected: PASS; static contracts include explicit scope, optimistic rating fields, modal payload fields, and all exact badge copy.

- [ ] **Step 5: Commit**

```bash
git add web/js/api.js web/js/pages/sessions.js web/js/pages/review.js web/js/components/status-badges.js web/js/router.js web/css/app.css internal/httpapi/web_test.go
git commit -m "feat: add promote and review web flows"
```

## Phase self-check

- Publication §6 and F4: Tasks 25–26 and 31–32 cover freezing, staging, reservation, no-clobber publish, exact durable statuses, finalization, idempotency/fingerprints, 409 conflicts, project-home target restriction, recovery, operation polling, and badges.
- Review §7 and F7: Tasks 27–30 and 32 cover the exact `sm2-lite-v1` table, immediately due whole items, bite snapshots, explicit `project:{id} | all` scope, rating transactions/events/versioning, suspension, and caught-up behavior.
- Data model §5: Tasks 25, 28, and 29 use PromoteOperation, Note, ReviewPending, ReviewItem, and ReviewEvent contracts with exact locked statuses and versions.
- Failure/status UX §9: Tasks 29, 31, and 32 preserve ready Notes after generation failure and expose retry-card controls plus all exact badge text.
- Acceptance §13: Task 25 proves same-key uniqueness and destination no-overwrite (1, 6); Task 26 proves post-publish convergence (2); Task 29 proves bite failure preserves the Note (3); Task 25 enforces project session scope (4); Phase 4 supplies rooted path/symlink enforcement used here; Task 28 proves one event on rating retry (8).
