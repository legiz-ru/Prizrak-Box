//go:build windows

package ipc

import (
	"log"
	"net"

	"github.com/Microsoft/go-winio"
)

// createListeners creates the Windows named pipe listener.
//
// The pipe keeps its "allow everyone" DACL: the GUI runs unelevated and has to be
// able to talk to the LocalSystem service. What a caller may ask for is
// restricted per command instead (see validatePxPath), so a reachable pipe is no
// longer a way to have the service execute an arbitrary binary as LocalSystem.
func createListeners() ([]net.Listener, error) {
	config := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;WD)", // Allow everyone
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}
	ln, err := winio.ListenPipe(WindowsPipeName, config)
	if err != nil {
		return nil, err
	}
	log.Printf("[IPC] listening on %s", WindowsPipeName)
	return []net.Listener{ln}, nil
}

// removeSockets has no Windows counterpart: closing the pipe listener is enough.
func removeSockets() {}
