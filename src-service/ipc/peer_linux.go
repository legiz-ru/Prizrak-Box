//go:build linux

package ipc

import (
	"fmt"
	"net"
	"syscall"
)

// peerIdentity is who is on the other end of an IPC connection.
type peerIdentity struct {
	Known bool // credentials could be read at all
	Uid   int
	Gid   int
	Pid   int
}

// peerOf reads the connecting process' credentials from the kernel via
// SO_PEERCRED. They are set by the kernel at connect() time and cannot be forged
// by the client, which is what makes them usable for authorization.
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
		cred     *syscall.Ucred
		credErr  error
		controlE error
	)
	controlE = raw.Control(func(fd uintptr) {
		cred, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if controlE != nil || credErr != nil || cred == nil {
		return peerIdentity{}
	}
	return peerIdentity{Known: true, Uid: int(cred.Uid), Gid: int(cred.Gid), Pid: int(cred.Pid)}
}

func (p peerIdentity) String() string {
	if !p.Known {
		return "unknown peer"
	}
	return fmt.Sprintf("uid=%d gid=%d pid=%d", p.Uid, p.Gid, p.Pid)
}
