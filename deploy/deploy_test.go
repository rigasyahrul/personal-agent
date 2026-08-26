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
		"Dockerfile":             {"node:22-alpine AS web-build", "npm ci", "npm run build", "golang:1.24", "CMD", "/app/web/dist"},
		"Dockerfile.dev":         {"node:22-alpine", "golang:1.24-alpine", "air", "dev-entrypoint.sh"},
		"docker-compose.yml":     {"personal-agent:", "caddy:", "pa-data:", "OPENAI_API_KEY", "OPENAI_BASE_URL", "PA_MODELS"},
		"docker-compose.dev.yml": {"Dockerfile.dev", "..:/src", "go-mod-cache:", "PA_UI_DEV_PROXY: http://127.0.0.1:5173", "dev-entrypoint"},
		"air.toml":               {"go build", "./cmd/personal-agent", "tmp/personal-agent"},
		"Caddyfile":              {"reverse_proxy personal-agent:8080"},
		".env.example":           {"BOOTSTRAP_TOKEN=", "PA_DOMAIN=", "OPENAI_API_KEY=", "OPENAI_BASE_URL=", "PA_MODELS="},
		"../README.md":           {"docker compose", "PA_MODELS", "PA_DATA_DIR", "docker-compose.dev.yml"},
		"../.agents/setup":       {"1.24"},
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
	// Production compose must stay image-baked (no host source mounts).
	for _, forbidden := range []string{"../web:", "..:/src", "Dockerfile.dev"} {
		if strings.Contains(contents, forbidden) {
			t.Errorf("docker-compose.yml production service must not include %q (use docker-compose.dev.yml)", forbidden)
		}
	}
}

func TestComposeDevOverrideMountsFullRepo(t *testing.T) {
	dev := readFile(t, "docker-compose.dev.yml")
	for _, required := range []string{
		"Dockerfile.dev",
		"..:/src",
		"go-mod-cache:",
		"PA_DATA_DIR: /data",
		"PA_UI_DEV_PROXY: http://127.0.0.1:5173",
		"dev-entrypoint",
	} {
		if !strings.Contains(dev, required) {
			t.Errorf("docker-compose.dev.yml missing %q", required)
		}
	}
	if strings.Contains(dev, `profiles:`) {
		t.Error("dev override should be selected via -f docker-compose.dev.yml, not a compose profile on the same file")
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

func TestSetupPersistsFallbackGoPathForFutureShells(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	log := filepath.Join(home, "go.log")
	writeExecutable(t, filepath.Join(bin, "go"), "#!/bin/sh\n[ \"$1\" = version ] && echo 'go version go1.23.0 linux/amd64'\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "curl"), "#!/bin/sh\n: > /tmp/go.tgz\n")
	writeExecutable(t, filepath.Join(bin, "id"), "#!/bin/sh\n[ \"$1\" = -u ] && echo 1000\n")
	writeExecutable(t, filepath.Join(bin, "sudo"), "#!/bin/sh\nexit 1\n")
	writeExecutable(t, filepath.Join(bin, "tar"), `#!/bin/sh
while [ "$1" != "-C" ]; do shift; done
parent=$2
mkdir -p "$parent/go/bin"
cat > "$parent/go/bin/go" <<'EOF'
#!/bin/sh
if [ "$1" = version ]; then echo 'go version go1.24.0 linux/amd64'; exit; fi
if [ "$1 $2" = 'mod download' ]; then echo download >> "$SETUP_TEST_LOG"; exit; fi
exit 1
EOF
chmod +x "$parent/go/bin/go"
`)

	for range 2 {
		cmd := exec.Command("bash", "../.agents/setup")
		cmd.Env = append(os.Environ(), "HOME="+home, "PATH="+bin+":/usr/bin:/bin", "PA_SETUP_INSTALL_PARENT="+filepath.Join(home, ".local"), "SETUP_TEST_LOG="+log)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup fallback failed: %v\n%s", err, output)
		}
	}

	profile := readFile(t, filepath.Join(home, ".profile"))
	want := `export PATH="$HOME/.local/go/bin:$PATH"`
	if strings.Count(profile, want) != 1 {
		t.Fatalf("profile must contain one persistent Go PATH entry; got:\n%s", profile)
	}
	if downloads := strings.Count(readFile(t, log), "download\n"); downloads != 2 {
		t.Fatalf("installed Go must continue to go mod download on each run; got %d calls", downloads)
	}
	cmd := exec.Command("bash", "-c", `. "$HOME/.profile" && go version`)
	cmd.Env = append(os.Environ(), "HOME="+home, "PATH=/usr/bin:/bin")
	if output, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(output), "go1.24.0") {
		t.Fatalf("future shell did not select persisted Go: %v\n%s", err, output)
	}
}

func TestProductionImageCopiesOnlyBuiltWebAssets(t *testing.T) {
	dockerfile := readFile(t, "Dockerfile")
	if !strings.Contains(dockerfile, "COPY --from=web-build --chown=app:app /src/web/dist /app/web/dist") {
		t.Fatal("production image must copy Vite dist")
	}
	if strings.Contains(dockerfile, "COPY --chown=app:app web /app/web") {
		t.Fatal("production image must not copy web sources")
	}
}

func TestDockerDevHMRIsDocumentedAndProductionIsMountFree(t *testing.T) {
	docs := readFile(t, "../docs/ops/deploy.md")
	for _, required := range []string{
		"http://localhost:8080",
		"PA_UI_DEV_PROXY=http://127.0.0.1:5173",
		"ws://localhost:8080/@vite-hmr",
		"Production compose has no host source mounts",
	} {
		if !strings.Contains(docs, required) {
			t.Errorf("deploy docs missing %q", required)
		}
	}
	prod := readFile(t, "docker-compose.yml")
	for _, forbidden := range []string{"../web:", "..:/src", "PA_UI_DEV_PROXY", "5173:5173"} {
		if strings.Contains(prod, forbidden) {
			t.Errorf("production compose must not contain %q", forbidden)
		}
	}
}

func TestRootDockerignoreShrinksComposeBuildContext(t *testing.T) {
	// Compose build.context is the repo root. Docker only reads `.dockerignore`
	// next to that context — deploy/.dockerignore is unused.
	for _, file := range []string{"docker-compose.yml", "docker-compose.dev.yml"} {
		if !strings.Contains(readFile(t, file), "context: ..") {
			t.Errorf("%s must keep build context at repo root", file)
		}
	}

	ignore, err := os.ReadFile("../.dockerignore")
	if err != nil {
		t.Fatal("repo-root .dockerignore is required so --build does not ship .worktrees/node_modules into the daemon:", err)
	}
	text := string(ignore)
	for _, pattern := range []string{
		".worktrees",
		"web/node_modules",
		"web/dist",
		".git",
		"/data",
		"/tmp",
		"/personal-agent",
		"deploy/.env",
	} {
		if !strings.Contains(text, pattern) {
			t.Errorf(".dockerignore missing %q", pattern)
		}
	}
	for _, line := range []string{"deploy", "deploy/", "cmd", "cmd/", "internal", "internal/", "web", "web/", "*"} {
		for _, raw := range strings.Split(text, "\n") {
			got := strings.TrimSpace(raw)
			if i := strings.Index(got, "#"); i >= 0 {
				got = strings.TrimSpace(got[:i])
			}
			if got == line {
				t.Errorf(".dockerignore must not ignore %q (Dockerfiles COPY Go/web/deploy inputs)", line)
			}
		}
	}
	docs := readFile(t, "../docs/ops/deploy.md")
	if !strings.Contains(docs, ".dockerignore") {
		t.Error("deploy docs must explain repo-root .dockerignore; Sending build context is host→daemon, not a registry pull")
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
