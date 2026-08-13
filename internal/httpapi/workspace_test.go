package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

func newWorkspaceFixture(t *testing.T) (http.Handler, string, string, []*http.Cookie) {
	t.Helper()
	h, _, dataDir, pid, cookies := chatAPIServer(t, nil)
	return h, dataDir, pid, cookies
}

func createWorkspaceSession(t *testing.T, h http.Handler, pid string, cookies []*http.Cookie, granted bool) domain.Session {
	t.Helper()
	res := apiRequest(t, h, http.MethodPost, "/api/v1/projects/"+pid+"/sessions", map[string]any{
		"title": "workspace", "provider": "openai", "model_id": "m",
		"tool_grants": map[string]bool{"workspace_files": granted},
	}, cookies, "csrf")
	if res.Code != http.StatusCreated {
		t.Fatalf("create session: %d %s", res.Code, res.Body.String())
	}
	var session domain.Session
	if err := json.Unmarshal(res.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	return session
}

func TestWorkspaceTreeAndFileRead(t *testing.T) {
	h, dataDir, pid, cookies := newWorkspaceFixture(t)
	session := createWorkspaceSession(t, h, pid, cookies, true)
	workspace := layout.SessionWorkspace(dataDir, session.Home, "v", pid, session.ID)
	if err := os.Mkdir(filepath.Join(workspace, "drafts"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "drafts", "note.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	tree := apiRequest(t, h, http.MethodGet, "/api/v1/sessions/"+session.ID+"/workspace/tree", nil, cookies, "wrong-csrf")
	if tree.Code != http.StatusOK {
		t.Fatalf("tree: %d %s", tree.Code, tree.Body.String())
	}
	var payload struct {
		Entries []struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
			Size int64  `json:"size"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(tree.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Entries) != 2 || payload.Entries[0].Path != "drafts" || payload.Entries[0].Kind != "directory" || payload.Entries[1].Path != "drafts/note.txt" || payload.Entries[1].Size != 5 {
		t.Fatalf("entries = %#v", payload.Entries)
	}

	file := apiRequest(t, h, http.MethodGet, "/api/v1/sessions/"+session.ID+"/workspace/file?path="+url.QueryEscape("drafts/note.txt"), nil, cookies, "")
	if file.Code != http.StatusOK || file.Body.String() != `{"content":"hello","path":"drafts/note.txt"}`+"\n" {
		t.Fatalf("file: %d %s", file.Code, file.Body.String())
	}
}

func TestWorkspaceAuthenticationGrantMissingAndEmptyTree(t *testing.T) {
	h, _, pid, cookies := newWorkspaceFixture(t)
	on := createWorkspaceSession(t, h, pid, cookies, true)
	off := createWorkspaceSession(t, h, pid, cookies, false)

	if got := apiRequest(t, h, http.MethodGet, "/api/v1/sessions/"+on.ID+"/workspace/tree", nil, nil, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous = %d", got)
	}
	if got := apiRequest(t, h, http.MethodGet, "/api/v1/sessions/"+off.ID+"/workspace/tree", nil, cookies, "").Code; got != http.StatusForbidden {
		t.Fatalf("grant off = %d", got)
	}
	if got := apiRequest(t, h, http.MethodGet, "/api/v1/sessions/missing/workspace/tree", nil, cookies, "").Code; got != http.StatusNotFound {
		t.Fatalf("missing = %d", got)
	}
	empty := apiRequest(t, h, http.MethodGet, "/api/v1/sessions/"+on.ID+"/workspace/tree", nil, cookies, "")
	if empty.Code != http.StatusOK || empty.Body.String() != `{"entries":[]}`+"\n" {
		t.Fatalf("empty tree: %d %s", empty.Code, empty.Body.String())
	}
}

func TestWorkspaceFileRejectsUnsafeContentAndPathsWithoutLeakingHostPaths(t *testing.T) {
	h, dataDir, pid, cookies := newWorkspaceFixture(t)
	session := createWorkspaceSession(t, h, pid, cookies, true)
	workspace := layout.SessionWorkspace(dataDir, session.Home, "v", pid, session.ID)
	outside := filepath.Join(dataDir, "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "link")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(workspace, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "invalid.txt"), []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "large.txt"), make([]byte, (1<<20)+1), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"../secret.txt", "link", "pipe", "invalid.txt", "large.txt", "missing.txt"} {
		res := apiRequest(t, h, http.MethodGet, "/api/v1/sessions/"+session.ID+"/workspace/file?path="+url.QueryEscape(path), nil, cookies, "")
		if res.Code != http.StatusBadRequest {
			t.Errorf("%q status = %d (%s)", path, res.Code, res.Body.String())
		}
		if strings.Contains(res.Body.String(), dataDir) || strings.Contains(res.Body.String(), outside) || strings.Contains(res.Body.String(), "secret") {
			t.Errorf("%q leaked host details: %s", path, res.Body.String())
		}
	}
}

func TestWorkspaceTreeRejectsUnsafeNodesAndHidesAtomicTemps(t *testing.T) {
	h, dataDir, pid, cookies := newWorkspaceFixture(t)
	session := createWorkspaceSession(t, h, pid, cookies, true)
	workspace := layout.SessionWorkspace(dataDir, session.Home, "v", pid, session.ID)
	if err := os.WriteFile(filepath.Join(workspace, ".pa-write-deadbeef"), []byte("temporary"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := apiRequest(t, h, http.MethodGet, "/api/v1/sessions/"+session.ID+"/workspace/tree", nil, cookies, "")
	if res.Code != http.StatusOK || strings.Contains(res.Body.String(), ".pa-write-") {
		t.Fatalf("atomic temp visible: %d %s", res.Code, res.Body.String())
	}
	if err := os.Symlink(filepath.Join(dataDir, "outside"), filepath.Join(workspace, "unsafe")); err != nil {
		t.Fatal(err)
	}
	res = apiRequest(t, h, http.MethodGet, "/api/v1/sessions/"+session.ID+"/workspace/tree", nil, cookies, "")
	if res.Code != http.StatusBadRequest || strings.Contains(res.Body.String(), dataDir) {
		t.Fatalf("unsafe tree: %d %s", res.Code, res.Body.String())
	}
}
