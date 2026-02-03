//go:build windows

package main

import (
	"fmt"

	"github.com/kardianos/service"
)

// installServiceWindows устанавливает сервис на Windows
func installServiceWindows(s service.Service) error {
	if err := s.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}
	return nil
}

// uninstallServiceWindows удаляет сервис на Windows
func uninstallServiceWindows(s service.Service) error {
	// Сначала останавливаем
	s.Stop()

	if err := s.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}
	return nil
}

// startServiceWindows запускает сервис на Windows
func startServiceWindows(s service.Service) error {
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

// stopServiceWindows останавливает сервис на Windows
func stopServiceWindows(s service.Service) error {
	if err := s.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

// statusServiceWindows проверяет статус сервиса на Windows
func statusServiceWindows(s service.Service) (string, error) {
	status, err := s.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get service status: %w", err)
	}

	switch status {
	case service.StatusRunning:
		return "running", nil
	case service.StatusStopped:
		return "stopped", nil
	default:
		return "unknown", nil
	}
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
