//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kardianos/service"
)

const (
	serviceName     = "com.prizrak-box.service"
	plistPath       = "/Library/LaunchDaemons/" + serviceName + ".plist"
	logPath         = "/var/log/prizrak-box-service.log"
	errorLogPath    = "/var/log/prizrak-box-service-error.log"
)

// getLaunchdPlistContent возвращает содержимое launchd plist файла
func getLaunchdPlistContent(execPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
    "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>UserName</key>
    <string>root</string>
</dict>
</plist>`, serviceName, execPath, logPath, errorLogPath)
}

// executeWithPrivilege выполняет команду с привилегиями администратора используя AppleScript
func executeWithPrivilege(script string) error {
	// Экранируем кавычки в скрипте
	escapedScript := script
	for i := 0; i < len(script); i++ {
		if script[i] == '"' {
			escapedScript = escapedScript[:i] + "\\" + escapedScript[i:]
			i++
		}
	}

	command := fmt.Sprintf(`do shell script "%s" with administrator privileges`, escapedScript)
	cmd := exec.Command("osascript", "-e", command)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to execute with privileges: %w, output: %s", err, string(output))
	}

	return nil
}

// installServiceDarwin устанавливает сервис на macOS с использованием launchd
func installServiceDarwin() error {
	// Получаем путь к текущему исполняемому файлу
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Разрешаем symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Создаём содержимое plist файла
	plistContent := getLaunchdPlistContent(execPath)

	// Создаём временный файл
	tmpFile, err := os.CreateTemp("", "prizrak-box-service-*.plist")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Записываем содержимое во временный файл
	if _, err := tmpFile.WriteString(plistContent); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	fmt.Printf("Created temporary plist file: %s\n", tmpPath)

	// Проверяем запущены ли мы уже с правами root (через Electron osascript)
	isRoot := os.Geteuid() == 0

	if isRoot {
		// Уже root - выполняем команды напрямую
		if err := exec.Command("cp", tmpPath, plistPath).Run(); err != nil {
			return fmt.Errorf("failed to copy plist: %w", err)
		}
		if err := exec.Command("chmod", "644", plistPath).Run(); err != nil {
			return fmt.Errorf("failed to chmod plist: %w", err)
		}
		if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
			return fmt.Errorf("failed to load service: %w", err)
		}
	} else {
		// Не root - используем osascript для запроса прав
		installScript := fmt.Sprintf("cp '%s' '%s' && chmod 644 '%s' && launchctl load '%s'",
			tmpPath, plistPath, plistPath, plistPath)

		fmt.Println("Requesting administrator privileges...")
		if err := executeWithPrivilege(installScript); err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}
	}

	fmt.Printf("Installed and loaded service: %s\n", plistPath)

	return nil
}

// uninstallServiceDarwin удаляет сервис на macOS
func uninstallServiceDarwin() error {
	isRoot := os.Geteuid() == 0

	if isRoot {
		// Уже root - выполняем команды напрямую
		// Останавливаем сервис (игнорируем ошибки)
		exec.Command("launchctl", "unload", plistPath).Run()

		// Удаляем plist файл
		if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove plist: %w", err)
		}
	} else {
		// Не root - используем osascript
		fmt.Println("Requesting administrator privileges...")
		uninstallScript := fmt.Sprintf("launchctl unload '%s' 2>/dev/null || true && rm -f '%s'",
			plistPath, plistPath)

		if err := executeWithPrivilege(uninstallScript); err != nil {
			return fmt.Errorf("failed to uninstall service: %w", err)
		}
	}

	fmt.Printf("Uninstalled service: %s\n", plistPath)

	return nil
}

// startServiceDarwin запускает сервис на macOS
func startServiceDarwin() error {
	isRoot := os.Geteuid() == 0

	if isRoot {
		// Уже root - выполняем команду напрямую
		if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
	} else {
		// Не root - используем osascript
		fmt.Println("Requesting administrator privileges...")
		startScript := fmt.Sprintf("launchctl load '%s'", plistPath)

		if err := executeWithPrivilege(startScript); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
	}

	return nil
}

// stopServiceDarwin останавливает сервис на macOS
func stopServiceDarwin() error {
	isRoot := os.Geteuid() == 0

	if isRoot {
		// Уже root - выполняем команду напрямую
		if err := exec.Command("launchctl", "unload", plistPath).Run(); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
	} else {
		// Не root - используем osascript
		fmt.Println("Requesting administrator privileges...")
		stopScript := fmt.Sprintf("launchctl unload '%s'", plistPath)

		if err := executeWithPrivilege(stopScript); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
	}

	return nil
}

// statusServiceDarwin проверяет статус сервиса на macOS
func statusServiceDarwin() (string, error) {
	// Проверяем существование plist файла
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return "not installed", nil
	}

	// Проверяем загружен ли сервис
	cmd := exec.Command("launchctl", "list", serviceName)
	if err := cmd.Run(); err != nil {
		return "stopped", nil
	}

	return "running", nil
}

// Заглушки для других платформ

func installServiceLinux() error {
	return fmt.Errorf("linux is not supported on this build")
}

func uninstallServiceLinux() error {
	return fmt.Errorf("linux is not supported on this build")
}

func startServiceLinux() error {
	return fmt.Errorf("linux is not supported on this build")
}

func stopServiceLinux() error {
	return fmt.Errorf("linux is not supported on this build")
}

func statusServiceLinux() (string, error) {
	return "", fmt.Errorf("linux is not supported on this build")
}

func installServiceWindows(s service.Service) error {
	return fmt.Errorf("windows is not supported on this build")
}

func uninstallServiceWindows(s service.Service) error {
	return fmt.Errorf("windows is not supported on this build")
}

func startServiceWindows(s service.Service) error {
	return fmt.Errorf("windows is not supported on this build")
}

func stopServiceWindows(s service.Service) error {
	return fmt.Errorf("windows is not supported on this build")
}

func statusServiceWindows(s service.Service) (string, error) {
	return "", fmt.Errorf("windows is not supported on this build")
}
