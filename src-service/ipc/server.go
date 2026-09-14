package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"github.com/legiz-ru/prizrak-box-service/manager"
)

// Константы для IPC
const (
	// Windows named pipe
	WindowsPipeName = `\\.\pipe\prizrak-box-service`
)

// Request представляет IPC запрос
type Request struct {
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Response представляет IPC ответ
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// StartPxRequest данные для запуска px
type StartPxRequest struct {
	PxPath  string `json:"pxPath"`
	Addr    string `json:"addr"`
	HomeDir string `json:"homeDir"`
}

// Server представляет IPC сервер
type Server struct {
	listeners []net.Listener
	running   bool
	mu        sync.Mutex
	wg        sync.WaitGroup

	// trustAnyPxPath relaxes the px path validation for standalone (debug) runs,
	// where px is built in the repo next to the sources rather than installed
	// into a root-owned directory.
	trustAnyPxPath bool

	// dataDir and dataUid/dataGid remember the data directory of the most recent
	// start_px and who owns it, so ownership can be restored when px stops (see
	// restoreOwnership).
	dataDir string
	dataUid int
	dataGid int
}

// NewServer создаёт новый IPC сервер
func NewServer() *Server {
	return &Server{dataUid: -1, dataGid: -1}
}

// NewStandaloneServer creates a server for debug runs (px-service -standalone),
// where px is not in an installed, root-owned location.
func NewStandaloneServer() *Server {
	s := NewServer()
	s.trustAnyPxPath = true
	return s
}

// Start запускает IPC сервер
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	listeners, err := createListeners()
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.listeners = listeners
	s.mu.Unlock()

	log.Printf("[IPC] Server listening...")

	// Every endpoint is served by its own accept loop; Start blocks until they
	// all return (i.e. until Stop closes them), preserving its previous
	// behaviour for the caller in main.go.
	var loops sync.WaitGroup
	for _, ln := range listeners {
		loops.Add(1)
		go func(ln net.Listener) {
			defer loops.Done()
			s.acceptLoop(ln)
		}(ln)
	}
	loops.Wait()
	return nil
}

func (s *Server) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if !running {
				return
			}
			log.Printf("[IPC] Accept error: %v", err)
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// Stop останавливает IPC сервер
func (s *Server) Stop() {
	s.mu.Lock()
	s.running = false
	listeners := s.listeners
	s.listeners = nil
	s.mu.Unlock()

	for _, ln := range listeners {
		_ = ln.Close()
	}

	s.wg.Wait()

	removeSockets()
}

// handleConnection обрабатывает подключение
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.wg.Done()
	}()

	peer := peerOf(conn)
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendResponse(conn, Response{
				Success: false,
				Error:   "Invalid request format",
			})
			continue
		}

		response := s.handleRequest(req, peer)
		s.sendResponse(conn, response)
	}
}

// handleRequest обрабатывает запрос
func (s *Server) handleRequest(req Request, peer peerIdentity) Response {
	switch req.Command {
	case "ping":
		return Response{Success: true, Data: "pong"}

	case "version":
		return Response{Success: true, Data: "1.0.0"}

	case "start_px":
		var data StartPxRequest
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{Success: false, Error: "Invalid start_px data: " + err.Error()}
		}
		return s.startPx(data, peer)

	case "stop_px":
		manager.StopPx()
		s.restoreDataOwnership()
		return Response{Success: true}

	case "status":
		running := manager.IsPxRunning()
		return Response{Success: true, Data: map[string]interface{}{
			"px_running": running,
		}}

	case "is_admin":
		return Response{Success: true, Data: isRunningAsAdmin()}

	default:
		return Response{Success: false, Error: "Unknown command: " + req.Command}
	}
}

// startPx validates the request, launches px and restores the data directory's
// ownership.
func (s *Server) startPx(data StartPxRequest, peer peerIdentity) Response {
	pxPath := data.PxPath
	if !s.trustAnyPxPath {
		validated, err := validatePxPath(data.PxPath)
		if err != nil {
			log.Printf("[IPC] start_px rejected (%s): %v", peer, err)
			return Response{Success: false, Error: err.Error()}
		}
		pxPath = validated
	} else if pxPath == "" {
		return Response{Success: false, Error: "pxPath is required"}
	}

	uid, gid, err := s.authorizeDataDir(data.HomeDir, peer)
	if err != nil {
		log.Printf("[IPC] start_px rejected (%s): %v", peer, err)
		return Response{Success: false, Error: err.Error()}
	}

	if err := manager.StartPx(pxPath, data.Addr, data.HomeDir); err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	s.mu.Lock()
	s.dataDir, s.dataUid, s.dataGid = data.HomeDir, uid, gid
	s.mu.Unlock()

	// px has only just started, so this mostly repairs what a previous privileged
	// run (or a reboot during one) left behind.
	s.restoreDataOwnership()
	return Response{Success: true}
}

// authorizeDataDir checks that the caller may have px run against this data
// directory and returns the uid/gid its contents must end up owned by.
//
// The service runs as root and its endpoint is reachable by any local user, so
// the directory's owner is the one identity in the request that cannot be chosen
// freely: a caller may only point px at a data directory that is their own.
func (s *Server) authorizeDataDir(homeDir string, peer peerIdentity) (uid, gid int, err error) {
	if homeDir == "" {
		return -1, -1, nil
	}
	ownerUid, ownerGid, ok := ownerOf(homeDir)
	if !ok {
		// Either the directory does not exist yet (px creates it) or ownership is
		// not a concept here (Windows). Fall back to the caller's own identity.
		if peer.Known {
			return peer.Uid, peer.Gid, nil
		}
		return -1, -1, nil
	}
	if peer.Known && peer.Uid != 0 && peer.Uid != ownerUid {
		return -1, -1, fmt.Errorf("data directory %q belongs to uid %d, not to the caller", homeDir, ownerUid)
	}
	return ownerUid, ownerGid, nil
}

// ShutdownPx stops px and hands its data directory back to the user. Used on
// service shutdown, where skipping the ownership repair would leave the files
// root-owned until the next start_px.
func (s *Server) ShutdownPx() {
	manager.StopPx()
	s.restoreDataOwnership()
}

// restoreDataOwnership hands the remembered data directory back to its user.
func (s *Server) restoreDataOwnership() {
	s.mu.Lock()
	dir, uid, gid := s.dataDir, s.dataUid, s.dataGid
	s.mu.Unlock()
	if dir == "" || uid < 0 {
		return
	}
	if _, err := os.Stat(dir); err != nil {
		return
	}
	if err := restoreOwnership(dir, uid, gid); err != nil {
		log.Printf("[IPC] could not fully restore ownership of %s to %d:%d: %v", dir, uid, gid, err)
	}
}

// sendResponse отправляет ответ
func (s *Server) sendResponse(conn net.Conn, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[IPC] Failed to marshal response: %v", err)
		return
	}

	data = append(data, '\n')
	conn.Write(data)
}
