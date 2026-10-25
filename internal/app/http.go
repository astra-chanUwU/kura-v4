package app

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"mime"
	"path"
	"strings"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/controllers"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/site"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

type HTTPDeps struct {
	Site         *site.Service
	Accounts     *accounts.Service
	Board        *board.Service
	Collections  *collections.Service
	Ping         func(context.Context) error
	Assets       fs.FS
	Log          *slog.Logger
	SecureCookie bool
}

func NewHTTP(deps HTTPDeps) *fiber.App {
	service, ping, assets, log := deps.Site, deps.Ping, deps.Assets, deps.Log
	app := fiber.New(fiber.Config{
		AppName: "Kura", BodyLimit: 64 << 20,
		ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			status, code, message := 500, "internal_error", "Something went wrong. Please try again."
			var httpErr *fiber.Error
			if errors.As(err, &httpErr) {
				status = httpErr.Code
				if status == 404 {
					code, message = "not_found", "Not found."
				} else {
					code, message = "request_error", httpErr.Message
				}
			}
			if status >= 500 {
				log.Error("request failed", "request_id", requestid.FromContext(c), "error", err)
			}
			return c.Status(status).JSON(dto.Error{Code: code, Message: message, RequestID: requestid.FromContext(c)})
		},
	})
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("X-Frame-Options", "DENY")
		return c.Next()
	})
	app.Get("/health/live", func(c fiber.Ctx) error { return c.SendStatus(200) })
	app.Get("/health/ready", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		if err := ping(ctx); err != nil {
			return fiber.NewError(503, "Database is unavailable.")
		}
		return c.SendStatus(200)
	})
	api := app.Group("/api/v1")
	api.Use(func(c fiber.Ctx) error { c.Set("Cache-Control", "no-store"); return c.Next() })
	controllers.NewSite(service).RegisterRoutes(api)
	if deps.Accounts != nil && deps.Board != nil && deps.Collections != nil {
		auth := controllers.NewAuth(deps.Accounts, deps.Collections, deps.Board, deps.SecureCookie)
		api.Use(auth.OptionalAuth)
		auth.RegisterRoutes(api)
		media := app.Group("/media", auth.OptionalAuth)
		auth.RegisterMedia(media)
		posts := controllers.NewPosts(deps.Board, deps.Collections)
		posts.RegisterRoutes(api)
		posts.RegisterMedia(media)
		controllers.NewCollections(deps.Collections).RegisterRoutes(api)
	}
	app.Use("/api", func(c fiber.Ctx) error { return fiber.ErrNotFound })
	app.Use("/health", func(c fiber.Ctx) error { return fiber.ErrNotFound })
	app.Use("/media", func(c fiber.Ctx) error { return fiber.ErrNotFound })
	app.Get("/*", func(c fiber.Ctx) error {
		if assets == nil {
			return c.Type("html").SendString("<!doctype html><title>Kura</title><p>Open the Vite development server (KURA_PUBLIC_URL in .env).</p>")
		}
		name := strings.TrimPrefix(c.Path(), "/")
		if name == "" {
			name = "index.html"
		}
		if !fs.ValidPath(name) {
			return fiber.ErrNotFound
		}
		data, err := fs.ReadFile(assets, name)
		if errors.Is(err, fs.ErrNotExist) && path.Ext(name) == "" {
			name = "index.html"
			data, err = fs.ReadFile(assets, name)
		}
		if err != nil {
			return fiber.ErrNotFound
		}
		c.Set("Content-Type", mime.TypeByExtension(path.Ext(name)))
		if strings.HasPrefix(name, "assets/") {
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Set("Cache-Control", "no-cache")
		}
		return c.Send(data)
	})
	return app
}
