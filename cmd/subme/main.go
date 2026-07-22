package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/subme-app/subme/internal/config"
	"github.com/subme-app/subme/internal/db"
	"github.com/subme-app/subme/internal/scheduler"
	"github.com/subme-app/subme/internal/server"
)

func main() {
	port := flag.Int("port", 9090, "HTTP server port")
	dbPath := flag.String("db", "data/subme.db", "SQLite database path")
	cacheDir := flag.String("cache", "cache", "Cache directory")
	collectorsDir := flag.String("collectors", "collectors", "Collectors directory")
	flag.Parse()

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	if err := os.MkdirAll(*cacheDir, 0755); err != nil {
		log.Fatalf("create cache dir: %v", err)
	}
	if err := os.MkdirAll(*collectorsDir, 0755); err != nil {
		log.Fatalf("create collectors dir: %v", err)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	settings := loadOrInitSettings(database, *port)

	absCacheDir, _ := filepath.Abs(*cacheDir)
	absCollectorsDir, _ := filepath.Abs(*collectorsDir)

	srv, err := server.New(database, absCacheDir, settings, absCollectorsDir)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	handler := srv.Handler()

	hasUsers, _ := database.HasUsers()
	if hasUsers {
		log.Println("initial refresh starting...")
		srv.RefreshAllSync()
		log.Println("initial refresh complete")
	} else {
		log.Println("no admin user registered yet; waiting for first registration")
	}

	sched := scheduler.New(settings.RefreshInterval, func(ctx context.Context) error {
		srv.RefreshAllSync()
		return nil
	})
	sched.Start(context.Background())

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: handler,
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down...")
		sched.Stop()
		httpServer.Shutdown(context.Background())
	}()

	log.Printf("SubMe listening on :%d", *port)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func loadOrInitSettings(database *db.DB, defaultPort int) *config.SystemSettings {
	settings := &config.SystemSettings{
		Port:            defaultPort,
		RefreshInterval: 3600,
		NotifyOn: config.NotifyOn{
			CollectFailure: true,
			RefreshFailure: true,
		},
	}

	if portStr, err := database.GetSetting("port"); err == nil && portStr != "" {
		fmt.Sscanf(portStr, "%d", &settings.Port)
	}
	if intervalStr, err := database.GetSetting("refresh_interval"); err == nil && intervalStr != "" {
		fmt.Sscanf(intervalStr, "%d", &settings.RefreshInterval)
	}
	if proxy, err := database.GetSetting("proxy"); err == nil {
		settings.Proxy = proxy
	}
	if appToken, err := database.GetSetting("wxpusher_app_token"); err == nil {
		settings.WxPusher.AppToken = appToken
	}
	if uids, err := database.GetSetting("wxpusher_uids"); err == nil {
		settings.WxPusher.UIDs = []string{uids}
	}

	return settings
}
