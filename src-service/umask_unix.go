//go:build !windows

package main

import "syscall"

// relaxUmask makes files this service (and the px it spawns) creates readable by
// the user who owns them once ownership is restored.
//
// The installed systemd unit sets UMask=0022 for the same reason, but a service
// registered by an older package still carries UMask=0077, under which root-px
// writes mode-0600 files: after restoreOwnership hands them to the user they are
// at least usable, yet anything created between start and the repair is
// unreadable to them. Setting it here covers that case too, and is inherited by
// px.
func relaxUmask() { syscall.Umask(0o022) }
