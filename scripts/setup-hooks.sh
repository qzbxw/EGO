#!/usr/bin/env bash

# 🧠 EGO Hooks Setup Script
set -e

echo ""
echo "╔═══════════════════════════════════════════════════╗"
echo "║                                                   ║"
echo "║     🧠 EGO Development Environment Setup          ║"
echo "║                                                   ║"
echo "╚═══════════════════════════════════════════════════╝"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_step() {
  echo -e "${BLUE}▶${NC} $1"
}

print_success() {
  echo -e "${GREEN}✅${NC} $1"
}

print_warning() {
  echo -e "${YELLOW}⚠️${NC}  $1"
}

print_error() {
  echo -e "${RED}❌${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "package.json" ]; then
  print_error "Must run from project root!"
  exit 1
fi

# Install npm dependencies
print_step "Installing npm dependencies..."
npm install
print_success "NPM dependencies installed"
echo ""

# Install Python dependencies
print_step "Setting up Python environment..."
if [ -d "backend/python-api" ]; then
  cd backend/python-api

  # Check if venv exists
  if [ ! -d ".venv" ]; then
    print_step "Creating Python virtual environment..."
    python3 -m venv .venv
  fi

  # Activate and install
  . .venv/bin/activate
  pip install -r requirements-dev.txt 2>/dev/null || print_warning "requirements-dev.txt not found"
  deactivate
  cd ../..
  print_success "Python environment ready"
else
  print_warning "Python backend not found"
fi
echo ""

# Install Go dependencies
print_step "Setting up Go environment..."
if [ -d "backend/go-api" ]; then
  cd backend/go-api
  go mod download
  cd ../..
  print_success "Go dependencies installed"
else
  print_warning "Go backend not found"
fi
echo ""

# Install pre-commit
print_step "Installing pre-commit..."
if command -v pre-commit &> /dev/null; then
  pre-commit install
  pre-commit install --hook-type commit-msg
  print_success "Pre-commit hooks installed"
else
  print_warning "pre-commit not found. Install with: pip install pre-commit"
  print_warning "Run 'pre-commit install' manually after installing"
fi
echo ""

# Make husky hooks executable
print_step "Setting up Husky hooks..."
chmod +x .husky/pre-commit
chmod +x .husky/pre-push
chmod +x .husky/commit-msg
print_success "Husky hooks configured"
echo ""

# Install frontend dependencies
print_step "Installing frontend dependencies..."
if [ -d "frontend" ]; then
  cd frontend
  npm install
  cd ..
  print_success "Frontend dependencies installed"
else
  print_warning "Frontend directory not found"
fi
echo ""

echo "╔═══════════════════════════════════════════════════╗"
echo "║                                                   ║"
echo "║  🚀 Awesome! Environment ready!                   ║"
echo "║                                                   ║"
echo "║  Git hooks installed:                             ║"
echo "║  • pre-commit:  Lint & format staged files        ║"
echo "║  • pre-push:    Run tests before push             ║"
echo "║  • commit-msg:  Validate commit messages          ║"
echo "║                                                   ║"
echo "║  Quick commands:                                  ║"
echo "║  • make lint    - Run all linters                 ║"
echo "║  • make format  - Format all code                 ║"
echo "║  • make test    - Run all tests                   ║"
echo "║                                                   ║"
echo "╚═══════════════════════════════════════════════════╝"
echo ""
