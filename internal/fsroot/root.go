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

	"github.com/rigasyahrul/personal-agent/internal/paths"
	"golang.org/x/sys/unix"
)

var (
	ErrInvalidPath = errors.New("invalid rooted path")
	ErrUnsafe      = errors.New("unsafe filesystem node")
)

type Root struct {
	root *os.Root
	fd   int
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
	fd, err := unix.Openat2(unix.AT_FDCWD, abs, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS})
	if err != nil {
		_ = r.Close()
		return nil, ErrUnsafe
	}
	return &Root{root: r, fd: fd}, nil
}

func (r *Root) Close() error {
	a := r.root.Close()
	b := unix.Close(r.fd)
	if a != nil {
		return a
	}
	return b
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
	dir := filepath.ToSlash(filepath.Dir(name))
	if dir == "." {
		dir = ""
	}
	parentFD := r.fd
	var err error
	if dir != "" {
		parentFD, err = unix.Openat2(r.fd, dir, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS})
		if errors.Is(err, unix.ELOOP) || errors.Is(err, unix.EXDEV) || errors.Is(err, unix.ENOTDIR) {
			return ErrUnsafe
		}
		if err != nil {
			return err
		}
		defer unix.Close(parentFD)
	}
	final := filepath.Base(name)
	var stat unix.Stat_t
	if err := unix.Fstatat(parentFD, final, &stat, unix.AT_SYMLINK_NOFOLLOW); err == nil {
		if stat.Mode&unix.S_IFMT != unix.S_IFREG {
			return ErrUnsafe
		}
	} else if !errors.Is(err, unix.ENOENT) {
		return err
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	tmp := ".pa-write-" + hex.EncodeToString(random)
	tfd, err := unix.Openat(parentFD, tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(perm.Perm()))
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = unix.Unlinkat(parentFD, tmp, 0)
		}
	}()
	f := os.NewFile(uintptr(tfd), tmp)
	if _, err = f.Write(body); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := unix.Renameat(parentFD, tmp, parentFD, final); err != nil {
		return err
	}
	remove = false
	return unix.Fsync(parentFD)
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
	parentFD := r.fd
	var err error
	if dir != "" {
		parentFD, err = unix.Openat2(r.fd, dir, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS})
		if errors.Is(err, unix.ELOOP) || errors.Is(err, unix.EXDEV) || errors.Is(err, unix.ENOTDIR) {
			return ErrUnsafe
		}
		if err != nil {
			return err
		}
		defer unix.Close(parentFD)
	}
	final := filepath.Base(name)
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	tmp := ".publish-" + hex.EncodeToString(random)
	tfd, err := unix.Openat(parentFD, tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(perm.Perm()))
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = unix.Unlinkat(parentFD, tmp, 0)
		}
	}()
	f := os.NewFile(uintptr(tfd), tmp)
	if _, err = f.Write(body); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	err = unix.Renameat2(parentFD, tmp, parentFD, final, unix.RENAME_NOREPLACE)
	if errors.Is(err, unix.EEXIST) {
		return fs.ErrExist
	}
	if errors.Is(err, unix.ELOOP) || errors.Is(err, unix.EXDEV) || errors.Is(err, unix.ENOTDIR) {
		return ErrUnsafe
	}
	if err != nil {
		return err
	}
	remove = false
	return unix.Fsync(parentFD)
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
