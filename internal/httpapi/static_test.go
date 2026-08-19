package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestStaticShell(t *testing.T) {
	if _, err := os.Stat("../../web/dist/index.html"); err != nil {
		t.Fatal("run npm --prefix web run build first:", err)
	}
	h := http.FileServer(http.Dir("../../web/dist"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Personal Agent") || !strings.Contains(w.Body.String(), `type="module"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
