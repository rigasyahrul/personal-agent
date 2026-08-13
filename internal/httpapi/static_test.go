package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestStaticShell(t *testing.T) {
	if _, err := os.Stat("../../web/index.html"); err != nil {
		t.Fatal(err)
	}
	h := http.FileServer(http.Dir("../../web"))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Personal Agent") || !strings.Contains(w.Body.String(), `type="module"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
