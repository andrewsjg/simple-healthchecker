package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrewsjg/simple-healthchecker/copilot/internal/config"
	"github.com/andrewsjg/simple-healthchecker/copilot/internal/server"
	"github.com/andrewsjg/simple-healthchecker/copilot/internal/state"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file (yaml or toml)")
	addr := flag.String("addr", ":8080", "http listen address")
	interval := flag.Duration("interval", 30*time.Second, "check interval")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	st := state.New(cfg)
	st.SetConfigPath(*cfgPath)
	stop := make(chan struct{})
	st.StartScheduler(*interval, stop)

	srv := server.New(st)
	go func() {
		if err := srv.Start(*addr); err != nil {
			log.Fatalf("http server: %v", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	close(stop)
	_ = srv.Stop()
}
