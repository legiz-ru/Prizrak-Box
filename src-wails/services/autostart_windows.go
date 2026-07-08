//go:build windows

// Launch-at-login on Windows via a Task Scheduler logon task instead of the
// HKCU ...\CurrentVersion\Run registry key that the built-in Wails Autostart
// manager writes.
//
// Why: a Run-key start fires at the very peak of the logon rush, racing
// explorer.exe; if the shell's notification area isn't up yet the tray icon
// registers into the void and the app appears to run "invisibly". A scheduled
// task can delay its start (PT15S below) until the desktop has settled —
// the same approach GoclashZ uses.
//
// The task runs as the logged-on user with the interactive token
// (TASK_LOGON_INTERACTIVE_TOKEN) at normal run level — never elevated; TUN
// elevation is px-service's job, not the GUI's.
//
// Enabling/disabling also removes any legacy Run-key entry pointing at this
// executable, migrating installs that enabled autostart before this change.
package services

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows/registry"
)

const (
	autostartTaskName = "Prizrak-Box Startup"
	// autostartDelay keeps the app out of the logon rush (see package comment).
	autostartDelay = "PT15S"

	taskActionExec         = 0 // TASK_ACTION_EXEC
	taskTriggerLogon       = 9 // TASK_TRIGGER_LOGON
	taskCreateOrUpdate     = 6 // TASK_CREATE_OR_UPDATE
	taskLogonInteractive   = 3 // TASK_LOGON_INTERACTIVE_TOKEN
	taskRunLevelLUA        = 0 // TASK_RUNLEVEL_LUA (normal user)
	legacyRunRegistrySubKey = `Software\Microsoft\Windows\CurrentVersion\Run`
)

// clsidTaskScheduler is the Task Scheduler 2.0 service COM class.
var clsidTaskScheduler = ole.NewGUID("{0F87369F-A4E5-4CFC-BD3E-73E6154572DD}")

// withTaskScheduler runs fn with a connected ITaskService IDispatch. It pins
// the goroutine to its OS thread for the duration of the COM usage and
// tolerates COM already being initialised by the Wails main thread in a
// different apartment mode (RPC_E_CHANGED_MODE) or the same mode (S_FALSE).
func withTaskScheduler(fn func(sched *ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	needUninit := true
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		if oleErr, ok := err.(*ole.OleError); ok {
			switch oleErr.Code() {
			case 1: // S_FALSE: already initialised in this mode
				// CoInitializeEx still added a reference; balance it.
			case 0x80010106: // RPC_E_CHANGED_MODE: initialised in another mode
				needUninit = false
			default:
				return fmt.Errorf("autostart: COM init: %w", err)
			}
		} else {
			return fmt.Errorf("autostart: COM init: %w", err)
		}
	}
	if needUninit {
		defer ole.CoUninitialize()
	}

	unk, err := ole.CreateInstance(clsidTaskScheduler, nil)
	if err != nil {
		return fmt.Errorf("autostart: create TaskScheduler instance: %w", err)
	}
	sched, err := unk.QueryInterface(ole.IID_IDispatch)
	unk.Release()
	if err != nil {
		return fmt.Errorf("autostart: query IDispatch: %w", err)
	}
	defer sched.Release()

	if _, err := sched.CallMethod("Connect"); err != nil {
		return fmt.Errorf("autostart: connect to Task Scheduler service: %w", err)
	}
	return fn(sched)
}

// autostartEnabled reports whether the logon task exists and is enabled. A
// legacy Run-key entry from the previous registry-based implementation also
// counts as enabled, so the settings toggle reflects reality on old installs
// (flipping it then migrates to the task and cleans the Run key up).
func autostartEnabled() bool {
	enabled := false
	err := withTaskScheduler(func(sched *ole.IDispatch) error {
		rootV, err := sched.CallMethod("GetFolder", `\`)
		if err != nil {
			return err
		}
		root := rootV.ToIDispatch()
		defer root.Release()

		taskV, err := root.CallMethod("GetTask", autostartTaskName)
		if err != nil {
			return nil // task does not exist
		}
		task := taskV.ToIDispatch()
		defer task.Release()

		if enabledV, err := task.GetProperty("Enabled"); err == nil {
			if b, ok := enabledV.Value().(bool); ok {
				enabled = b
			}
		}
		return nil
	})
	if err == nil && enabled {
		return true
	}
	return hasLegacyRunEntry()
}

// setAutostart registers or removes the logon task and, either way, removes
// any legacy Run-key entry pointing at this executable.
func setAutostart(enabled bool) error {
	defer removeLegacyRunEntries()
	if !enabled {
		return withTaskScheduler(func(sched *ole.IDispatch) error {
			rootV, err := sched.CallMethod("GetFolder", `\`)
			if err != nil {
				return err
			}
			root := rootV.ToIDispatch()
			defer root.Release()
			// The task may not exist; that is fine.
			_, _ = root.CallMethod("DeleteTask", autostartTaskName, 0)
			return nil
		})
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("autostart: resolve executable: %w", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("autostart: absolute executable path: %w", err)
	}

	return withTaskScheduler(func(sched *ole.IDispatch) error {
		defV, err := sched.CallMethod("NewTask", 0)
		if err != nil {
			return fmt.Errorf("autostart: new task definition: %w", err)
		}
		def := defV.ToIDispatch()
		defer def.Release()

		if riV, err := def.GetProperty("RegistrationInfo"); err == nil {
			ri := riV.ToIDispatch()
			_, _ = ri.PutProperty("Description", "Start Prizrak-Box at logon")
			ri.Release()
		}

		if settingsV, err := def.GetProperty("Settings"); err == nil {
			settings := settingsV.ToIDispatch()
			_, _ = settings.PutProperty("DisallowStartIfOnBatteries", false)
			_, _ = settings.PutProperty("StopIfGoingOnBatteries", false)
			_, _ = settings.PutProperty("StartWhenAvailable", true)
			_, _ = settings.PutProperty("ExecutionTimeLimit", "PT0S") // no time limit
			settings.Release()
		}

		actionsV, err := def.GetProperty("Actions")
		if err != nil {
			return fmt.Errorf("autostart: get Actions: %w", err)
		}
		actions := actionsV.ToIDispatch()
		actionV, err := actions.CallMethod("Create", taskActionExec)
		actions.Release()
		if err != nil {
			return fmt.Errorf("autostart: create action: %w", err)
		}
		action := actionV.ToIDispatch()
		_, _ = action.PutProperty("Path", exe)
		_, _ = action.PutProperty("WorkingDirectory", filepath.Dir(exe))
		action.Release()

		triggersV, err := def.GetProperty("Triggers")
		if err != nil {
			return fmt.Errorf("autostart: get Triggers: %w", err)
		}
		triggers := triggersV.ToIDispatch()
		triggerV, err := triggers.CallMethod("Create", taskTriggerLogon)
		triggers.Release()
		if err != nil {
			return fmt.Errorf("autostart: create logon trigger: %w", err)
		}
		trigger := triggerV.ToIDispatch()
		_, _ = trigger.PutProperty("Enabled", true)
		_, _ = trigger.PutProperty("Delay", autostartDelay)
		trigger.Release()

		if principalV, err := def.GetProperty("Principal"); err == nil {
			principal := principalV.ToIDispatch()
			_, _ = principal.PutProperty("LogonType", taskLogonInteractive)
			_, _ = principal.PutProperty("RunLevel", taskRunLevelLUA)
			principal.Release()
		}

		rootV, err := sched.CallMethod("GetFolder", `\`)
		if err != nil {
			return fmt.Errorf("autostart: get root task folder: %w", err)
		}
		root := rootV.ToIDispatch()
		defer root.Release()

		_, err = root.CallMethod("RegisterTaskDefinition",
			autostartTaskName,
			def,
			taskCreateOrUpdate,
			"",  // userId: current user
			nil, // password
			taskLogonInteractive,
		)
		if err != nil {
			return fmt.Errorf("autostart: register task: %w", err)
		}
		return nil
	})
}

// runEntryTargetsThisExe reports whether a Run-key command line points at the
// current executable (with or without quoting).
func runEntryTargetsThisExe(cmd string) bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(cmd), strings.ToLower(exe))
}

// hasLegacyRunEntry reports whether the old registry-based autostart (Wails
// Autostart manager) is still registered for this executable.
func hasLegacyRunEntry() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, legacyRunRegistrySubKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	names, err := key.ReadValueNames(0)
	if err != nil {
		return false
	}
	for _, name := range names {
		if cmd, _, err := key.GetStringValue(name); err == nil && runEntryTargetsThisExe(cmd) {
			return true
		}
	}
	return false
}

// removeLegacyRunEntries deletes any Run-key values pointing at this
// executable, regardless of the value name the old implementation used.
func removeLegacyRunEntries() {
	key, err := registry.OpenKey(registry.CURRENT_USER, legacyRunRegistrySubKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()

	names, err := key.ReadValueNames(0)
	if err != nil {
		return
	}
	for _, name := range names {
		if cmd, _, err := key.GetStringValue(name); err == nil && runEntryTargetsThisExe(cmd) {
			_ = key.DeleteValue(name)
		}
	}
}
