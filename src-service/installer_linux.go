//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kardianos/service"
)

const (
	serviceFileName = "PrizrakBoxService.service"
	servicePath     = "/etc/systemd/system/" + serviceFileName
)

// getSystemdUnitContent возвращает содержимое systemd unit файла с capabilities
func getSystemdUnitContent(execPath string) string {
	return fmt.Sprintf(`[Unit]
Description=Prizrak Box TUN Service
After=network.target

[Service]
Type=simple
UMask=0077
ExecStart=%s
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=prizrak-box

# Minimal capability set for TUN mode
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE CAP_DAC_READ_SEARCH CAP_DAC_OVERRIDE
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE CAP_DAC_READ_SEARCH CAP_DAC_OVERRIDE

# Capability explanations:
# CAP_NET_ADMIN: Network management (TUN device, routing table)
# CAP_NET_RAW: Raw sockets (ICMP, transparent proxy)
# CAP_NET_BIND_SERVICE: Bind privileged ports (< 1024)
# CAP_DAC_READ_SEARCH: Bypass file read permissions (config files)
# CAP_DAC_OVERRIDE: Bypass file write permissions (log files)

[Install]
WantedBy=multi-user.target
`, execPath)
}

// installServiceLinux устанавливает сервис на Linux с использованием systemd
func installServiceLinux() error {
	// Проверяем права root
	if os.Geteuid() != 0 {
		return fmt.Errorf("installation requires root privileges. Please run with sudo")
	}

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

	// Создаём содержимое unit файла
	unitContent := getSystemdUnitContent(execPath)

	// Записываем unit файл
	if err := os.WriteFile(servicePath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	fmt.Printf("Created systemd unit file: %s\n", servicePath)

	// Перезагружаем systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Println("Reloaded systemd daemon")

	// Включаем автозапуск
	if err := exec.Command("systemctl", "enable", serviceFileName).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Println("Enabled service auto-start")

	return nil
}

// uninstallServiceLinux удаляет сервис на Linux
func uninstallServiceLinux() error {
	// Проверяем права root
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstallation requires root privileges. Please run with sudo")
	}

	// Останавливаем сервис (игнорируем ошибки если не запущен)
	exec.Command("systemctl", "stop", serviceFileName).Run()
	fmt.Println("Stopped service")

	// Отключаем автозапуск (игнорируем ошибки если не включен)
	exec.Command("systemctl", "disable", serviceFileName).Run()
	fmt.Println("Disabled service auto-start")

	// Удаляем unit файл
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	fmt.Printf("Removed service file: %s\n", servicePath)

	// Перезагружаем systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Println("Reloaded systemd daemon")

	return nil
}

// startServiceLinux запускает сервис на Linux
func startServiceLinux() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("starting service requires root privileges. Please run with sudo")
	}

	if err := exec.Command("systemctl", "start", serviceFileName).Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// stopServiceLinux останавливает сервис на Linux
func stopServiceLinux() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("stopping service requires root privileges. Please run with sudo")
	}

	if err := exec.Command("systemctl", "stop", serviceFileName).Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

// statusServiceLinux проверяет статус сервиса на Linux
func statusServiceLinux() (string, error) {
	cmd := exec.Command("systemctl", "is-active", serviceFileName)
	output, err := cmd.Output()

	status := string(output)
	if err != nil {
		// systemctl is-active возвращает ненулевой код если сервис не запущен
		// но это не ошибка, просто возвращаем статус
		return status, nil
	}

	return status, nil
}

// Заглушки для других платформ

func installServiceDarwin() error {
	return fmt.Errorf("darwin is not supported on this build")
}

func uninstallServiceDarwin() error {
	return fmt.Errorf("darwin is not supported on this build")
}

func startServiceDarwin() error {
	return fmt.Errorf("darwin is not supported on this build")
}

func stopServiceDarwin() error {
	return fmt.Errorf("darwin is not supported on this build")
}

func statusServiceDarwin() (string, error) {
	return "", fmt.Errorf("darwin is not supported on this build")
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
