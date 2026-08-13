package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type OpenAICompat struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

func (p *OpenAICompat) Chat(ctx context.Context, in ChatRequest) (ChatResponse, error) {
	body := make(map[string]any, len(in.Parameters)+3)
	for key, value := range in.Parameters {
		if key != "model" && key != "messages" && key != "tools" {
			body[key] = value
		}
	}
	body["model"] = in.Model
	messages := make([]openAIMessage, 0, len(in.Messages))
	for _, message := range in.Messages {
		wire := openAIMessage{Role: message.Role, Content: message.Content, ToolCallID: message.ToolCallID}
		for _, call := range message.ToolCalls {
			wire.ToolCalls = append(wire.ToolCalls, toOpenAIToolCall(call))
		}
		messages = append(messages, wire)
	}
	body["messages"] = messages
	if len(in.Tools) > 0 {
		tools := make([]any, 0, len(in.Tools))
		for _, tool := range in.Tools {
			tools = append(tools, map[string]any{"type": "function", "function": tool})
		}
		body["tools"] = tools
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	baseURL := strings.TrimRight(p.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return ChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return ChatResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return ChatResponse{}, fmt.Errorf("provider status %d: %s", resp.StatusCode, body)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, err
	}
	var out struct {
		Choices []struct {
			Message openAIMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return ChatResponse{}, err
	}
	if len(out.Choices) == 0 {
		return ChatResponse{}, errors.New("provider returned no choices")
	}
	result := ChatResponse{Content: out.Choices[0].Message.Content, Raw: json.RawMessage(raw)}
	for _, call := range out.Choices[0].Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments})
	}
	return result, nil
}

func toOpenAIToolCall(call ToolCall) openAIToolCall {
	result := openAIToolCall{ID: call.ID, Type: "function"}
	result.Function.Name = call.Name
	result.Function.Arguments = call.Arguments
	return result
}
