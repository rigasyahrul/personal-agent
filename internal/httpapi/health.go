package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

func healthHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		path := filepath.Join(dataDir, ".health-write")
		err := os.WriteFile(path, []byte("ok"), 0o600)
		if err == nil {
			err = os.Remove(path)
		}

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{
			"ok":               err == nil,
			"storage_writable": err == nil,
		})
	}
}
