package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/server"
	"github.com/rob121/cannon/internal/version"
)

func main() {
	port := flag.Int("port", 8001, "HTTP listen port")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		if version.Commit != "" {
			fmt.Printf("cannon %s (%s)\n", version.Version, version.Commit)
		} else {
			fmt.Printf("cannon %s\n", version.Version)
		}
		return
	}

	wd, _ := os.Getwd()
	cfg, cfgPath, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	mode := "application"
	if config.NeedsInstall(cfg) {
		mode = "install"
	}
	if cfgPath == "" {
		log.Printf("config: using defaults (no sites.json found, cwd=%s)", wd)
	} else {
		log.Printf("config: loaded %s (install_enabled=%v sites=%d mode=%s version=%s cwd=%s)",
			cfgPath, cfg.InstallEnabled, len(cfg.Sites), mode, version.Version, wd)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	go func() {
		if err := srv.ListenAndServe(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			log.Printf("SIGHUP received, reloading application")
			if err := srv.Reload(); err != nil {
				log.Printf("reload failed: %v", err)
			} else {
				log.Printf("reload complete")
			}
		default:
			log.Printf("signal %v received, exiting", sig)
			os.Exit(0)
		}
	}
}
