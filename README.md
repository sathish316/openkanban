# OpenKanban

A TUI kanban board for orchestrating AI coding agents across multiple projects.

```
    BACKLOG (3)          IN PROGRESS (2)         DONE (1)
 +-----------------+  +-----------------+  +-----------------+
 | auth-system     |  | api-endpoints   |  | db-schema       |
 | [idle]          |  | [working]       |  | [done]          |
 |                 |  | opencode        |  | claude          |
 +-----------------+  +-----------------+  +-----------------+
 | user-dashboard  |  | payment-flow    |
 | [idle]          |  | [working]       |
 |                 |  | claude          |
 +-----------------+  +-----------------+
 | notifications   |
 | [idle]          |
 |                 |
 +-----------------+

 [n]ew  [enter]open  [h/l]move  [d]elete  [q]uit  [?]help
```

## What It Does

Each ticket on the board represents a task. When you start working on a ticket:

1. **Git worktree created** - Isolated branch for that task
2. **AI agent spawned** - Claude Code, OpenCode, or your preferred agent
3. **tmux session managed** - Each task gets its own session
4. **Status tracked** - See which agents are working/idle/done

Move tickets across columns by dragging (or pressing `h`/`l`), and the board reflects real-time agent status.

## Why This Exists

Tools like [vibe-kanban](https://github.com/BloopAI/vibe-kanban) pioneered this concept with a web UI. But many developers prefer:

- Staying in the terminal
- Keyboard-driven workflows  
- Integration with existing tmux setups
- No browser/Electron overhead

OpenKanban brings the same workflow to your terminal.

## Features

### Core
- Kanban board with customizable columns
- Per-ticket git worktrees (isolated development)
- AI agent spawning (Claude Code, OpenCode, Aider, etc.)
- Real-time agent status monitoring
- tmux session management

### Navigation
- Vim-style keybindings (`h/j/k/l`)
- Quick ticket creation (`n`)
- Instant agent attachment (`enter`)
- Drag tickets between columns

### Integration
- Works with any AI coding agent
- Git worktree lifecycle management
- Status hooks for your statusline/dashboard
- JSON/SQLite state persistence

## Installation

```bash
# From source
go install github.com/techdufus/openkanban@latest

# Or build locally
git clone https://github.com/techdufus/openkanban
cd openkanban
go build -o openkanban .
```

## Quick Start

```bash
# Initialize in a git repository
cd ~/projects/my-app
openkanban init

# Launch the board
openkanban

# Or specify a different project
openkanban --project ~/projects/other-app
```

## Configuration

Config lives in `~/.config/openkanban/config.yaml` or `.openkanban.yaml` in your project:

```yaml
# Default AI agent to spawn
default_agent: opencode  # or: claude, aider, cursor

# Columns (customize your workflow)
columns:
  - name: Backlog
    key: backlog
  - name: In Progress  
    key: in_progress
  - name: Review
    key: review
  - name: Done
    key: done

# Worktree settings
worktree:
  base_dir: .worktrees  # Where to create worktrees
  branch_prefix: task/  # Branch naming: task/ticket-slug

# Agent-specific settings
agents:
  opencode:
    command: opencode
    args: ["--continue"]
  claude:
    command: claude
    args: []
  aider:
    command: aider
    args: ["--yes"]

# tmux settings
tmux:
  session_prefix: ab-  # Session naming: ab-ticket-slug
```

## Keybindings

| Key | Action |
|-----|--------|
| `j/k` | Move cursor up/down |
| `h/l` | Move ticket left/right (change status) |
| `enter` | Open/attach to ticket's agent session |
| `n` | Create new ticket |
| `e` | Edit ticket title |
| `d` | Delete ticket (with confirmation) |
| `r` | Refresh agent statuses |
| `s` | Sync worktrees |
| `?` | Show help |
| `q` | Quit |

## How It Works

### Ticket Lifecycle

1. **Create ticket** (`n`)
   - Prompts for title/description
   - Generates slug from title
   - Saves to state file

2. **Start work** (move to "In Progress")
   - Creates git worktree: `.worktrees/task-slug`
   - Creates branch: `task/ticket-slug`
   - Spawns tmux session: `ab-ticket-slug`
   - Launches configured agent in session

3. **Open ticket** (`enter`)
   - Attaches to existing tmux session
   - Or creates one if worktree exists

4. **Complete ticket** (move to "Done")
   - Agent keeps running (manual cleanup)
   - Or auto-cleanup if configured

5. **Delete ticket** (`d`)
   - Kills tmux session
   - Removes worktree
   - Deletes branch (optional)

### State Management

State is persisted to `.openkanban/state.json`:

```json
{
  "tickets": [
    {
      "id": "abc123",
      "title": "Implement authentication",
      "slug": "implement-auth",
      "status": "in_progress",
      "agent": "opencode",
      "worktree": ".worktrees/implement-auth",
      "branch": "task/implement-auth",
      "created_at": "2024-12-16T10:00:00Z",
      "updated_at": "2024-12-16T14:30:00Z"
    }
  ]
}
```

## Architecture

```
openkanban/
├── cmd/
│   └── openkanban/
│       └── main.go           # Entry point
├── internal/
│   ├── ui/                   # Bubbletea TUI
│   │   ├── app.go           # Main application model
│   │   ├── board.go         # Kanban board component
│   │   ├── ticket.go        # Ticket card component
│   │   ├── dialog.go        # Modal dialogs
│   │   └── styles.go        # Lipgloss styles
│   ├── core/                 # Business logic
│   │   ├── ticket.go        # Ticket model
│   │   ├── board.go         # Board operations
│   │   └── config.go        # Configuration
│   ├── git/                  # Git operations
│   │   ├── worktree.go      # Worktree management
│   │   └── branch.go        # Branch operations
│   ├── agent/                # Agent spawning
│   │   ├── manager.go       # Agent lifecycle
│   │   ├── status.go        # Status monitoring
│   │   └── tmux.go          # tmux integration
│   └── store/                # Persistence
│       ├── json.go          # JSON file store
│       └── sqlite.go        # SQLite store (optional)
├── docs/
│   ├── ARCHITECTURE.md
│   ├── AGENT_INTEGRATION.md
│   ├── DATA_MODEL.md
│   └── UI_DESIGN.md
├── go.mod
├── go.sum
└── README.md
```

## Supported Agents

| Agent | Status | Notes |
|-------|--------|-------|
| OpenCode | Full | Native support |
| Claude Code | Full | Native support |
| Aider | Full | `--yes` flag recommended |
| Cursor | Partial | Opens in GUI |
| Codex | Planned | |

## Roadmap

### MVP (v0.1)
- [ ] Basic kanban board UI
- [ ] Ticket CRUD
- [ ] Git worktree creation
- [ ] tmux session spawning
- [ ] Agent status detection

### v0.2
- [ ] Multiple agent support
- [ ] Custom columns
- [ ] Ticket filtering/search
- [ ] Status hooks for external tools

### v0.3
- [ ] SQLite persistence option
- [ ] Ticket templates
- [ ] Time tracking
- [ ] GitHub/GitLab issue sync

## Prior Art

- [vibe-kanban](https://github.com/BloopAI/vibe-kanban) - Web-based kanban for AI agents (inspiration)
- [claude-flow](https://github.com/ruvnet/claude-flow) - Multi-agent orchestration
- [kanban-tui](https://github.com/Zaloog/kanban-tui) - Python TUI kanban (no AI integration)

## License

MIT
