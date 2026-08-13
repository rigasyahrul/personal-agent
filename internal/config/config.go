package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type ModelRef struct {
	Provider string
	ModelID  string
}

type Config struct {
	DataDir, Addr, BootstrapToken string
	SecureCookies                 bool
	OpenAIAPIKey                  string
	OpenAIBaseURL                 string
	Models                        []ModelRef
	// Backup S3 is optional. Empty BackupS3Bucket => local-only; do not load AWS.
	BackupS3Bucket   string
	BackupS3Region   string
	BackupS3Endpoint string // optional path-style endpoint for MinIO/R2/etc.
}

func Load() (Config, error) {
	c := Config{
		DataDir:          os.Getenv("PA_DATA_DIR"),
		Addr:             os.Getenv("PA_ADDR"),
		BootstrapToken:   os.Getenv("BOOTSTRAP_TOKEN"),
		SecureCookies:    os.Getenv("PA_SECURE_COOKIES") != "false",
		OpenAIAPIKey:     os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:    os.Getenv("OPENAI_BASE_URL"),
		BackupS3Bucket:   firstEnv("PA_BACKUP_S3_BUCKET", "PA_S3_BUCKET"),
		BackupS3Region:   firstEnv("PA_BACKUP_S3_REGION", "PA_S3_REGION"),
		BackupS3Endpoint: firstEnv("PA_BACKUP_S3_ENDPOINT", "PA_S3_ENDPOINT"),
	}
	if c.DataDir == "" {
		c.DataDir = "./data"
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if !strings.HasPrefix(c.Addr, ":") {
		return c, errors.New("PA_ADDR must begin with ':'")
	}

	models := os.Getenv("PA_MODELS")
	if models == "" {
		return c, nil
	}
	for _, value := range strings.Split(models, ",") {
		provider, modelID, ok := strings.Cut(value, ":")
		if !ok || provider == "" || modelID == "" {
			return c, fmt.Errorf("invalid PA_MODELS entry %q: must be provider:model_id", value)
		}
		c.Models = append(c.Models, ModelRef{Provider: provider, ModelID: modelID})
	}
	return c, nil
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}
