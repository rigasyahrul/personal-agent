package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
)

var healthProbeReady = func() {}

func healthHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		file, err := os.CreateTemp(dataDir, ".health-write-*")
		if err == nil {
			path := file.Name()
			defer os.Remove(path)

			_, err = file.Write([]byte("ok"))
			closeErr := file.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				healthProbeReady()
				err = os.Remove(path)
			}
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
