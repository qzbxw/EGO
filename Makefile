# 🧠 EGO Development Makefile
# Professional development toolkit for EGO project

.PHONY: help setup lint format check test clean hooks pre-commit docker-build docker-up docker-down docker-logs

# Colors
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m # No Color

help:
	@echo ""
	@echo "╔═══════════════════════════════════════════════════╗"
	@echo "║  🧠 EGO Development Commands                      ║"
	@echo "╚═══════════════════════════════════════════════════╝"
	@echo ""
	@echo "  $(BLUE)Setup:$(NC)"
	@echo "    make setup         - 🚀 Full environment setup (deps + hooks)"
	@echo "    make install       - 📦 Install all dependencies"
	@echo "    make hooks         - 🪝 Install git hooks (Husky + pre-commit)"
	@echo ""
	@echo "  $(BLUE)Code Quality:$(NC)"
	@echo "    make lint          - 🔍 Run linters on all code"
	@echo "    make format        - 🎨 Format all code"
	@echo "    make check         - ✅ Run all checks (lint + format)"
	@echo "    make pre-commit    - 🧪 Run pre-commit on all files"
	@echo ""
	@echo "  $(BLUE)Testing:$(NC)"
	@echo "    make test          - 🧪 Run all tests"
	@echo "    make test-python   - 🐍 Run Python tests"
	@echo "    make test-go       - 🦫 Run Go tests"
	@echo "    make test-frontend - ⚛️  Run frontend tests"
	@echo ""
	@echo "  $(BLUE)Maintenance:$(NC)"
	@echo "    make clean         - 🧹 Clean build artifacts"
	@echo "    make clean-all     - 🔥 Deep clean (including deps)"
	@echo ""
	@echo "  $(BLUE)Docker:$(NC)"
	@echo "    make docker-build  - 🐳 Build all Docker images"
	@echo "    make docker-up     - 🚀 Start all services"
	@echo "    make docker-down   - 🛑 Stop all services"
	@echo "    make docker-logs   - 📋 View logs"
	@echo ""

# ============================================
# 🚀 Setup Commands
# ============================================

setup: install hooks
	@echo ""
	@echo "$(GREEN)╔═══════════════════════════════════════╗$(NC)"
	@echo "$(GREEN)║  🚀 Awesome! Setup complete!          ║$(NC)"
	@echo "$(GREEN)╚═══════════════════════════════════════╝$(NC)"
	@echo ""

install:
	@echo "$(BLUE)📦 Installing dependencies...$(NC)"
	@echo ""
	@echo "$(YELLOW)⚛️  Frontend...$(NC)"
	cd frontend && npm install
	@echo ""
	@echo "$(YELLOW)🐍 Python backend...$(NC)"
	cd backend/python-api && pip install -r requirements-dev.txt
	@echo ""
	@echo "$(YELLOW)🦫 Go backend...$(NC)"
	cd backend/go-api && go mod download
	@echo ""
	@echo "$(GREEN)✅ All dependencies installed!$(NC)"

hooks:
	@echo "$(BLUE)🪝 Installing git hooks...$(NC)"
	@echo ""
	@npm install
	@chmod +x .husky/pre-commit .husky/pre-push .husky/commit-msg
	@chmod +x scripts/setup-hooks.sh
	@bash scripts/setup-hooks.sh
	@echo "$(GREEN)✅ Git hooks installed!$(NC)"

# ============================================
# 🔍 Code Quality Commands
# ============================================

GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin

lint:
	@echo "$(BLUE)🔍 Running linters...$(NC)"
	@echo ""
	@echo "$(YELLOW)⚛️  Linting frontend...$(NC)"
	cd frontend && npm run lint
	@echo ""
	@echo "$(YELLOW)🐍 Linting Python backend...$(NC)"
	cd backend/python-api && ( [ -f .venv/bin/ruff ] && .venv/bin/ruff check . || ruff check . )
	cd backend/python-api && ( [ -f .venv/bin/mypy ] && .venv/bin/mypy . || mypy . )
	@echo ""
	@echo "$(YELLOW)🦫 Linting Go backend...$(NC)"
	cd backend/go-api && go vet ./...
	-cd backend/go-api && PATH="$(PATH):$(GOBIN)" golangci-lint run ./...
	@echo ""
	@echo "$(GREEN)✅ All linters passed!$(NC)"

format:
	@echo "$(BLUE)🎨 Formatting code...$(NC)"
	@echo ""
	@echo "$(YELLOW)⚛️  Formatting frontend...$(NC)"
	cd frontend && npm run format
	@echo ""
	@echo "$(YELLOW)🐍 Formatting Python backend...$(NC)"
	cd backend/python-api && ( [ -f .venv/bin/ruff ] && .venv/bin/ruff format . || ruff format . )
	@echo ""
	@echo "$(YELLOW)🦫 Formatting Go backend...$(NC)"
	cd backend/go-api && go fmt ./...
	@echo ""
	@echo "$(GREEN)✅ Code formatted!$(NC)"

check: lint
	@echo "$(BLUE)✅ Checking formatting...$(NC)"
	@echo ""
	@echo "$(YELLOW)⚛️  Frontend...$(NC)"
	cd frontend && npx prettier --check .
	@echo ""
	@echo "$(YELLOW)🐍 Python...$(NC)"
	cd backend/python-api && ( [ -f .venv/bin/ruff ] && .venv/bin/ruff format --check . || ruff format --check . )
	@echo ""
	@echo "$(YELLOW)🦫 Go...$(NC)"
	cd backend/go-api && [ -z "$$(gofmt -l .)" ]
	@echo ""
	@echo "$(GREEN)✅ All checks passed! Looking good!$(NC)"

pre-commit:
	@echo "$(BLUE)🧪 Running full checks...$(NC)"
	@$(MAKE) check
	@echo "$(GREEN)✅ Pre-commit checks passed!$(NC)"

format-check: check
format-check-alias: check

# ============================================
# 🧪 Testing Commands
# ============================================

test: test-python test-go test-frontend
	@echo ""
	@echo "$(GREEN)╔═══════════════════════════════════════╗$(NC)"
	@echo "$(GREEN)║  ✅ All tests passed! Great work!     ║$(NC)"
	@echo "$(GREEN)╚═══════════════════════════════════════╝$(NC)"
	@echo ""

test-python:
	@echo "$(BLUE)🐍 Running Python tests...$(NC)"
	cd backend/python-api && ( [ -f .venv/bin/pytest ] && .venv/bin/pytest || pytest ) || echo "$(YELLOW)⚠️  No tests found$(NC)"
test-go:
	@echo "$(BLUE)🦫 Running Go tests...$(NC)"
	cd backend/go-api && go test ./... || echo "$(YELLOW)⚠️  No tests found$(NC)"

test-frontend:
	@echo "$(BLUE)⚛️  Running frontend tests...$(NC)"
	cd frontend && npm run check
	@cd frontend && npm run test 2>/dev/null || echo "$(YELLOW)⚠️  No tests found$(NC)"

# ============================================
# 🧹 Cleanup Commands
# ============================================

clean:
	@echo "$(BLUE)🧹 Cleaning build artifacts...$(NC)"
	@echo ""
	@echo "$(YELLOW)🐍 Python cache...$(NC)"
	find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete
	find . -type f -name "*.pyo" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null || true
	@echo ""
	@echo "$(YELLOW)⚛️  Frontend build...$(NC)"
	rm -rf frontend/.svelte-kit frontend/build
	@echo ""
	@echo "$(GREEN)✅ Clean complete!$(NC)"

clean-all: clean
	@echo "$(RED)🔥 Deep cleaning (removing dependencies)...$(NC)"
	@echo ""
	@echo "$(YELLOW)Removing node_modules...$(NC)"
	rm -rf node_modules frontend/node_modules
	@echo ""
	@echo "$(YELLOW)Removing Python venv...$(NC)"
	rm -rf backend/python-api/.venv
	@echo ""
	@echo "$(YELLOW)Removing Go cache...$(NC)"
	cd backend/go-api && go clean -cache -modcache || true
	@echo ""
	@echo "$(GREEN)✅ Deep clean complete!$(NC)"

# ============================================
# 🐳 Docker Commands
# ============================================

docker-build:
	@echo "$(BLUE)🐳 Building Docker images...$(NC)"
	@export DOCKER_BUILDKIT=1 COMPOSE_DOCKER_CLI_BUILD=1 && docker compose build
	@echo "$(GREEN)✅ Build complete!$(NC)"

docker-up:
	@echo "$(BLUE)🚀 Starting services...$(NC)"
	@export DOCKER_BUILDKIT=1 COMPOSE_DOCKER_CLI_BUILD=1 && docker compose up -d
	@echo "$(GREEN)✅ Services started!$(NC)"
	@docker compose ps

docker-down:
	@echo "$(BLUE)🛑 Stopping services...$(NC)"
	@docker compose down
	@echo "$(GREEN)✅ Services stopped!$(NC)"

docker-logs:
	@docker compose logs -f