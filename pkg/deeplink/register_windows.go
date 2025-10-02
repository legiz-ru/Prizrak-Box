//go:build windows

package deeplink

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func registerProtocol(scheme, displayName string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("normalise executable path: %w", err)
	}

	keyPath := `Software\\Classes\\` + scheme

	key, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("create protocol key: %w", err)
	}
	defer key.Close()

	if err := key.SetStringValue("", fmt.Sprintf("URL:%s", displayName)); err != nil {
		return fmt.Errorf("set protocol display name: %w", err)
	}

	if err := key.SetStringValue("URL Protocol", ""); err != nil {
		return fmt.Errorf("mark scheme as url protocol: %w", err)
	}

	if iconKey, _, iconErr := registry.CreateKey(registry.CURRENT_USER, keyPath+`\\DefaultIcon`, registry.SET_VALUE); iconErr == nil {
		_ = iconKey.SetStringValue("", exePath)
		iconKey.Close()
	}

	commandKey, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath+`\\shell\\open\\command`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("create open command key: %w", err)
	}
	defer commandKey.Close()

	command := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	if err := commandKey.SetStringValue("", command); err != nil {
		return fmt.Errorf("set open command: %w", err)
	}

	return nil
}
