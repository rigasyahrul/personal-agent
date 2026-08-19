package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUIProxyForwardsRequestToVite(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/src/App.svelte" || r.URL.RawQuery != "direct=1" {
			t.Fatalf("unexpected URL %s", r.URL.String())
		}
		w.Header().Set("X-Vite", "yes")
		_, _ = io.WriteString(w, "compiled svelte")
	}))
	defer upstream.Close()
	h, err := NewUIProxy(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/src/App.svelte?direct=1", nil))
	if w.Code != http.StatusOK || w.Header().Get("X-Vite") != "yes" || w.Body.String() != "compiled svelte" {
		t.Fatalf("code=%d header=%q body=%q", w.Code, w.Header().Get("X-Vite"), w.Body.String())
	}
}

func TestUIProxyRejectsInvalidURL(t *testing.T) {
	if _, err := NewUIProxy("://bad"); err == nil {
		t.Fatal("expected invalid URL error")
	}
}
