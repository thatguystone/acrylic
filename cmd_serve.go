package main

import (
	"log"
	"net"
	"net/http"
	"path/filepath"
)

func cmdServe(cfgFile string) {
	fnot := fwatch()
	for {
		cmdServeRun(cfgFile, fnot)
	}
}

func cmdServeRun(cfgFile string, fnot *fnotify) {
	cfg, err := loadConfig(cfgFile)
	if err != nil {
		log.Fatal(err)
	}

	err = cmdBuild(cfg)
	if err != nil {
		if !cfg.Server.NoWatch {
			log.Fatal(err)
		}
	}

	lsock, err := net.Listen("tcp", cfg.Server.ListenAddr)
	if err != nil {
		log.Fatal(err)
	}

	defer lsock.Close()

	server := &http.Server{
		Handler: http.FileServer(http.Dir(cfg.getPublicDir())),
	}
	go server.Serve(lsock)

	log.Printf("Listening for connections on %s...", cfg.Server.ListenAddr)

	if cfg.Server.NoWatch {
		fnot.setDir("")
		select {}
	}

	fnot.setDir(filepath.Dir(cfg.path))
	for {
		select {
		case <-fnot.rebuild:
			log.Println("Change detected, rebuilding...")
			return
		}
	}
}
