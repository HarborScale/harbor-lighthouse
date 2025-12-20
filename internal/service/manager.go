package service

import (
	"github.com/kardianos/service"
)

type program struct {
	runFunc func()
}

func (p *program) Start(s service.Service) error {
	// execute the application logic in a separate goroutine
	// so the service manager doesn't block waiting for Start() to finish.
	go p.runFunc()
	return nil
}

func (p *program) Stop(s service.Service) error {
	// Place any graceful shutdown logic here (e.g. context cancellation)
	// For now, returning nil kills the process immediately, which is fine for an agent.
	return nil
}

// Setup initializes the service configuration with standard naming.
func Setup(logic func()) (service.Service, error) {
	svcConfig := &service.Config{
		Name:        "harbor-lighthouse",           // Lowercase, hyphenated (Standard for Systemd)
		DisplayName: "Harbor Lighthouse",           // Clean name for Windows Service list
		Description: "Harbor Scale Metrics Collector Agent",
		Arguments:   []string{}, // No args needed; main() defaults to running the daemon
	}

	prg := &program{runFunc: logic}
	return service.New(prg, svcConfig)
}
