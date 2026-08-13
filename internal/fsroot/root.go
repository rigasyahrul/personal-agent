package fsroot

import (
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/paths"
)

var (
	ErrInvalidPath = errors.New("invalid rooted path")
	ErrUnsafe      = errors.New("unsafe filesystem node")
)

type Root struct{ root *os.Root }

func Open(name string) (*Root, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	components := []string{}
	for current := filepath.Clean(abs); ; current = filepath.Dir(current) {
		parent := filepath.Dir(current)
		if current == parent {
			break
		}
		components = append(components, current)
	}
	for i := len(components) - 1; i >= 0; i-- {
		info, err := os.Lstat(components[i])
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, ErrUnsafe
		}
	}
	r, err := os.OpenRoot(abs)
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

func (r *Root) ReadFile(name string, max int64) ([]byte, error) {
	if err := valid(name); err != nil {
		return nil, err
	}
	if max < 0 || max == math.MaxInt64 {
		return nil, ErrUnsafe
	}
	parts := strings.Split(name, "/")
	current := ""
	for i, part := range parts {
		if current == "" {
			current = part
		} else {
			current += "/" + part
		}
		info, err := r.root.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !info.IsDir()) || (i == len(parts)-1 && !info.Mode().IsRegular()) {
			return nil, ErrUnsafe
		}
		if i == len(parts)-1 && info.Size() > max {
			return nil, ErrUnsafe
		}
	}
	f, err := r.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, ErrUnsafe
	}
	return b, nil
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
			if err := valid(name); err != nil {
				return err
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
