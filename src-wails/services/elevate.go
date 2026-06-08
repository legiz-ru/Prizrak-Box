package services

import (
	"fmt"
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
		if p, err := exec.LookPath("pkexec"); err == nil {
			cmd := exec.Command(p, append([]string{path}, args...)...)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
		if p, err := exec.LookPath("sudo"); err == nil {
			return exec.Command(p, append([]string{path}, args...)...).Run()
		}
		return fmt.Errorf("no elevation helper found (pkexec/sudo)")
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func osaQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
