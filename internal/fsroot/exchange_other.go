//go:build !linux && !darwin

package fsroot

import "fmt"

// exchangeRename is unavailable on this platform.
func exchangeRename(olddirfd int, oldpath string, newdirfd int, newpath string) error {
	return fmt.Errorf("atomic file exchange is not supported on this platform")
}
