package deploy_test

import (
	"os"
	"strings"
	"testing"
)

func TestDeploymentFiles(t *testing.T) {
	checks := map[string][]string{
		"Dockerfile":         {"golang:1.24", "CMD", "/app/web"},
		"docker-compose.yml": {"personal-agent:", "caddy:", "pa-data:", "OPENAI_API_KEY", "OPENAI_BASE_URL", "PA_MODELS"},
		"Caddyfile":          {"reverse_proxy personal-agent:8080"},
		".env.example":       {"BOOTSTRAP_TOKEN=", "PA_DOMAIN=", "OPENAI_API_KEY=", "OPENAI_BASE_URL=", "PA_MODELS="},
		"../README.md":       {"docker compose", "PA_MODELS", "PA_DATA_DIR"},
		"../.agents/setup":   {"1.24"},
	}

	for file, needed := range checks {
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Error(file, err)
			continue
		}
		for _, text := range needed {
			if !strings.Contains(string(contents), text) {
				t.Errorf("%s missing %q", file, text)
			}
		}
	}
}
