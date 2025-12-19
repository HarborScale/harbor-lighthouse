package service

import (
	"github.com/kardianos/service"
)

type program struct{
	runFunc func()
}

func (p *program) Start(s service.Service) error {
	go p.runFunc()
	return nil
}

func (p *program) Stop(s service.Service) error {
	// Clean shutdown logic here if needed
	return nil
}

func Setup(logic func()) (service.Service, error) {
	svcConfig := &service.Config{
		Name:        "HarborLighthouse",
		DisplayName: "Harbor Scale Lighthouse Agent",
		Description: "Telemetry collection agent for Harbor Scale.",
		Arguments:   []string{},
	}

	prg := &program{runFunc: logic}
	return service.New(prg, svcConfig)
}
