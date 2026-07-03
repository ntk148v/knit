#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOME_DIR="${1:-$ROOT/.tmp/vhs/check/home}"

rm -rf "$HOME_DIR"
mkdir -p "$HOME_DIR"
cp -R "$ROOT/scripts/vhs/fixtures/home/." "$HOME_DIR/"

export HOME="$HOME_DIR"
export PATH="$ROOT/scripts/vhs/bin:$PATH"

require_contains() {
  local haystack="$1" needle="$2" label="$3"
  case "$haystack" in
    *"$needle"*) ;;
    *)
      printf 'fixture check failed: %s\nmissing: %s\noutput:\n%s\n' "$label" "$needle" "$haystack" >&2
      exit 1
      ;;
  esac
}

installed="$(npx skills list --json)"
require_contains "$installed" '"name": "caveman"' "installed list includes caveman"
require_contains "$installed" '"name": "frontend-design"' "installed list includes frontend-design"
require_contains "$installed" '"name": "code-reviewer"' "installed list includes code-reviewer"
require_contains "$installed" '"scope": "project"' "installed list includes project scope"

global="$(npx skills list -g --json)"
require_contains "$global" '[' "global list returns JSON"

find_review="$(npx skills find review)"
require_contains "$find_review" 'code-reviewer' "discover search returns code-reviewer"
require_contains "$find_review" 'code-review' "discover search returns code-review"

source_list="$(npx skills add ntk148v/skills --list)"
require_contains "$source_list" 'caveman' "source list includes caveman"
require_contains "$source_list" 'code-reviewer' "source list includes code-reviewer"
require_contains "$source_list" 'uv' "source list includes uv"

for skill in caveman code-reviewer frontend-design; do
  test -f "$HOME_DIR/skills-data/$skill/SKILL.md" || {
    printf 'fixture check failed: missing %s/SKILL.md\n' "$skill" >&2
    exit 1
  }
done

printf 'fixture check passed\n'
