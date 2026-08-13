package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PA_DATA_DIR", "")
	t.Setenv("PA_ADDR", "")
	t.Setenv("BOOTSTRAP_TOKEN", "")
	t.Setenv("PA_SECURE_COOKIES", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("PA_MODELS", "")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DataDir != "./data" || c.Addr != ":8080" || !c.SecureCookies {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	if c.BootstrapToken != "" || c.OpenAIAPIKey != "" || c.OpenAIBaseURL != "" {
		t.Fatalf("unexpected empty-field defaults: %+v", c)
	}
	if len(c.Models) != 0 {
		t.Fatalf("Models = %#v, want empty", c.Models)
	}
}

func TestLoadEnvironment(t *testing.T) {
	t.Setenv("PA_DATA_DIR", "/var/lib/personal-agent")
	t.Setenv("PA_ADDR", ":9090")
	t.Setenv("BOOTSTRAP_TOKEN", "bootstrap-secret")
	t.Setenv("PA_SECURE_COOKIES", "false")
	t.Setenv("OPENAI_API_KEY", "openai-secret")
	t.Setenv("OPENAI_BASE_URL", "https://openai.example/v1")
	t.Setenv("PA_MODELS", "openai:gpt-4o-mini")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DataDir != "/var/lib/personal-agent" || c.Addr != ":9090" {
		t.Fatalf("unexpected paths: %+v", c)
	}
	if c.BootstrapToken != "bootstrap-secret" || c.SecureCookies {
		t.Fatalf("unexpected auth config: %+v", c)
	}
	if c.OpenAIAPIKey != "openai-secret" || c.OpenAIBaseURL != "https://openai.example/v1" {
		t.Fatalf("unexpected OpenAI config: %+v", c)
	}
	want := ModelRef{Provider: "openai", ModelID: "gpt-4o-mini"}
	if len(c.Models) != 1 || c.Models[0] != want {
		t.Fatalf("Models = %#v, want %#v", c.Models, []ModelRef{want})
	}
}

func TestLoadRejectsAddressWithoutColon(t *testing.T) {
	t.Setenv("PA_ADDR", "localhost:8080")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid PA_ADDR error")
	}
}

func TestLoadBackupS3Optional(t *testing.T) {
	t.Setenv("PA_BACKUP_S3_BUCKET", "")
	t.Setenv("PA_S3_BUCKET", "")
	t.Setenv("PA_BACKUP_S3_REGION", "")
	t.Setenv("PA_S3_REGION", "")
	t.Setenv("PA_BACKUP_S3_ENDPOINT", "")
	t.Setenv("PA_S3_ENDPOINT", "")
	t.Setenv("PA_ADDR", ":8080")
	t.Setenv("PA_MODELS", "")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.BackupS3Bucket != "" {
		t.Fatalf("bucket=%q want empty", c.BackupS3Bucket)
	}
	t.Setenv("PA_BACKUP_S3_BUCKET", "my-backups")
	t.Setenv("PA_BACKUP_S3_REGION", "auto")
	t.Setenv("PA_BACKUP_S3_ENDPOINT", "https://s3.example")
	c, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.BackupS3Bucket != "my-backups" || c.BackupS3Region != "auto" || c.BackupS3Endpoint != "https://s3.example" {
		t.Fatalf("unexpected s3 config: %+v", c)
	}
}
