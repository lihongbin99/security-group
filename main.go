package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"security-group/internal/aliyun"
	"security-group/internal/auth"
	"security-group/internal/config"
	"security-group/internal/server"
)

//go:embed web/*
var webContent embed.FS

func main() {
	configPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	aliyunClient, err := aliyun.New(&cfg.Aliyun)
	if err != nil {
		log.Fatalf("create aliyun client: %v", err)
	}

	a := auth.New(cfg)
	srv := server.New(a, aliyunClient, webContent)

	httpServer := &http.Server{
		Addr:    cfg.Server.Listen,
		Handler: srv.Handler(),
	}

	go func() {
		log.Printf("server listening on %s", cfg.Server.Listen)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}
