package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/jobs"
	"github.com/astra-chanUwU/kura-v4/internal/moderation"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
	"github.com/astra-chanUwU/kura-v4/internal/site"
	"github.com/astra-chanUwU/kura-v4/internal/storage"
	"github.com/astra-chanUwU/kura-v4/internal/webui"
)

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, err := db.Open(connectCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer conn.Close()
	media, err := storage.NewLocal(cfg.MediaDir)
	if err != nil {
		return err
	}
	defer media.Close()
	svc := site.NewService(repository.NewSite(dao.NewSite(conn)))
	accountsSvc, err := accounts.NewService(repository.NewAccounts(dao.NewAccounts(conn)), cfg.PublicURL, media)
	if err != nil {
		return err
	}
	postsRepo := repository.NewPosts(dao.NewPosts(conn))
	boardSvc := board.NewService(postsRepo, media)
	collectionsSvc := collections.NewService(repository.NewCollections(dao.NewCollections(conn)))
	moderationSvc := moderation.NewService(repository.NewModeration(dao.NewAccounts(conn), postsRepo, dao.NewAudit(conn)))
	// Fail clearly if migrations have not been applied.
	readyCtx, readyCancel := context.WithTimeout(ctx, 5*time.Second)
	_, err = svc.Get(readyCtx)
	readyCancel()
	if err != nil {
		return errors.New("site schema unavailable; run the migrate command before serve")
	}
	api := NewHTTP(HTTPDeps{Site: svc, Accounts: accountsSvc, Board: boardSvc, Collections: collectionsSvc, Moderation: moderationSvc, Ping: conn.PingContext, Assets: webui.Files(), Log: log, SecureCookie: strings.HasPrefix(cfg.PublicURL, "https://")})
	workerCtx, stopWorkers := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		// Register feature handlers here as media/import workflows are added.
		jobs.Run(workerCtx, repository.NewJobs(dao.NewJobs(conn)), map[string]jobs.Handler{}, 2, log)
	}()
	defer func() { stopWorkers(); <-workerDone }()
	listenErr := make(chan error, 1)
	go func() { listenErr <- api.Listen(cfg.Address) }()
	log.Info("Kura listening", "address", cfg.Address)
	select {
	case err := <-listenErr:
		return err
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()
		return api.ShutdownWithContext(shutdownCtx)
	}
}
