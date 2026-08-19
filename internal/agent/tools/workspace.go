package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

type Result struct {
	Content     string `json:"content,omitempty"`
	ChangedPath string `json:"changed_path,omitempty"`
}

// MutBarrier is implemented by *backup.Barrier (local to avoid import cycles).
type MutBarrier interface {
	Mutate(func() error) error
}

type Workspace struct {
	root    *fsroot.Root
	Barrier MutBarrier
}

func NewWorkspace(root *fsroot.Root) *Workspace { return &Workspace{root: root} }

func (w *Workspace) withMutate(fn func() error) error {
	if w.Barrier == nil {
		return fn()
	}
	return w.Barrier.Mutate(fn)
}

func decode(raw json.RawMessage, dst any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("exactly one JSON object required")
	}
	return nil
}

func (w *Workspace) Execute(ctx context.Context, name string, raw json.RawMessage) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	switch name {
	case "read_file":
		var a struct {
			Path string `json:"path"`
		}
		if err := decode(raw, &a); err != nil {
			return Result{}, err
		}
		b, err := w.root.ReadFile(a.Path, paths.MaxMarkdownBytes)
		return Result{Content: string(b)}, err
	case "write_file":
		var a struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := decode(raw, &a); err != nil {
			return Result{}, err
		}
		if len(a.Content) > paths.MaxMarkdownBytes {
			return Result{}, fmt.Errorf("content exceeds %d bytes", paths.MaxMarkdownBytes)
		}
		if err := w.withMutate(func() error {
			return w.root.WriteFileAtomic(a.Path, []byte(a.Content), 0644)
		}); err != nil {
			return Result{}, err
		}
		return Result{ChangedPath: a.Path}, nil
	case "edit_file":
		var a struct {
			Path        string `json:"path"`
			Old         string `json:"old"`
			Replacement string `json:"replacement"`
		}
		if err := decode(raw, &a); err != nil {
			return Result{}, err
		}
		if err := w.withMutate(func() error {
			return w.root.EditFileAtomic(a.Path, a.Old, a.Replacement)
		}); err != nil {
			return Result{}, err
		}
		return Result{ChangedPath: a.Path}, nil
	case "mkdir":
		var a struct {
			Path string `json:"path"`
		}
		if err := decode(raw, &a); err != nil {
			return Result{}, err
		}
		if err := w.withMutate(func() error {
			return w.root.MkdirAll(a.Path, 0755)
		}); err != nil {
			return Result{}, err
		}
		return Result{ChangedPath: a.Path}, nil
	default:
		return Result{}, fmt.Errorf("workspace tool %q is not allowed", name)
	}
}
