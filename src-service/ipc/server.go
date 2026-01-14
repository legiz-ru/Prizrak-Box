package ipc

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
	"runtime"
	"sync"

	"github.com/legiz-ru/prizrak-box-service/manager"
)

// Константы для IPC
const (
	// Windows named pipe
	WindowsPipeName = `\\.\pipe\prizrak-box-service`
	// Unix socket path
	UnixSocketPath = "/tmp/prizrak-box-service.sock"
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
	listener net.Listener
	running  bool
	mu       sync.Mutex
	wg       sync.WaitGroup
}

// NewServer создаёт новый IPC сервер
func NewServer() *Server {
	return &Server{}
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

	var err error

	if runtime.GOOS == "windows" {
		s.listener, err = createWindowsListener()
	} else {
		// Удаляем старый сокет если существует
		os.Remove(UnixSocketPath)
		s.listener, err = net.Listen("unix", UnixSocketPath)
		if err == nil {
			// Устанавливаем права доступа
			os.Chmod(UnixSocketPath, 0666)
		}
	}

	if err != nil {
		return err
	}

	log.Printf("[IPC] Server listening...")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if !running {
				return nil
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
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Unlock()

	s.wg.Wait()

	// Удаляем Unix сокет
	if runtime.GOOS != "windows" {
		os.Remove(UnixSocketPath)
	}
}

// handleConnection обрабатывает подключение
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.wg.Done()
	}()

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

		response := s.handleRequest(req)
		s.sendResponse(conn, response)
	}
}

// handleRequest обрабатывает запрос
func (s *Server) handleRequest(req Request) Response {
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

		if data.PxPath == "" {
			return Response{Success: false, Error: "pxPath is required"}
		}

		err := manager.StartPx(data.PxPath, data.Addr, data.HomeDir)
		if err != nil {
			return Response{Success: false, Error: err.Error()}
		}
		return Response{Success: true}

	case "stop_px":
		manager.StopPx()
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
