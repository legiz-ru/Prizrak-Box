//go:build darwin

package ipc

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// peerIdentity is who is on the other end of an IPC connection.
type peerIdentity struct {
	Known bool // credentials could be read at all
	Uid   int
	Gid   int
	Pid   int // not reported by LOCAL_PEERCRED; always 0 here
}

// peerOf reads the connecting process' credentials from the kernel.
//
// macOS has no SO_PEERCRED; the equivalent is LOCAL_PEERCRED, which yields an
// xucred (uid plus group list, no pid). Like SO_PEERCRED it is filled in by the
// kernel at connect() time, so the client cannot forge it.
func peerOf(conn net.Conn) peerIdentity {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return peerIdentity{}
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return peerIdentity{}
	}
	var (
		cred    *unix.Xucred
		credErr error
	)
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil || credErr != nil || cred == nil {
		return peerIdentity{}
	}
	gid := 0
	if cred.Ngroups > 0 {
		gid = int(cred.Groups[0])
	}
	return peerIdentity{Known: true, Uid: int(cred.Uid), Gid: gid}
}

func (p peerIdentity) String() string {
	if !p.Known {
		return "unknown peer"
	}
	return fmt.Sprintf("uid=%d gid=%d", p.Uid, p.Gid)
}
