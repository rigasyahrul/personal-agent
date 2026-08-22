package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/store"
)

// ParseCompoundItemsFromAssistant extracts the first ```json ... ``` fence
// or a raw JSON array from the assistant message.
func ParseCompoundItemsFromAssistant(content string) ([]store.CompoundItem, error) {
	raw, err := extractJSONArray(content)
	if err != nil {
		return nil, err
	}
	var items []store.CompoundItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("compound items json: %w", err)
	}
	if items == nil {
		return nil, fmt.Errorf("compound items json: expected array")
	}
	return items, nil
}

func extractJSONArray(content string) ([]byte, error) {
	s := strings.TrimSpace(content)
	if s == "" {
		return nil, fmt.Errorf("empty assistant content")
	}
	lower := strings.ToLower(s)
	const fence = "```json"
	if idx := strings.Index(lower, fence); idx >= 0 {
		rest := s[idx+len(fence):]
		end := strings.Index(rest, "```")
		if end < 0 {
			return nil, fmt.Errorf("unclosed json fence")
		}
		inner := strings.TrimSpace(rest[:end])
		if inner == "" {
			return nil, fmt.Errorf("empty json fence")
		}
		return []byte(inner), nil
	}
	if strings.HasPrefix(s, "[") {
		return []byte(s), nil
	}
	return nil, fmt.Errorf("no json array in assistant content")
}
