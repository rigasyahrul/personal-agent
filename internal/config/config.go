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
}

func Load() (Config, error) {
	c := Config{
		DataDir:        os.Getenv("PA_DATA_DIR"),
		Addr:           os.Getenv("PA_ADDR"),
		BootstrapToken: os.Getenv("BOOTSTRAP_TOKEN"),
		SecureCookies:  os.Getenv("PA_SECURE_COOKIES") != "false",
		OpenAIAPIKey:   os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:  os.Getenv("OPENAI_BASE_URL"),
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
