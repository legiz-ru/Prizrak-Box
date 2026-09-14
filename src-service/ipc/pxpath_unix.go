//go:build !windows

package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// trustedSystemPath reports whether only the superuser can change what this path
// executes: the binary and the directory holding it must both be owned by root
// and must not be writable by anyone else. A path anywhere a normal user can
// write (their home, /tmp) is rejected, because they could swap the file between
// this check and the exec.
func trustedSystemPath(resolved string) error {
	for _, p := range []string{resolved, filepath.Dir(resolved)} {
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("cannot read ownership of %q", p)
		}
		if stat.Uid != 0 {
			return fmt.Errorf("%q is not owned by root", p)
		}
		if info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("%q is writable by non-root users", p)
		}
	}
	return nil
}
