# AGENTS.md - OpenKanban

TUI kanban board for orchestrating AI coding agents. Built with Go 1.23+, Bubbletea, and Lipgloss.

## Commands

| Task | Command |
|------|---------|
| Build | `go build ./...` |
| All tests | `go test ./...` |
| Single package | `go test ./internal/config/...` |
| Single test | `go test -run TestName ./...` |
| Lint | `go vet ./...` |

## Critical Patterns

**Bubbletea (Elm Architecture)**: Never block the UI thread. All I/O, network calls, and long operations must return `tea.Cmd` for async execution. The Update function must return immediately.

**Message Flow**: User input -> `tea.Msg` -> `Update()` returns new model + `tea.Cmd` -> command executes async -> produces new `tea.Msg` -> cycle repeats.

**PTY Integration**: Terminal panes use `creack/pty` and `hinshun/vt10x`. Agent sessions run in embedded PTYs, not tmux.

## Code Style

- **Imports**: stdlib, blank line, external, blank line, internal (`github.com/techdufus/openkanban/...`)
- **Naming**: PascalCase exported, camelCase private, snake_case JSON tags
- **Errors**: Return error last, wrap with context via `fmt.Errorf`
- **Config principle**: All user-facing behavior must be configurable

## Package Responsibilities

| Package | Purpose |
|---------|---------|
| `internal/ui/` | Bubbletea model, view, update cycle. All rendering logic. |
| `internal/board/` | Ticket/Board data structures, persistence (JSON). |
| `internal/agent/` | Agent config, status detection, context injection. |
| `internal/terminal/` | PTY-based terminal panes for embedded agents. |
| `internal/git/` | Worktree creation/removal, branch management. |
| `internal/config/` | Global config loading from `~/.config/openkanban/config.json`. |

## Key Files

- `internal/ui/model.go` - Main UI state and Update logic (the heart of the app)
- `internal/ui/view.go` - All rendering code
- `internal/board/board.go` - Ticket/Board types and persistence
- `internal/config/config.go` - Config types and defaults

## Active Refactoring

Issue #47: Transitioning from "Board" to "Project" model. Check issue for current terminology decisions before adding new board-related code.

## Do Not

- Block in `Update()` - always return `tea.Cmd` for async work
- Use tmux directly - the app uses embedded PTY panes
- Add AI attribution to commits
- Assume config values exist - check `config.go` for defaults and use them

## Commits

Use conventional commits: `feat:`, `fix:`, `refactor:`, `perf:`, `docs:`, `test:`, `chore:`
