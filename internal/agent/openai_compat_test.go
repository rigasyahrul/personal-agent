package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompatConvertsRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer secret" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("request = %s, headers %#v", r.URL.Path, r.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-test" || body["temperature"] != 0.25 {
			t.Errorf("body = %#v", body)
		}
		messages := body["messages"].([]any)
		if messages[0].(map[string]any)["content"] != "hello" {
			t.Errorf("messages = %#v", messages)
		}
		if _, exists := body["tools"]; exists {
			t.Errorf("unexpected tools: %#v", body["tools"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi","tool_calls":[{"id":"c1","type":"function","function":{"name":"read","arguments":"{\"path\":\"x\"}"}}]}}]}`))
	}))
	defer server.Close()

	provider := &OpenAICompat{BaseURL: server.URL + "/v1/", APIKey: "secret"}
	got, err := provider.Chat(context.Background(), ChatRequest{Model: "gpt-test", Messages: []ChatMessage{{Role: "user", Content: "hello"}}, Parameters: map[string]any{"temperature": 0.25}})
	if err != nil || got.Content != "hi" || len(got.ToolCalls) != 1 || got.ToolCalls[0].Name != "read" || len(got.Raw) == 0 {
		t.Fatalf("response = %#v, %v", got, err)
	}
}

func TestOpenAICompatErrors(t *testing.T) {
	for _, tc := range []struct{ name, response, want string }{
		{name: "non-2xx bounded", response: strings.Repeat("x", 5000), want: "provider status 500: " + strings.Repeat("x", 4096)},
		{name: "empty choices", response: `{"choices":[]}`, want: "provider returned no choices"},
		{name: "invalid JSON", response: `{`, want: "unexpected end"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.name == "non-2xx bounded" {
					w.WriteHeader(http.StatusInternalServerError)
				}
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			_, err := (&OpenAICompat{BaseURL: server.URL}).Chat(context.Background(), ChatRequest{Model: "m"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
