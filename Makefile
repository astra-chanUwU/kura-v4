SHELL := /bin/bash
.DEFAULT_GOAL := help
ENV_RUN := node --env-file=.env scripts/run.mjs

.PHONY: help setup db-up db-down dev migrate migration generate generate-check format format-check lint typecheck test test-go test-integration test-web check build compose-check
help:
	@echo "setup | dev | check | build | migrate | migration name=description | generate"
setup:
	@test -f .env || cp .env.example .env
	pnpm install --frozen-lockfile
	go mod download
db-up:
	docker compose up -d --wait postgres
db-down:
	docker compose stop postgres
migrate:
	$(ENV_RUN) go run -tags dev ./cmd/kura migrate
migration:
	@test -n "$(name)" || (echo "Usage: make migration name=description"; exit 1)
	go tool goose -dir internal/db/migrations create "$(name)" sql
dev: db-up migrate
	pnpm dev
generate:
	CGO_ENABLED=0 go tool sqlc generate
	GOFLAGS=-tags=dev go tool mockery
generate-check:
	node scripts/check-generated.mjs
format:
	gofmt -w cmd internal
	pnpm --filter @kura/frontend format:fix
format-check:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)
	pnpm --filter @kura/frontend format
lint:
	go vet -tags dev ./...
	go tool staticcheck -tags dev ./...
	go tool go-arch-lint check
	pnpm --filter @kura/frontend lint
typecheck:
	pnpm --filter @kura/frontend typecheck
test-go:
	go test -race -tags dev ./...
test-integration:
	go test -race -tags 'dev,integration' ./internal/db -count=1
test-web:
	pnpm --filter @kura/frontend test
test: test-go test-integration test-web
compose-check:
	docker compose --env-file .env.example config --quiet
	docker compose --env-file .env.example -f compose.prod.yml config --quiet
check: format-check lint typecheck generate-check test compose-check build
build:
	pnpm --filter @kura/frontend build
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -o bin/kura ./cmd/kura
