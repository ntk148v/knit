# knit

`knit` is a keyboard-first terminal UI for managing [`npx skills`](https://github.com/vercel-labs/skills) skills.

It wraps the skills CLI with a small Bubble Tea interface so you can list installed skills, discover new ones, manage sources, and keep a session log without remembering every command.

## Features

- Browse project and global skills in one place.
- Search installed skills, discovered skills, and sources.
- Install, update, uninstall, and prune skills from the TUI.
- Add, update, remove, and inspect skill sources.
- Dump/sync a portable skills snapshot at `~/.config/knit/knit.json`.
- Review action logs for commands run during the session.

## Requirements

- Go version matching `go.mod`.
- Node.js/npm with `npx` available.
- The upstream `skills` CLI runnable as `npx skills ...`.

## Install

From this repository:

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

| Tab       | Purpose                                    |
| --------- | ------------------------------------------ |
| Installed | View and manage installed skills.          |
| Discover  | Search available skills and install them.  |
| Sources   | Manage skill sources and dump/sync config. |
| Logs      | Inspect actions run in this session.       |

### Common keys

| Key                 | Action                                         |
| ------------------- | ---------------------------------------------- |
| `Tab` / `Shift+Tab` | Switch tabs.                                   |
| `1`-`4`             | Jump to a tab.                                 |
| `/`                 | Focus search.                                  |
| `j`/`k` or arrows   | Move selection.                                |
| `Enter`             | View selected item or confirm selected action. |
| `Esc`               | Go back, close overlay, or clear search.       |
| `?`                 | Show help for the current tab.                 |
| `q` / `Ctrl+C`      | Quit.                                          |

### Tab actions

| Tab       | Keys                                                        |
| --------- | ----------------------------------------------------------- |
| Installed | `u` update, `d` uninstall, `c` actions, `D` dump, `S` sync. |
| Discover  | `i` install, `s` add source, `c` actions.                   |
| Sources   | `a` add, `u` update, `d` remove, `D` dump, `S` sync.        |
| Logs      | `Enter` detail, `c` clear.                                  |

## Snapshot config

`D` writes the current skills state to:

```text
~/.config/knit/knit.json
```

Example:

```json
{
  "sources": {
    "ntk148v/skills": "github.com/ntk148v/skills"
  },
  "skills": [
    {
      "source": "skills/demo",
      "scope": "project",
      "agents": ["opencode", "codex"]
    }
  ]
}
```

`S` reads that file and syncs sources and skills back through `npx skills`.

## Development

```sh
go test ./...
go run ./cmd/knit
```

Project layout:

- `cmd/knit` — CLI entrypoint.
- `internal/app` — Bubble Tea model, views, key handling.
- `internal/skills` — `npx skills` adapter and parsers.
- `internal/config` — snapshot config path/load/save helpers.

Keep changes boring: prefer small UI/state updates, no extra dependencies unless the standard library and current Charm stack cannot do it.
