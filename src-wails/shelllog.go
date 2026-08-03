// Shell crash/error logging. The Wails shell runs without a console and with
// LogLevel=Error, so an unhandled panic or an internal framework error would
// otherwise vanish without a trace. PanicHandler/ErrorHandler (wired in
// main.go) append to a small log file next to the other shell preference
// files (see locate.ShellLogFile), which survives data-dir moves.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/legiz-ru/prizrak-box-wails/internal/locate"
)

// shellLogMaxSize caps the log file; when exceeded it is truncated (start
// fresh) so a repeating error can never fill the disk.
const shellLogMaxSize = 1 << 20 // 1 MiB

// appendShellLog writes one timestamped entry to the shell log (best effort).
func appendShellLog(kind, msg string) {
	path := locate.ShellLogFile()
	if info, err := os.Stat(path); err == nil && info.Size() > shellLogMaxSize {
		_ = os.Remove(path)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s [%s] %s\n", time.Now().Format(time.RFC3339), kind, msg)
}

// shellPanicHandler logs panic details; the app still terminates afterwards
// (Wails re-raises), but the cause is preserved for the next session.
func shellPanicHandler(p *application.PanicDetails) {
	appendShellLog("PANIC", fmt.Sprintf("%v\n%s", p.Error, p.FullStackTrace))
	// Also mirror to stderr for dev runs from a terminal.
	fmt.Fprintf(os.Stderr, "panic: %v\n%s\n", p.Error, p.FullStackTrace)
}

// shellErrorHandler logs internal framework errors (window/webview failures
// that are otherwise swallowed at LogLevel=Error with no console attached).
func shellErrorHandler(err error) {
	appendShellLog("ERROR", err.Error())
	fmt.Fprintln(os.Stderr, "wails error:", err)
}
