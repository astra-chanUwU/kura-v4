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
	"github.com/astra-chanUwU/kura-v4/internal/comments"
	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/jobs"
	"github.com/astra-chanUwU/kura-v4/internal/moderation"
	"github.com/astra-chanUwU/kura-v4/internal/notifications"
	"github.com/astra-chanUwU/kura-v4/internal/reports"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
	"github.com/astra-chanUwU/kura-v4/internal/searches"
	"github.com/astra-chanUwU/kura-v4/internal/site"
	"github.com/astra-chanUwU/kura-v4/internal/storage"
	"github.com/astra-chanUwU/kura-v4/internal/tags"
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
	jobQueue := repository.NewJobs(dao.NewJobs(conn))
	boardSvc := board.NewService(postsRepo, media)
	boardSvc.SetJobQueue(jobQueue, log)
	collectionsSvc := collections.NewService(repository.NewCollections(dao.NewCollections(conn)))
	commentsSvc := comments.NewService(repository.NewComments(dao.NewComments(conn)))
	reportsSvc := reports.NewService(repository.NewReports(dao.NewReports(conn)))
	searchesSvc := searches.NewService(repository.NewSavedSearches(dao.NewSavedSearches(conn)))
	tagsSvc := tags.NewService(repository.NewTagPages(dao.NewTagPages(conn)))
	notesSvc := notifications.NewService(repository.NewNotifications(dao.NewNotifications(conn)))
	moderationSvc := moderation.NewService(repository.NewModeration(dao.NewAccounts(conn), postsRepo, dao.NewAudit(conn)))
	jobsSvc := jobs.NewService(repository.NewJobs(dao.NewJobs(conn)))
	// Fail clearly if migrations have not been applied.
	readyCtx, readyCancel := context.WithTimeout(ctx, 5*time.Second)
	_, err = svc.Get(readyCtx)
	readyCancel()
	if err != nil {
		return errors.New("site schema unavailable; run the migrate command before serve")
	}
	api := NewHTTP(HTTPDeps{Site: svc, Accounts: accountsSvc, Board: boardSvc, Collections: collectionsSvc, Moderation: moderationSvc, Jobs: jobsSvc, Comments: commentsSvc, Reports: reportsSvc, Searches: searchesSvc, Tags: tagsSvc, Notifications: notesSvc, Ping: conn.PingContext, Assets: webui.Files(), Log: log, SecureCookie: strings.HasPrefix(cfg.PublicURL, "https://")})
	workerCtx, stopWorkers := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	sweepDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		jobs.Run(workerCtx, jobQueue, map[string]jobs.Handler{
			board.KindPostDerive: boardSvc.DerivePost,
		}, 2, log)
	}()
	go func() {
		defer close(sweepDone)
		sweepAbandonedUploads(workerCtx, cfg.MediaDir, log)
	}()
	defer func() { stopWorkers(); <-workerDone; <-sweepDone }()
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
