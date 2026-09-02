//go:build !windows

package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestCoreStartHandshake validates the px <-> shell contract headlessly using
// a fake px: the shell must run a callback server that captures port/secret
// from GET /pxStore and keeps px alive via GET /pxAlive. No GUI required.
func TestCoreStartHandshake(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl not available")
	}

	dir := t.TempDir()
	fakePx := filepath.Join(dir, "px")
	script := `#!/usr/bin/env bash
set -e
addr=""
for arg in "$@"; do
  case "$arg" in
    -addr=*) addr="${arg#-addr=}" ;;
  esac
done
# Report our (fake) control port + secret to the shell, like real px does.
curl -s "http://${addr}/pxStore?port=12345&secret=testsecret" >/dev/null
# Keep-alive loop: exit when the shell stops answering "alive".
while true; do
  body=$(curl -s "http://${addr}/pxAlive" || true)
  [ "$body" = "alive" ] || exit 0
  sleep 1
done
`
	if err := os.WriteFile(fakePx, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRIZRAK_PX_BIN", fakePx)
	t.Setenv("PRIZRAK_HOME", dir)

	core := NewCoreService()
	defer core.Stop()

	done := make(chan struct {
		info ConnInfo
		err  error
	}, 1)
	go func() {
		info, err := core.Start()
		done <- struct {
			info ConnInfo
			err  error
		}{info, err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("Start() error: %v", res.err)
		}
		if res.info.Port != 12345 {
			t.Errorf("port = %d, want 12345", res.info.Port)
		}
		if res.info.Secret != "testsecret" {
			t.Errorf("secret = %q, want %q", res.info.Secret, "testsecret")
		}
		if res.info.Host != "127.0.0.1" {
			t.Errorf("host = %q, want 127.0.0.1", res.info.Host)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for px callback handshake")
	}

	// GetConnInfo should now return immediately with the same data.
	info, err := core.GetConnInfo()
	if err != nil || info.Port != 12345 {
		t.Fatalf("GetConnInfo = %+v, err=%v", info, err)
	}
}
