//go:build darwin

package fsroot

import "golang.org/x/sys/unix"

// exchangeRename atomically swaps two directory entries under the given
// directory file descriptors (Darwin renameatx_np RENAME_SWAP).
func exchangeRename(olddirfd int, oldpath string, newdirfd int, newpath string) error {
	return unix.RenameatxNp(olddirfd, oldpath, newdirfd, newpath, unix.RENAME_SWAP)
}
