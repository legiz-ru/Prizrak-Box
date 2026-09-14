//go:build !windows

package ipc

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// ownerOf returns the uid/gid owning path.
func ownerOf(path string) (uid, gid int, ok bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, false
	}
	stat, isStat := info.Sys().(*syscall.Stat_t)
	if !isStat {
		return 0, 0, false
	}
	return int(stat.Uid), int(stat.Gid), true
}

// restoreOwnership gives the data directory back to the desktop user.
//
// px spawned by this service runs as root, so everything it writes into the
// user's data directory — config, profiles, cache.db, logs — ends up owned by
// root. The unprivileged px that runs whenever TUN is off then cannot write
// those files any more, which silently breaks saving profiles and settings after
// the first TUN session. Re-chowning on every start and stop keeps the directory
// usable by both, including after a reboot that interrupted a TUN session.
//
// Entries already owned by the target are skipped, so the common case walks the
// tree without touching anything. Errors are reported but never fatal: a
// partially restored directory is still better than none, and the caller only
// logs this.
func restoreOwnership(root string, uid, gid int) error {
	if root == "" || uid < 0 || gid < 0 {
		return nil
	}
	var firstErr error
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable subtree is not a reason to abandon the rest.
			if firstErr == nil {
				firstErr = err
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return nil
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			if int(stat.Uid) == uid && int(stat.Gid) == gid {
				return nil
			}
		}
		// Lchown, not Chown: a symlink inside the data dir must not be followed
		// out of it.
		if err := os.Lchown(path, uid, gid); err != nil && firstErr == nil {
			firstErr = err
		}
		return nil
	})
	if err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}
