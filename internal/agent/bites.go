package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/review"
)

type ProviderBiteGenerator struct{ Provider Provider }

const biteSystemPrompt = `Return JSON only using exactly this schema: {"bites":[{"prompt":string,"answer":string}]}. Return 1 to 8 bites. Every prompt and answer must be non-empty. Do not add fields or prose.`

func (g ProviderBiteGenerator) Generate(ctx context.Context, noteBody string) ([]review.Bite, error) {
	if g.Provider == nil {
		return nil, errors.New("bite provider is nil")
	}
	response, err := g.Provider.Chat(ctx, ChatRequest{Messages: []ChatMessage{{Role: "system", Content: biteSystemPrompt}, {Role: "user", Content: noteBody}}})
	if err != nil {
		return nil, err
	}
	var output struct {
		Bites []review.Bite `json:"bites"`
	}
	dec := json.NewDecoder(strings.NewReader(response.Content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&output); err != nil {
		return nil, fmt.Errorf("decode bites: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("trailing JSON")
		}
		return nil, fmt.Errorf("trailing JSON: %w", err)
	}
	if len(output.Bites) < 1 || len(output.Bites) > 8 {
		return nil, fmt.Errorf("generator returned %d bites", len(output.Bites))
	}
	for i, b := range output.Bites {
		b.Prompt = strings.TrimSpace(b.Prompt)
		b.Answer = strings.TrimSpace(b.Answer)
		if b.Prompt == "" || b.Answer == "" {
			return nil, errors.New("bite prompt and answer must be non-empty")
		}
		output.Bites[i] = b
	}
	return output.Bites, nil
}
