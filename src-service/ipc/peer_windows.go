//go:build windows

package ipc

import "net"

// peerIdentity is who is on the other end of an IPC connection. Windows named
// pipes carry the client's token rather than a uid/gid pair, and the checks that
// use this on Unix (data-directory ownership) have no meaning there, so the
// identity is deliberately left unknown: authorization on Windows rests on the
// px path validation, which is platform-neutral.
type peerIdentity struct {
	Known bool
	Uid   int
	Gid   int
	Pid   int
}

func peerOf(_ net.Conn) peerIdentity { return peerIdentity{} }

func (p peerIdentity) String() string { return "named-pipe client" }
