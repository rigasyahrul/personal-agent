package fsroot

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
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
	tmp := filepath.ToSlash(filepath.Join(dir, fmt.Sprintf(".publish-%d-%d", os.Getpid(), unix.Gettid())))
	tfd, err := unix.Openat(r.fd, tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(perm.Perm()))
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = unix.Unlinkat(r.fd, tmp, 0)
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
	err = unix.Renameat2(r.fd, tmp, r.fd, name, unix.RENAME_NOREPLACE)
	if errors.Is(err, unix.EEXIST) {
		return fs.ErrExist
	}
	if errors.Is(err, unix.ELOOP) || errors.Is(err, unix.ENOTDIR) {
		return ErrUnsafe
	}
	if err != nil {
		return err
	}
	remove = false
	dfd, err := unix.Openat(r.fd, func() string {
		if dir == "" {
			return "."
		}
		return dir
	}(), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(dfd)
	return unix.Fsync(dfd)
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
