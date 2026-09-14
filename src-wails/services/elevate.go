package services

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// runElevated runs a binary with elevated privileges and waits for it to
// finish. This mirrors the per-platform elevation logic from
// src-electron/service.ts (PowerShell RunAs / osascript / pkexec).
//
//   - Windows: PowerShell Start-Process -Verb RunAs -Wait (UAC prompt).
//   - macOS:   osascript "do shell script ... with administrator privileges".
//   - Linux:   pkexec, falling back to sudo.
func runElevated(path string, prompt string, args ...string) error {
	switch runtime.GOOS {
	case "windows":
		argList := ""
		if len(args) > 0 {
			quoted := make([]string, len(args))
			for i, a := range args {
				quoted[i] = "'" + a + "'"
			}
			argList = " -ArgumentList " + strings.Join(quoted, ",")
		}
		ps := fmt.Sprintf("Start-Process -FilePath '%s'%s -Verb RunAs -Wait", path, argList)
		return exec.Command("powershell.exe", "-NoProfile", "-Command", ps).Run()

	case "darwin":
		inner := shellQuote(path)
		for _, a := range args {
			inner += " " + shellQuote(a)
		}
		script := fmt.Sprintf("do shell script %s with administrator privileges", osaQuote(inner))
		if prompt != "" {
			script = fmt.Sprintf("do shell script %s with administrator privileges with prompt %s",
				osaQuote(inner), osaQuote(prompt))
		}
		return exec.Command("osascript", "-e", script).Run()

	default: // linux and friends
		// pkexec first (the packages ship a polkit action for it), then the
		// graphical su wrappers older desktops still use. Plain `sudo` is
		// deliberately NOT in this list: with no terminal it can only succeed on a
		// passwordless configuration and otherwise hangs waiting for a password
		// nobody can type. `sudo -A` is tried instead, and only when an askpass
		// helper is actually configured.
		var lastErr error
		tried := false
		for _, helper := range []string{"pkexec", "gksudo", "kdesudo"} {
			p, err := exec.LookPath(helper)
			if err != nil {
				continue
			}
			tried = true
			if err := exec.Command(p, append([]string{path}, args...)...).Run(); err == nil {
				return nil
			} else {
				lastErr = fmt.Errorf("%s: %w", helper, err)
			}
		}
		if askpass := os.Getenv("SUDO_ASKPASS"); askpass != "" {
			if p, err := exec.LookPath("sudo"); err == nil {
				tried = true
				if err := exec.Command(p, append([]string{"-A", path}, args...)...).Run(); err == nil {
					return nil
				} else {
					lastErr = fmt.Errorf("sudo -A: %w", err)
				}
			}
		}
		if !tried {
			return fmt.Errorf("no graphical elevation helper found (pkexec/gksudo/kdesudo)")
		}
		return lastErr
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func osaQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
