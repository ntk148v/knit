<div align="center">

<img src="assets/logo-transparent.png" alt="knit" width="320">

# KNIT

**A terminal UI for agent skills.**

[![CI](https://img.shields.io/github/actions/workflow/status/ntk148v/knit/ci.yml?branch=master&style=flat-square&label=CI&labelColor=0f172a&color=3dbbff)](https://github.com/ntk148v/knit/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/ntk148v/knit?style=flat-square)](https://goreportcard.com/report/github.com/ntk148v/knit) [![Release](https://img.shields.io/github/v/release/ntk148v/knit?style=flat-square&labelColor=0f172a&color=ff79f2)](https://github.com/ntk148v/knit/releases/latest) [![Go](https://img.shields.io/github/go-mod/go-version/ntk148v/knit?style=flat-square&logo=go&logoColor=white&label=Go&labelColor=0f172a&color=3dbbff)](go.mod) [![License](https://img.shields.io/badge/license-Apache-b253f5?style=flat-square&labelColor=0f172a)](LICENSE) [![Stars](https://img.shields.io/github/stars/ntk148v/knit?style=flat-square&labelColor=0f172a&color=556bf4)](https://github.com/ntk148v/knit/stargazers)

</div>

## Overview

`knit` is a keyboard-first terminal UI for managing [`npx skills`](https://github.com/vercel-labs/skills) skills.

It keeps the upstream CLI as the source of truth, then adds the part terminals are good at: fast browsing, focused detail views, visible actions, and no command memorization.

<p align="center">
  <img src="assets/knit-demo.gif" alt="knit UI demo" width="900">
</p>

## Why knit?

`npx skills` is already good. It is simple, scriptable, and the right tool when you know the exact command to run.

`knit` exists for the other half of the workflow: browsing installed skills, discovering new ones, checking source metadata, and managing changes without repeatedly typing long commands. A TUI makes that loop faster because the current state stays on screen.

## Features

- Browse project and global skills in one place.
- Search installed skills, discovered skills, and sources.
- Inspect focused skill and source detail views with `Esc` back navigation.
- Install, update, uninstall, and prune skills from the TUI.
- Add, update, remove, and inspect skill sources.
- Sync skills from a project or global skills lock file.
- Review session action logs, including command output details.

## Requirements

- Go version matching `go.mod`.
- Node.js/npm with `npx` available.
- The upstream `skills` CLI runnable as `npx skills ...`.

## Install

### From a release (Linux, macOS)

```sh
curl -fsSL https://github.com/ntk148v/knit/raw/master/scripts/install.sh | bash
```

### From a release (Windows)

```pwsh
iex "& { $(Invoke-RestMethod https://github.com/ntk148v/knit/raw/master/scripts/install.ps1) }"
```

The script installs `npx skills` (first run caches it) then downloads the latest `knit` binary from GitHub releases to `/usr/local/bin` (Unix) or `~/.local/bin` (Windows).

Override the install directory:

```sh
INSTALL_DIR=~/.local/bin curl -fsSL https://github.com/ntk148v/knit/raw/master/scripts/install.sh | bash
```

```pwsh
$InstallDir = "$env:USERPROFILE\bin"; iex "& { $(Invoke-RestMethod https://github.com/ntk148v/knit/raw/master/scripts/install.ps1) }"
```

### From source

```sh
go install ./cmd/knit
```

Or run without installing:

```sh
go run ./cmd/knit
```

## Usage

```sh
knit
```

`knit` opens with four tabs:

| Tab       | Purpose                                   |
| --------- | ----------------------------------------- |
| Installed | View and manage installed skills.         |
| Discover  | Search available skills and install them. |
| Sources   | Manage skill sources.                     |
| Logs      | Inspect actions run in this session.      |

### Common keys

| Key                                   | Action                                         |
| ------------------------------------- | ---------------------------------------------- |
| <kbd>Tab</kbd> / <kbd>Shift+Tab</kbd> | Switch tabs.                                   |
| <kbd>1</kbd>-<kbd>4</kbd>             | Jump to a tab.                                 |
| <kbd>/</kbd>                          | Focus search.                                  |
| <kbd>j</kbd>/<kbd>k</kbd> or arrows   | Move selection.                                |
| <kbd>Enter</kbd>                      | View selected item or confirm selected action. |
| <kbd>Esc</kbd>                        | Go back, close overlay, or clear search.       |
| <kbd>?</kbd>                          | Show help for the current tab.                 |
| <kbd>q</kbd> / <kbd>Ctrl+C</kbd>      | Quit.                                          |

### Tab actions

| Tab       | Keys                                                                 |
| --------- | -------------------------------------------------------------------- |
| Installed | <kbd>u</kbd> update, <kbd>d</kbd> uninstall, <kbd>c</kbd> actions.   |
| Discover  | <kbd>i</kbd> install, <kbd>s</kbd> add source, <kbd>c</kbd> actions. |
| Sources   | <kbd>a</kbd> add, <kbd>u</kbd> update, <kbd>d</kbd> remove.          |
| Logs      | <kbd>Enter</kbd> detail, <kbd>c</kbd> clear.                         |

## Sync mode

`knit` uses the upstream skills lock files directly:

| Scope   | Lock file                    |
| ------- | ---------------------------- |
| Project | `./skills-lock.json`         |
| Global  | `~/.agents/.skill-lock.json` |

Sync a lock file into the current project:

```sh
knit sync -f skills-lock.json
```

Sync a lock file globally:

```sh
knit sync -f ~/.agents/.skill-lock.json -g
```

Until [upstream skills sync](https://github.com/vercel-labs/skills/issues/283) is implemented, `knit sync` is the boring bridge: it reads the lock file and installs each entry with `npx skills add <source> --skill <name> -y`.

## Development

```sh
go test ./...
go run ./cmd/knit
```

Project layout:

- `cmd/knit` — CLI entrypoint.
- `internal/app` — Bubble Tea model, views, and key handling.
- `internal/skills` — `npx skills` adapter, parsers, and lock-file reader.

Keep changes boring: prefer small UI/state updates, no extra dependencies unless the standard library and current Charm stack cannot do it.

### Recording a demo GIF

The project includes a [VHS](https://github.com/charmbracelet/vhs) tape to record a UI demo GIF:

```sh
scripts/vhs/record.sh
```

Requires `vhs` from [github.com/charmbracelet/vhs](https://github.com/charmbracelet/vhs).

The recording uses a fake `npx` binary and isolated home directory fixtures so it's fully deterministic and independent of your local skills configuration.
