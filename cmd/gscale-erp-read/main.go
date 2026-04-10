package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gscale_erp_read/internal/appconfig"
	"gscale_erp_read/internal/httpapi"
	"gscale_erp_read/internal/store"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cfg, err := appconfig.LoadFromEnv()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("db ping failed: %v", err)
	}

	svc := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewHandler(store.New(db)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("gscale-erp-read listening on %s for site %s", cfg.Addr, cfg.SiteName)
		if err := svc.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := svc.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown warning: %v", err)
	}
}
