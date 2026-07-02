#!/usr/bin/env bash
# Record knit UI demo GIF using VHS.
# Requires: vhs (https://github.com/charmbracelet/vhs), go, bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"

BUILD_DIR=".tmp/vhs"
WORK_DIR="$BUILD_DIR/work"
TAPE="scripts/vhs/tapes/knit-demo.tape"

echo "==> Building knit binary..."
mkdir -p "$BUILD_DIR"
go build -o "$BUILD_DIR/knit" ./cmd/knit
echo "    Built $BUILD_DIR/knit"

echo "==> Preparing recording workspace..."
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"

# Copy home fixtures into workspace (.skills-data, .config, .agents, etc.)
if [ -d "scripts/vhs/fixtures/home" ]; then
  cp -r "scripts/vhs/fixtures/home/." "$WORK_DIR/home/"
fi

# Copy project lock to CWD so ListSources() reads it
PROJECT_LOCK="scripts/vhs/fixtures/project/skills-lock.json"
if [ -f "$PROJECT_LOCK" ]; then
  cp "$PROJECT_LOCK" "./skills-lock.json"
  CLEANUP_LOCK=true
else
  CLEANUP_LOCK=false
fi

echo "==> Recording GIF with VHS..."
export PATH="$PWD/scripts/vhs/bin:$PATH"
export HOME="$WORK_DIR/home"
export TERM="${TERM:-xterm-256color}"

vhs "$TAPE"

echo "==> Done! Output: assets/knit-demo.gif"
ls -lh assets/knit-demo.gif

# Cleanup
if [ "$CLEANUP_LOCK" = true ]; then
  rm -f "./skills-lock.json"
fi
