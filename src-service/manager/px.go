package manager

import (
	"log"
	"os"
	"os/exec"
	"sync"
)

var (
	pxCmd   *exec.Cmd
	pxMu    sync.Mutex
	running bool
)

// StartPx запускает px с указанными параметрами
func StartPx(pxPath, addr, homeDir string) error {
	pxMu.Lock()
	defer pxMu.Unlock()

	// Если уже запущен, сначала остановим
	if running && pxCmd != nil && pxCmd.Process != nil {
		log.Println("[Manager] Stopping existing px process...")
		pxCmd.Process.Kill()
		pxCmd.Wait()
		running = false
	}

	// Проверяем существование файла
	if _, err := os.Stat(pxPath); os.IsNotExist(err) {
		return err
	}

	// Формируем аргументы
	args := []string{}
	if addr != "" {
		args = append(args, "-addr="+addr)
	}
	if homeDir != "" {
		args = append(args, "-home="+homeDir)
	}

	log.Printf("[Manager] Starting px: %s %v", pxPath, args)

	pxCmd = exec.Command(pxPath, args...)

	// Перенаправляем вывод в логи
	pxCmd.Stdout = &logWriter{prefix: "[px stdout]"}
	pxCmd.Stderr = &logWriter{prefix: "[px stderr]"}

	if err := pxCmd.Start(); err != nil {
		log.Printf("[Manager] Failed to start px: %v", err)
		return err
	}

	running = true
	log.Printf("[Manager] px started with PID: %d", pxCmd.Process.Pid)

	// Ожидание завершения в горутине
	go func() {
		err := pxCmd.Wait()
		pxMu.Lock()
		running = false
		pxMu.Unlock()
		if err != nil {
			log.Printf("[Manager] px exited with error: %v", err)
		} else {
			log.Println("[Manager] px exited normally")
		}
	}()

	return nil
}

// StopPx останавливает px
func StopPx() {
	pxMu.Lock()
	defer pxMu.Unlock()

	if pxCmd != nil && pxCmd.Process != nil && running {
		log.Println("[Manager] Stopping px...")
		pxCmd.Process.Kill()
		running = false
	}
}

// IsPxRunning проверяет запущен ли px
func IsPxRunning() bool {
	pxMu.Lock()
	defer pxMu.Unlock()
	return running
}

// logWriter пишет в лог с префиксом
type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	log.Printf("%s %s", w.prefix, string(p))
	return len(p), nil
}
