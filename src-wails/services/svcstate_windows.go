//go:build windows

package services

import (
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// windowsServiceName matches svcConfig.Name in src-service/main.go.
const windowsServiceName = "PrizrakBoxService"

// queryServiceState asks the Service Control Manager about px-service. The GUI
// runs unelevated, so only query rights are requested — the default service
// security descriptor grants those to authenticated users.
func queryServiceState() serviceState {
	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return serviceState{}
	}
	defer windows.CloseServiceHandle(scm)

	name, err := windows.UTF16PtrFromString(windowsServiceName)
	if err != nil {
		return serviceState{}
	}

	// Config access is only needed to detect a disabled start type; if it is
	// refused, fall back to status-only rather than reporting "not installed".
	handle, err := windows.OpenService(scm, name, windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
	queryConfig := err == nil
	if err != nil {
		handle, err = windows.OpenService(scm, name, windows.SERVICE_QUERY_STATUS)
	}
	if err != nil {
		// ERROR_SERVICE_DOES_NOT_EXIST — and anything else we cannot see —
		// means there is nothing worth waiting for.
		return serviceState{}
	}
	defer windows.CloseServiceHandle(handle)

	state := serviceState{Installed: true}
	s := &mgr.Service{Name: windowsServiceName, Handle: handle}
	if status, err := s.Query(); err == nil {
		state.Running = status.State == svc.Running
		state.Starting = status.State == svc.StartPending
	}
	if queryConfig {
		if cfg, err := s.Config(); err == nil {
			state.Disabled = cfg.StartType == mgr.StartDisabled
		}
	}
	return state
}
