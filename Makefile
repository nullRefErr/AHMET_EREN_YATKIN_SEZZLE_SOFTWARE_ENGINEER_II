.PHONY: help test test-backend test-frontend lint cover up down

help:  ## List the available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'

test: test-backend test-frontend  ## Run every test

test-backend:  ## Run the Go tests with the race detector
	cd backend && go test -race ./...

test-frontend:  ## Run the frontend tests
	cd frontend && pnpm test

lint:  ## Lint both sides
	cd backend && go vet ./... && golangci-lint run ./...
	cd frontend && pnpm lint

cover:  ## Produce the backend coverage report
	cd backend && go test -race -coverprofile=coverage.out ./...
	cd backend && go tool cover -func=coverage.out
	cd backend && go tool cover -html=coverage.out -o coverage.html

up:  ## Start the whole system
	docker compose up --build

down:  ## Stop it and remove the volumes
	docker compose down -v
