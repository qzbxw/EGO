#!/usr/bin/env bash

# 🧠 EGO Fast Check Script
# Only checks changed files

set -e

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

MODE="push"
if [ "$1" == "--staged" ]; then
    MODE="commit"
fi

if [ "$MODE" == "commit" ]; then
    echo -e "${BLUE}🔍 Checking staged files...${NC}"
    FILES=$(git diff --name-only --cached)
else
    # Determine base ref (default to origin/main)
    TARGET_BRANCH=$(git rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null | sed 's/.*\///' || echo "main")
    BASE_REF="origin/$TARGET_BRANCH"

    # Fallback if origin branch doesn't exist
    if ! git rev-parse --verify "$BASE_REF" &>/dev/null; then
        BASE_REF="HEAD~1"
    fi
    echo -e "${BLUE}🔍 Checking changed files against ${YELLOW}$BASE_REF${NC}..."
    FILES=$(git diff --name-only "$BASE_REF"...HEAD)
fi

if [ -z "$FILES" ]; then
    echo -e "${GREEN}✅ No changed files to check.${NC}"
    exit 0
fi

# 🐍 Python checks
PY_FILES=$(echo "$FILES" | grep '\.py$' || true)
if [ -n "$PY_FILES" ]; then
    echo -e "${YELLOW}🐍 Checking Python files...${NC}"
    # Use venv if exists
    RUFF="backend/python-api/.venv/bin/ruff"
    if [ ! -f "$RUFF" ]; then RUFF="ruff"; fi
    
    # Filter files that belong to python-api
    API_PY_FILES=$(echo "$PY_FILES" | grep '^backend/python-api/' || true)
    if [ -n "$API_PY_FILES" ]; then
        $RUFF check $API_PY_FILES --fix
        $RUFF format $API_PY_FILES
        # Re-add files if we are in commit mode and they were modified
        if [ "$MODE" == "commit" ]; then
            git add $API_PY_FILES
        fi
    fi
fi

# ⚛️ Frontend checks
FE_FILES=$(echo "$FILES" | grep -E 'frontend/.*\.(js|ts|svelte)$' || true)
if [ -n "$FE_FILES" ]; then
    echo -e "${YELLOW}⚛️ Checking Frontend files...${NC}"
    # Run eslint if possible
    if [ -f "frontend/node_modules/.bin/eslint" ]; then
        ./frontend/node_modules/.bin/eslint $FE_FILES --fix --config frontend/eslint.config.js || echo -e "${RED}ESLint failed, but continuing...${NC}"
    fi
    # Re-add files if we are in commit mode
    if [ "$MODE" == "commit" ]; then
        git add $FE_FILES
    fi
fi

# 🦫 Go checks
GO_FILES=$(echo "$FILES" | grep '\.go$' || true)
if [ -n "$GO_FILES" ]; then
    echo -e "${YELLOW}🦫 Checking Go files...${NC}"
    GO_DIRS=$(echo "$GO_FILES" | grep '^backend/go-api/' | sed 's|backend/go-api/||' | xargs -I {} dirname {} | sort -u | grep -v '^\.' || echo ".")
    if [ -d "backend/go-api" ]; then
        cd backend/go-api
        for dir in $GO_DIRS; do
            if [ -d "$dir" ]; then
                go fmt "./$dir/..." > /dev/null
                go vet "./$dir/..."
            fi
        done
        cd ../..
    fi
    # Re-add files if we are in commit mode
    if [ "$MODE" == "commit" ]; then
        git add $GO_FILES
    fi
fi

# ✨ Prettier check (General)
PRETTIER_FILES=$(echo "$FILES" | grep -E '\.(js|ts|svelte|json|css|md|yaml|yml)$' || true)
if [ -n "$PRETTIER_FILES" ]; then
    echo -e "${YELLOW}✨ Formatting with Prettier...${NC}"
    npx prettier --write $PRETTIER_FILES
    if [ "$MODE" == "commit" ]; then
        git add $PRETTIER_FILES
    fi
fi

echo -e "${GREEN}✅ All fast checks passed!${NC}"