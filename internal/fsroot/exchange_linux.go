//go:build linux

package fsroot

import "golang.org/x/sys/unix"

// exchangeRename atomically swaps two directory entries under the given
// directory file descriptors (Linux renameat2 RENAME_EXCHANGE).
func exchangeRename(olddirfd int, oldpath string, newdirfd int, newpath string) error {
	return unix.Renameat2(olddirfd, oldpath, newdirfd, newpath, unix.RENAME_EXCHANGE)
}
