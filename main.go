package main

import (
	"github.com/moskvorechie/go-svc/svc"
	"github.com/moskvorechie/log1c/app"
	"log"
)

type program struct {
	env svc.Environment
	svr *app.App
}

func main() {
	prg := program{
		svr: &app.App{},
	}
	if err := svc.Run(&prg); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	p.env = env
	log.Printf("is win service? %v\n", env.IsWindowsService())
	return nil
}

func (p *program) Start() error {
	go p.svr.Start()
	log.Print("Started.\n")
	return nil
}

func (p *program) Stop() error {
	log.Print("Stopping...\n")
	if err := p.svr.Stop(); err != nil {
		return err
	}
	log.Print("Stopped.\n")
	return nil
}
