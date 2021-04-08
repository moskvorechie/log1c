package app

import (
	"4d63.com/tz"
	"context"
	"fmt"
	"github.com/moskvorechie/logs"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"gopkg.in/ini.v1"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"
)

type App struct {
	loc      *time.Location
	regex1   *regexp.Regexp
	logger   logs.Log
	cfg      *ini.File
	exit     chan bool
	mess     chan Message
	wg       *sync.WaitGroup
	name     string
	root     string
	instance string
}

func (a *App) Start() {

	var err error

	root, _ := os.Getwd()
	root += string(os.PathSeparator)
	a.root = root

	// Parse config
	a.cfg, err = ini.Load(a.root + "app.ini")
	if err != nil {
		log.Fatal(err)
	}

	// Name
	a.name = a.cfg.Section("main").Key("app").String()

	// Logs
	a.logger, err = logs.New(&logs.Config{
		App:      a.cfg.Section("main").Key("app").String(),
		FilePath: a.root + a.cfg.Section("main").Key("log_path").String(),
		Clear:    true,
	})

	if err != nil {
		log.Fatal(err)
	}

	// Set log level
	a.setLogLevel()

	// Exit on error
	defer func() {
		if err := recover(); err != nil {
			close(a.exit)
			a.logger.Error("Fatal stack: \n" + string(debug.Stack()))
			a.logger.FatalF("Recovered Fatal %v", err)
		}
	}()

	a.pprof()

	// Prepare regex
	a.regex1 = regexp.MustCompile(`(?mis){(\d+),(\w),\s+?{(\w+),(\w+)},(\d+),(\d+),(\d+),(\d+),(\d+),(\w+),"(.*)?",(\d+),\s+?{"(\w)",?(.*)?},"(.*)?",(\d+),(\d+),(\d+),(\d+),(\d+),([\d,]+)?,?\s+?{\d(.*)?}\s+?},?`)

	// Loc
	loc, err := tz.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatal(err)
	}
	a.loc = loc

	// Sync
	a.wg = &sync.WaitGroup{}

	// Send
	a.exit = make(chan bool)
	a.mess = make(chan Message, 100)

	// Start watch each log dir in separate goroutine
	section := a.cfg.Section("logs")
	for k, flog := range section.Keys() {

		// Run Sender
		a.wg.Add(1)
		var s Sender
		s.logger = a.logger
		s.mess = a.mess
		s.cfg = a.cfg
		s.wg = a.wg
		go s.Run(k)

		// Run DirReader
		a.wg.Add(1)
		var r DirReader
		r.app = a
		r.wg = a.wg
		r.exit = a.exit
		r.cfg = a.cfg
		r.logger = a.logger
		r.name = flog.Name()
		r.path = flog.String()
		go r.Run()
	}

	// Server for metrics
	http.Handle("/metrics", promhttp.Handler())
	addr := "0.0.0.0:54545"
	if a.name == "test" {
		addr = "0.0.0.0:80"
	}
	server := &http.Server{Addr: addr}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.FatalError(err)
		}
	}()

	// Sleep for defer close
	<-a.exit

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		a.logger.FatalError(err)
	}
}

func (a *App) Stop() error {
	close(a.exit)
	close(a.mess)
	a.wg.Wait()
	return nil
}

func (a *App) setLogLevel() {
	switch a.cfg.Section("main").Key("log_level").String() {
	case "info":
		a.logger.SetCustomLogger(a.logger.Logger().Level(zerolog.InfoLevel))
	case "warning":
		a.logger.SetCustomLogger(a.logger.Logger().Level(zerolog.WarnLevel))
	case "error":
		a.logger.SetCustomLogger(a.logger.Logger().Level(zerolog.ErrorLevel))
	default:
		a.logger.SetCustomLogger(a.logger.Logger().Level(zerolog.DebugLevel))
	}
}

func (a *App) pprof() {
	// Memory pprof
	go func() {
		time.Sleep(5 * time.Minute)
		for {
			os.Mkdir("pprof", 0755)
			f, err := os.OpenFile(fmt.Sprintf("%s\\pprof\\mem%s.prof", a.root, time.Now().Format("15")), os.O_RDWR|os.O_CREATE, 0755)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
			f.Close()
			time.Sleep(1 * time.Hour)
		}
	}()

	// CPU pprof
	go func() {
		time.Sleep(5 * time.Minute)
		for {
			os.Mkdir("pprof", 0755)
			f, err := os.OpenFile(fmt.Sprintf("%s\\pprof\\cpu%s.prof", a.root, time.Now().Format("15")), os.O_RDWR|os.O_CREATE, 0755)
			if err != nil {
				log.Fatal("could not create cpu profile: ", err)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Fatal("could not start CPU profile: ", err)
			}
			time.Sleep(1 * time.Minute)
			pprof.StopCPUProfile()
			f.Close()
			time.Sleep(1 * time.Hour)
		}
	}()
}
