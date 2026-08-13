package deploy_test

import (
	"os"
	"os/exec"
	"path/filepath"
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

func TestComposeDefaultsAreSafeForLocalHTTP(t *testing.T) {
	contents := readFile(t, "docker-compose.yml")
	if !strings.Contains(contents, `- "127.0.0.1:8080:8080"`) {
		t.Error("docker-compose.yml must publish the app only on localhost")
	}
	if !strings.Contains(contents, "PA_SECURE_COOKIES: ${PA_SECURE_COOKIES:-false}") {
		t.Error("docker-compose.yml must disable secure cookies by default for local HTTP")
	}
}

func TestCookieDocumentationSeparatesLocalAndDomainProfiles(t *testing.T) {
	env := readFile(t, ".env.example")
	if !strings.Contains(env, "PA_SECURE_COOKIES=false") {
		t.Error(".env.example must support the default local HTTP login flow")
	}
	readme := readFile(t, "../README.md")
	for _, required := range []string{"PA_SECURE_COOKIES=false", "PA_SECURE_COOKIES=true", "--profile domain"} {
		if !strings.Contains(readme, required) {
			t.Errorf("README must document %q", required)
		}
	}
}

func TestSetupAcceptsSuitableGoFromPathWithoutPrivilege(t *testing.T) {
	setup := readFile(t, "../.agents/setup")
	for _, required := range []string{"command -v go", "go version", "if command -v sudo"} {
		if !strings.Contains(setup, required) {
			t.Errorf("setup must contain %q", required)
		}
	}
	if strings.Contains(setup, "current=\"$(/usr/local/go/bin/go version") {
		t.Error("setup must not restrict version detection to /usr/local/go")
	}
}

func TestSetupDoesNotInstallWhenPathGoIsSuitable(t *testing.T) {
	bin := t.TempDir()
	writeExecutable(t, filepath.Join(bin, "go"), "#!/bin/sh\n[ \"$1\" = version ] && echo 'go version go1.24.3 linux/amd64'\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "curl"), "#!/bin/sh\necho curl must not run >&2\nexit 99\n")
	writeExecutable(t, filepath.Join(bin, "sudo"), "#!/bin/sh\necho sudo must not run >&2\nexit 99\n")

	cmd := exec.Command("bash", "../.agents/setup")
	cmd.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("setup rejected suitable Go on PATH: %v\n%s", err, output)
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
