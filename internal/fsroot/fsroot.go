package fsroot

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/paths"
)

var (
	ErrInvalidPath = errors.New("invalid rooted path")
	ErrUnsafe      = errors.New("unsafe filesystem node")
)

type Root struct{ root *os.Root }

func Open(name string) (*Root, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrUnsafe
	}
	r, err := os.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	return &Root{root: r}, nil
}

func (r *Root) Close() error { return r.root.Close() }

func valid(name string) error {
	if _, err := paths.ValidateRelPath(name); err != nil || strings.Contains(name, `\`) {
		return ErrInvalidPath
	}
	return nil
}

func (r *Root) ReadFile(name string) ([]byte, error) {
	if err := valid(name); err != nil {
		return nil, err
	}
	info, err := r.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrUnsafe
	}
	f, err := r.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (r *Root) MkdirAll(name string, perm fs.FileMode) error {
	if err := valid(name); err != nil {
		return err
	}
	current := ""
	for _, part := range strings.Split(name, "/") {
		if current == "" {
			current = part
		} else {
			current += "/" + part
		}
		info, err := r.root.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			if err := r.root.Mkdir(current, perm); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrUnsafe
		}
		if current == name {
			return fs.ErrExist
		}
	}
	return nil
}

func (r *Root) Walk(fn func(path string, info fs.FileInfo) error) error {
	var walk func(string) error
	walk = func(dir string) error {
		f, err := r.root.Open(dir)
		if err != nil {
			return err
		}
		entries, err := f.ReadDir(-1)
		_ = f.Close()
		if err != nil {
			return err
		}
		for _, entry := range entries {
			name := entry.Name()
			if dir != "." {
				name = dir + "/" + name
			}
			info, err := r.root.Lstat(name)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return ErrUnsafe
			}
			if err := fn(name, info); err != nil {
				return err
			}
			if info.IsDir() {
				if err := walk(name); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(".")
}
