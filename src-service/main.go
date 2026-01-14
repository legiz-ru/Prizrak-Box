package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kardianos/service"
	"github.com/legiz-ru/prizrak-box-service/ipc"
	"github.com/legiz-ru/prizrak-box-service/manager"
)

// Версия сервиса
const Version = "1.0.0"

// program реализует интерфейс service.Interface
type program struct {
	server *ipc.Server
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	log.Println("[Service] Starting IPC server...")
	if err := p.server.Start(); err != nil {
		log.Printf("[Service] Failed to start IPC server: %v", err)
	}
}

func (p *program) Stop(s service.Service) error {
	log.Println("[Service] Stopping...")
	if p.server != nil {
		p.server.Stop()
	}
	manager.StopPx()
	return nil
}

func main() {
	// Флаги командной строки
	install := flag.Bool("install", false, "Install the service")
	uninstall := flag.Bool("uninstall", false, "Uninstall the service")
	start := flag.Bool("start", false, "Start the service")
	stop := flag.Bool("stop", false, "Stop the service")
	status := flag.Bool("status", false, "Check service status")
	version := flag.Bool("version", false, "Show version")
	standalone := flag.Bool("standalone", false, "Run in standalone mode (not as service)")

	flag.Parse()

	if *version {
		fmt.Printf("px-service version %s\n", Version)
		return
	}

	// Конфигурация сервиса
	svcConfig := &service.Config{
		Name:        "PrizrakBoxService",
		DisplayName: "Prizrak Box TUN Service",
		Description: "Enables TUN mode for Prizrak Box without requiring administrator privileges for the main application",
	}

	prg := &program{
		server: ipc.NewServer(),
	}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Обработка команд
	if *install {
		err = s.Install()
		if err != nil {
			log.Fatalf("Failed to install service: %v", err)
		}
		fmt.Println("Service installed successfully")

		// Автоматически запускаем сервис после установки
		err = s.Start()
		if err != nil {
			log.Printf("Warning: Failed to start service after install: %v", err)
		} else {
			fmt.Println("Service started successfully")
		}
		return
	}

	if *uninstall {
		// Сначала останавливаем сервис
		_ = s.Stop()

		err = s.Uninstall()
		if err != nil {
			log.Fatalf("Failed to uninstall service: %v", err)
		}
		fmt.Println("Service uninstalled successfully")
		return
	}

	if *start {
		err = s.Start()
		if err != nil {
			log.Fatalf("Failed to start service: %v", err)
		}
		fmt.Println("Service started successfully")
		return
	}

	if *stop {
		err = s.Stop()
		if err != nil {
			log.Fatalf("Failed to stop service: %v", err)
		}
		fmt.Println("Service stopped successfully")
		return
	}

	if *status {
		st, err := s.Status()
		if err != nil {
			fmt.Printf("Service status: unknown (%v)\n", err)
			return
		}
		switch st {
		case service.StatusRunning:
			fmt.Println("Service status: running")
		case service.StatusStopped:
			fmt.Println("Service status: stopped")
		default:
			fmt.Println("Service status: unknown")
		}
		return
	}

	// Standalone режим - запуск без сервиса (для отладки)
	if *standalone {
		log.Println("[Standalone] Starting px-service in standalone mode...")
		prg.run()

		// Ожидание сигнала завершения
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("[Standalone] Shutting down...")
		prg.Stop(nil)
		return
	}

	// Запуск как сервис
	err = s.Run()
	if err != nil {
		log.Fatalf("Service run error: %v", err)
	}
}
