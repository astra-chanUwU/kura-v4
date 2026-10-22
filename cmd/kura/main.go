package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/app"
	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/astra-chanUwU/kura-v4/internal/db"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	if len(os.Args) != 2 || (os.Args[1] != "serve" && os.Args[1] != "migrate") {
		log.Error("usage: kura serve | migrate")
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Error("configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if os.Args[1] == "migrate" {
		migrateCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		conn, openErr := db.Open(migrateCtx, cfg.DatabaseURL)
		if openErr != nil {
			log.Error("database connection failed; check configuration and server availability")
			os.Exit(1)
		}
		defer conn.Close()
		err = db.Migrate(migrateCtx, conn)
	} else {
		err = app.Run(ctx, cfg, log)
	}
	if err != nil {
		log.Error("Kura stopped", "error", err)
		os.Exit(1)
	}
}
