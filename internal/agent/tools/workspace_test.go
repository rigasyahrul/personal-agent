package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/agent/tools"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

func TestWorkspaceToolsAcceptText(t *testing.T) {
	r, err := fsroot.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	w := tools.NewWorkspace(r)
	for _, tc := range []struct{ name, args string }{
		{"mkdir", `{"path":"research"}`},
		{"write_file", `{"path":"research/raw.txt","content":"first draft"}`},
		{"edit_file", `{"path":"research/raw.txt","old":"first","replacement":"second"}`},
		{"read_file", `{"path":"research/raw.txt"}`},
	} {
		if _, err := w.Execute(context.Background(), tc.name, json.RawMessage(tc.args)); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
	got, _ := r.ReadFile("research/raw.txt", 100)
	if string(got) != "second draft" {
		t.Fatalf("got %q", got)
	}
}

func TestWorkspaceToolsRejectUnsafeMalformedAndOversizedRequests(t *testing.T) {
	r, err := fsroot.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	w := tools.NewWorkspace(r)
	for _, tc := range []struct{ name, args string }{
		{"write_file", `{"path":"../x","content":"x"}`},
		{"read_file", `{"path":"x","extra":true}`},
		{"read_file", `{"path":"x"} {"path":"y"}`},
		{"shell", `{"command":"id"}`},
		{"write_file", `{"path":"big.txt","content":"` + strings.Repeat("x", paths.MaxMarkdownBytes+1) + `"}`},
	} {
		if _, err := w.Execute(context.Background(), tc.name, json.RawMessage(tc.args)); err == nil {
			t.Fatalf("%s unexpectedly accepted request", tc.name)
		}
	}
}

func TestWorkspaceToolsHonorCanceledContext(t *testing.T) {
	r, err := fsroot.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tools.NewWorkspace(r).Execute(ctx, "mkdir", json.RawMessage(`{"path":"x"}`)); err == nil {
		t.Fatal("canceled request accepted")
	}
}
