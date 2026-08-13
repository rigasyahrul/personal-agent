package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type fakeBiteProvider struct{ response string }

func (f fakeBiteProvider) Chat(context.Context, ChatRequest) (ChatResponse, error) {
	return ChatResponse{Content: f.response}, nil
}

func TestProviderBiteGeneratorStrictJSON(t *testing.T) {
	valid := func(n int) string {
		var b strings.Builder
		b.WriteString(`{"bites":[`)
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, `{"prompt":"p%d","answer":"a%d"}`, i, i)
		}
		b.WriteString(`]}`)
		return b.String()
	}
	for _, n := range []int{1, 8} {
		t.Run(fmt.Sprintf("accept_%d", n), func(t *testing.T) {
			got, err := (ProviderBiteGenerator{Provider: fakeBiteProvider{valid(n)}}).Generate(context.Background(), "note")
			if err != nil || len(got) != n {
				t.Fatalf("len=%d err=%v", len(got), err)
			}
		})
	}
	cases := map[string]string{"zero": valid(0), "nine": valid(9), "empty": `{"bites":[{"prompt":" ","answer":"a"}]}`, "unknown": `{"bites":[{"prompt":"p","answer":"a","x":1}]}`, "trailing": valid(1) + ` {}`}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := (ProviderBiteGenerator{Provider: fakeBiteProvider{payload}}).Generate(context.Background(), "note"); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
