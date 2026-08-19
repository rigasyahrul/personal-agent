package fsroot

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/rigasyahrul/personal-agent/internal/paths"
	"golang.org/x/sys/unix"
)

var (
	ErrInvalidPath = errors.New("invalid rooted path")
	ErrUnsafe      = errors.New("unsafe filesystem node")
)

type Root struct {
	root *os.Root
	mu   sync.Mutex
}

type Entry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size,omitempty"`
}

func Open(name string) (*Root, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	// Resolve directory-name symlinks (macOS /var → /private/var, user links)
	// before opening. os.OpenRoot follows them too; we pin the FD to the final
	// real directory. Escape via symlinks *inside* the root is still blocked by
	// per-path Lstat checks and os.Root.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrUnsafe
	}
	r, err := os.OpenRoot(resolved)
	if err != nil {
		return nil, err
	}
	return &Root{root: r}, nil
}

func (r *Root) Close() error {
	return r.root.Close()
}

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

// WriteFileAtomic replaces a regular file using a temporary sibling on the
// same filesystem. It never follows symlinks or replaces special nodes.
func (r *Root) WriteFileAtomic(name string, body []byte, perm fs.FileMode) error {
	if err := valid(name); err != nil {
		return err
	}
	if len(body) > paths.MaxMarkdownBytes {
		return ErrUnsafe
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	dir := filepath.ToSlash(filepath.Dir(name))
	if dir == "." {
		dir = ""
	}
	parent := r.root
	if dir != "" {
		var err error
		parent, err = r.root.OpenRoot(dir)
		if err != nil {
			return ErrUnsafe
		}
		defer parent.Close()
	}
	parentDir, err := parent.Open(".")
	if err != nil {
		return err
	}
	defer parentDir.Close()
	parentFD := int(parentDir.Fd())
	final := filepath.Base(name)
	if info, err := parent.Lstat(final); err == nil {
		if !info.Mode().IsRegular() {
			return ErrUnsafe
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	tmp := ".pa-write-" + hex.EncodeToString(random)
	f, err := parent.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm.Perm())
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = parent.Remove(tmp)
		}
	}()
	if _, err = f.Write(body); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	// Link is an atomic no-replace commit. It closes the check/rename race for
	// newly created destinations: a node appearing during the write survives.
	if _, err := parent.Lstat(final); errors.Is(err, fs.ErrNotExist) {
		if err := unix.Linkat(parentFD, tmp, parentFD, final, 0); err != nil {
			if errors.Is(err, fs.ErrExist) {
				return ErrUnsafe
			}
			return err
		}
		if err := parent.Remove(tmp); err != nil {
			return err
		}
		remove = false
		return syncDir(parent)
	} else if err != nil {
		return err
	} else if info, err := parent.Lstat(final); err != nil || !info.Mode().IsRegular() {
		return ErrUnsafe
	}
	// Exchange keeps the prior destination available at tmp so its type can be
	// validated after the atomic commit operation. If a special node won the
	// race after Lstat, put it back rather than replacing it.
	if err := exchangeRename(parentFD, tmp, parentFD, final); err != nil {
		return err
	}
	oldInfo, err := parent.Lstat(tmp)
	if err != nil || !oldInfo.Mode().IsRegular() {
		if restoreErr := exchangeRename(parentFD, tmp, parentFD, final); restoreErr != nil {
			return restoreErr
		}
		return ErrUnsafe
	}
	if err := parent.Remove(tmp); err != nil {
		return err
	}
	remove = false
	return syncDir(parent)
}

func syncDir(root *os.Root) error {
	d, err := root.Open(".")
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (r *Root) EditFileAtomic(name, old, replacement string) error {
	b, err := r.ReadFile(name, paths.MaxMarkdownBytes)
	if err != nil {
		return err
	}
	if old == "" || bytes.Count(b, []byte(old)) != 1 {
		return errors.New("old text must occur exactly once")
	}
	return r.WriteFileAtomic(name, bytes.Replace(b, []byte(old), []byte(replacement), 1), 0644)
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

// WriteFileNoReplace atomically installs a regular file without following
// symlinks or replacing an existing destination.
func (r *Root) WriteFileNoReplace(name string, body []byte, perm fs.FileMode) error {
	if err := valid(name); err != nil {
		return err
	}
	dir := filepath.ToSlash(filepath.Dir(name))
	if dir == "." {
		dir = ""
	}
	if dir != "" {
		if err := r.MkdirAll(dir, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	parent := r.root
	if dir != "" {
		var err error
		parent, err = r.root.OpenRoot(dir)
		if err != nil {
			return ErrUnsafe
		}
		defer parent.Close()
	}
	parentDir, err := parent.Open(".")
	if err != nil {
		return err
	}
	defer parentDir.Close()
	parentFD := int(parentDir.Fd())
	final := filepath.Base(name)
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	tmp := ".publish-" + hex.EncodeToString(random)
	f, err := parent.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm.Perm())
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = parent.Remove(tmp)
		}
	}()
	if _, err = f.Write(body); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	err = unix.Linkat(parentFD, tmp, parentFD, final, 0)
	if errors.Is(err, fs.ErrExist) {
		return fs.ErrExist
	}
	if err != nil {
		return err
	}
	if err := parent.Remove(tmp); err != nil {
		return err
	}
	remove = false
	return syncDir(parent)
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

func (r *Root) Tree() ([]Entry, error) {
	var entries []Entry
	err := r.Walk(func(name string, info fs.FileInfo) error {
		kind := "file"
		if info.IsDir() {
			kind = "directory"
		} else if !info.Mode().IsRegular() {
			return ErrUnsafe
		}
		entries = append(entries, Entry{Path: name, Kind: kind, Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}
