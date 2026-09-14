//go:build windows

package ipc

// ownerOf / restoreOwnership exist for the Unix problem they solve: px started by
// the service runs as root and leaves root-owned files in the user's data
// directory. Windows has no equivalent break — the service writes into the
// user's profile path with inherited ACLs — so these are no-ops.
func ownerOf(_ string) (uid, gid int, ok bool) { return 0, 0, false }

func restoreOwnership(_ string, _, _ int) error { return nil }
