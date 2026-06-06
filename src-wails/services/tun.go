package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/legiz-ru/prizrak-box-wails/internal/locate"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// TunService manages the privileged px-service helper that enables TUN mode
// without running the whole app as administrator. It is the Wails replacement
// for src-electron/service.ts and is bound to the frontend as window.pxService.
type TunService struct {
	core *CoreService
}

// NewTunService creates a TunService bound to the given CoreService.
func NewTunService(core *CoreService) *TunService { return &TunService{core: core} }

// ServiceStatus mirrors the shape the frontend expects from window.pxService.
type ServiceStatus struct {
	Installed bool `json:"installed"`
	Running   bool `json:"running"`
	IsAdmin   bool `json:"isAdmin"`
}

// --- IPC primitives ---------------------------------------------------------

type ipcRequest struct {
	Command string      `json:"command"`
	Data    interface{} `json:"data,omitempty"`
}

type ipcResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type startPxData struct {
	PxPath  string `json:"pxPath"`
	Addr    string `json:"addr"`
	HomeDir string `json:"homeDir"`
}

func (t *TunService) request(cmd string, data interface{}, timeout time.Duration) (*ipcResponse, error) {
	conn, err := dialService(timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	payload, err := json.Marshal(ipcRequest{Command: cmd, Data: data})
	if err != nil {
		return nil, err
	}
	payload = append(payload, '\n')
	if _, err := conn.Write(payload); err != nil {
		return nil, err
	}

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var resp ipcResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (t *TunService) ping() bool {
	resp, err := t.request("ping", nil, 2*time.Second)
	return err == nil && resp.Success
}

// --- Frontend-bound methods -------------------------------------------------

// IsRunning reports whether the px-service IPC endpoint is reachable.
func (t *TunService) IsRunning() bool { return t.ping() }

// GetStatus reports installed/running/admin state of the px-service.
func (t *TunService) GetStatus() ServiceStatus {
	installed := serviceBinaryExists()
	status := ServiceStatus{Installed: installed}
	if !installed {
		return status
	}
	if !t.ping() {
		return status
	}
	status.Running = true
	if resp, err := t.request("is_admin", nil, 2*time.Second); err == nil && resp.Success {
		var admin bool
		_ = json.Unmarshal(resp.Data, &admin)
		status.IsAdmin = admin
	}
	return status
}

// Install installs the px-service (requires elevation) and waits for it to
// come up. Returns true if the service is reachable afterwards.
func (t *TunService) Install() bool {
	bin := locate.ServiceBinary()
	if err := runElevated(bin, "Prizrak-Box requires installing the TUN service", "-install"); err != nil {
		application.Get().Logger.Error("service install failed", "error", err)
		return false
	}
	for i := 0; i < 20; i++ {
		if t.ping() {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return t.ping()
}

// Uninstall removes the px-service (requires elevation).
func (t *TunService) Uninstall() bool {
	bin := locate.ServiceBinary()
	if err := runElevated(bin, "", "-uninstall"); err != nil {
		application.Get().Logger.Error("service uninstall failed", "error", err)
		return false
	}
	return true
}

// RestartBackend restarts px. If the service is running, px is (re)started
// through the elevated service so it can manage TUN; otherwise it is spawned
// directly. Returns the fresh connection info for the frontend.
func (t *TunService) RestartBackend() (ConnInfo, error) {
	if t.ping() {
		// Stop any existing px (local and via service), then start via service.
		t.core.KillPx()
		_, _ = t.request("stop_px", nil, 5*time.Second)
		t.core.Arm()
		_, err := t.request("start_px", startPxData{
			PxPath:  t.core.PxPath(),
			Addr:    t.core.CbAddr(),
			HomeDir: t.core.Home(),
		}, 10*time.Second)
		if err != nil {
			return ConnInfo{}, fmt.Errorf("service start_px: %w", err)
		}
		t.core.MarkStartedBySvc()
		return t.core.Await(60 * time.Second)
	}
	return t.core.RestartDirect()
}

// ShowInstallDialog asks the user whether to install the TUN service.
// Returns "install", "skip" or "cancel".
func (t *TunService) ShowInstallDialog() string {
	result := "cancel"
	dlg := application.Get().Dialog.Question()
	dlg.SetTitle("TUN")
	dlg.SetMessage("TUN mode requires a privileged helper service. Install it now?")
	dlg.AddButton("Install").OnClick(func() { result = "install" })
	dlg.AddButton("Skip").OnClick(func() { result = "skip" })
	dlg.AddButton("Cancel").SetAsDefault().OnClick(func() { result = "cancel" })
	dlg.Show()
	return result
}

func serviceBinaryExists() bool {
	info, err := os.Stat(locate.ServiceBinary())
	return err == nil && !info.IsDir()
}
