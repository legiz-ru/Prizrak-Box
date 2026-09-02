package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/legiz-ru/prizrak-box-wails/internal/locate"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// serviceStartupBudget bounds how long startup waits for px-service to
	// answer its IPC endpoint while the OS reports it as starting — the boot
	// race after a reboot. It is deliberately short: falling back to a direct
	// px spawn is always possible, and the window is revealed independently of
	// this wait (see main.go), so the budget is never user-visible dead time.
	serviceStartupBudget = 5 * time.Second
	serviceStartupPoll   = 250 * time.Millisecond
	// serviceStateRecheck is how often the OS-level service state is
	// re-examined during the wait, so a service that never gets going (crashed
	// or was stopped meanwhile) aborts it instead of burning the whole budget.
	serviceStateRecheck = 1 * time.Second
)

// serviceState is the OS-level registration state of px-service as reported by
// the platform service manager (SCM / systemd / launchd). It answers the only
// question that matters at startup: can waiting for this service pay off?
type serviceState struct {
	Installed bool
	Running   bool // the manager reports the service as running
	Starting  bool // start pending / activating
	Disabled  bool
}

// mayComeUp reports whether the service can plausibly answer its IPC endpoint
// shortly, i.e. whether waiting for it is worthwhile at all.
func (s serviceState) mayComeUp() bool {
	return s.Installed && !s.Disabled && (s.Running || s.Starting)
}

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
//
// "Installed" means registered with the OS service manager — NOT merely that
// the bundled px-service binary is present on disk, which it always is. The
// frontend uses this to decide whether to offer installing the service and
// whether a restart-through-the-service recovery is worth attempting
// (MyProxy.vue), so the file-presence answer made it skip both.
func (t *TunService) GetStatus() ServiceStatus {
	running := t.ping()
	// A service answering its IPC endpoint is installed by definition, even if
	// the state query itself was refused.
	status := ServiceStatus{Installed: queryServiceState().Installed || running}
	if !running {
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
	// launchd/systemd/SCM return as soon as the job is registered; the daemon
	// still has to start and bind its IPC socket.
	if t.waitForService(15*time.Second, 300*time.Millisecond) {
		return true
	}
	application.Get().Logger.Error("service installed but its IPC endpoint never became reachable")
	return false
}

// Uninstall removes the px-service (requires elevation).
//
// px is stopped through the service FIRST: while the service is installed it
// owns the running px (spawned as root/LocalSystem, so the shell cannot signal
// it). Removing the service without stopping px left an orphaned privileged
// process holding the control port and the TUN device, and the freshly spawned
// unprivileged px then had to fall back to a random port.
func (t *TunService) Uninstall() bool {
	if t.ping() {
		// Ask px to exit on its own first: the service's stop_px is a hard kill,
		// which would skip px's cleanup and strand the system proxy settings.
		t.core.RequestExit()
		if _, err := t.request("stop_px", nil, 5*time.Second); err != nil {
			application.Get().Logger.Warn("stopping px through the service before uninstall failed", "error", err)
		}
	}

	bin := locate.ServiceBinary()
	if err := runElevated(bin, "", "-uninstall"); err != nil {
		application.Get().Logger.Error("service uninstall failed", "error", err)
		return false
	}

	// Wait for the daemon to actually go away so a RestartBackend right after
	// this does not get routed through the service that is still shutting down.
	for i := 0; i < 20; i++ {
		if !t.ping() {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.core.ClearStartedBySvc()
	return true
}

// startViaService stops any running px (local or service-managed) and relaunches
// it through the elevated px-service, returning the fresh connection info. The
// service must already be reachable (ping) before calling. Because the service
// runs as LocalSystem/root, the px it spawns is privileged and can bring up the
// TUN adapter without the GUI itself being elevated.
func (t *TunService) startViaService() (ConnInfo, error) {
	t.core.KillPx()
	_, _ = t.request("stop_px", nil, 5*time.Second)
	t.core.Arm()
	if _, err := t.request("start_px", startPxData{
		PxPath:  t.core.PxPath(),
		Addr:    t.core.CbAddr(),
		HomeDir: t.core.Home(),
	}, 10*time.Second); err != nil {
		return ConnInfo{}, fmt.Errorf("service start_px: %w", err)
	}
	t.core.MarkStartedBySvc()
	return t.core.Await(60 * time.Second)
}

// waitForService polls the px-service until it answers ping or the budget runs
// out. It deliberately does NOT consult the service manager: right after an
// install the job can already be registered and starting while the manager
// still reports nothing, so bailing out on its answer would fail a perfectly
// good install.
func (t *TunService) waitForService(total, step time.Duration) bool {
	deadline := time.Now().Add(total)
	for {
		if t.ping() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(step)
	}
}

// waitForServiceStartup is the startup variant: it rides out the boot race
// where the autostarted app outruns the still-starting service, but gives up as
// soon as the OS stops reporting the service as on its way up, since the rest
// of the budget cannot change the outcome then.
func (t *TunService) waitForServiceStartup(total, step time.Duration) bool {
	deadline := time.Now().Add(total)
	nextStateCheck := time.Now().Add(serviceStateRecheck)
	for {
		if t.ping() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		if time.Now().After(nextStateCheck) {
			if !queryServiceState().mayComeUp() {
				return false
			}
			nextStateCheck = time.Now().Add(serviceStateRecheck)
		}
		time.Sleep(step)
	}
}

// StartBackend is the application startup entry point for launching px. When the
// privileged px-service is reachable it routes px through the service so TUN can
// work without running the GUI as administrator — mirroring src-electron
// admin.ts. If TUN was enabled last session and the OS reports the service as
// starting, it waits briefly for it, covering the boot race where the
// autostarted (non-elevated) app launches px before the service has finished
// starting. If the service never comes up (or isn't installed) px is spawned
// directly.
func (t *TunService) StartBackend() (ConnInfo, error) {
	if err := t.core.ensureCallbackServer(); err != nil {
		return ConnInfo{}, fmt.Errorf("callback server: %w", err)
	}

	reachable := t.ping()
	if !reachable && locate.TunDesired() {
		// Wait only when the service manager says the service exists and is on
		// its way up — that is the boot race the budget exists for.
		//
		// This used to be gated on the px-service *file* being present, which
		// is always true (it ships with the app), so any machine with the TUN
		// toggle on but no installed/running service burned the full budget on
		// every single launch and then fell back to a direct spawn anyway.
		if queryServiceState().mayComeUp() {
			reachable = t.waitForServiceStartup(serviceStartupBudget, serviceStartupPoll)
		}
	}

	if reachable {
		if info, err := t.startViaService(); err == nil {
			return info, nil
		} else {
			application.Get().Logger.Warn("start px via service failed; falling back to direct spawn", "error", err)
		}
	} else if locate.TunDesired() {
		application.Get().Logger.Warn("px-service not reachable; starting px directly (TUN stays off until the service is available)")
	}
	return t.core.RestartDirect()
}

// RestartBackend restarts px. If the service is running, px is (re)started
// through the elevated service so it can manage TUN; otherwise it is spawned
// directly. Returns the fresh connection info for the frontend.
func (t *TunService) RestartBackend() (ConnInfo, error) {
	if t.ping() {
		return t.startViaService()
	}
	t.core.ClearStartedBySvc()
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
