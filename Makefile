.PHONY: help deps test test-backend test-frontend lint cover up down

help:  ## List the available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'

deps:  ## Install the front-end dependencies
	cd frontend && pnpm install --frozen-lockfile

test: deps  ## Run both suites and regenerate COVERAGE.md
	./scripts/coverage.sh

cover: test  ## The same run, named for the report it produces

test-backend:  ## Go tests only, without coverage — the fast loop while writing code
	cd backend && go test -race ./...

test-frontend: deps  ## Front-end tests only, without coverage — the fast loop
	cd frontend && pnpm test

lint: deps  ## Lint and type-check both sides
	cd backend && go vet ./... && golangci-lint run ./...
	cd frontend && pnpm lint && pnpm typecheck

up:  ## Start the whole system
	docker compose up --build

down:  ## Stop it and remove the volumes
	docker compose down -v
