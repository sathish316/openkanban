# Data Model & Persistence

This document defines the data structures, persistence strategies, and state management for Agent Board.

## Core Data Structures

### Ticket

The fundamental unit of work. Each ticket represents a task with an associated git worktree and agent session.

```go
type TicketID string // UUID v4

type TicketStatus string

const (
    StatusBacklog    TicketStatus = "backlog"
    StatusInProgress TicketStatus = "in_progress"
    StatusDone       TicketStatus = "done"
    StatusArchived   TicketStatus = "archived"
)

type AgentStatus string

const (
    AgentIdle      AgentStatus = "idle"      // Session exists, no activity
    AgentWorking   AgentStatus = "working"   // Active output detected
    AgentWaiting   AgentStatus = "waiting"   // Waiting for user input
    AgentCompleted AgentStatus = "completed" // Agent reported done
    AgentError     AgentStatus = "error"     // Agent crashed/errored
    AgentNone      AgentStatus = "none"      // No session spawned
)

type Ticket struct {
    ID          TicketID     `json:"id"`
    Title       string       `json:"title"`
    Description string       `json:"description,omitempty"`
    Status      TicketStatus `json:"status"`
    
    // Git integration
    WorktreePath string `json:"worktree_path,omitempty"`
    BranchName   string `json:"branch_name,omitempty"`
    BaseBranch   string `json:"base_branch,omitempty"` // e.g., "main"
    
    // Agent integration
    AgentType    string      `json:"agent_type,omitempty"` // "claude", "opencode", "aider"
    AgentStatus  AgentStatus `json:"agent_status"`
    TmuxSession  string      `json:"tmux_session,omitempty"`
    
    // Metadata
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    StartedAt   *time.Time `json:"started_at,omitempty"`   // When moved to in_progress
    CompletedAt *time.Time `json:"completed_at,omitempty"` // When moved to done
    
    // User-defined
    Labels   []string          `json:"labels,omitempty"`
    Priority int               `json:"priority,omitempty"` // 1=highest, 5=lowest
    Meta     map[string]string `json:"meta,omitempty"`     // Custom key-value pairs
}
```

### Board

Container for tickets with board-level configuration.

```go
type Board struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    RepoPath  string    `json:"repo_path"`  // Absolute path to git repo
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    
    // Columns (ordered)
    Columns []Column `json:"columns"`
    
    // All tickets (keyed by ID for fast lookup)
    Tickets map[TicketID]*Ticket `json:"tickets"`
    
    // Board settings
    Settings BoardSettings `json:"settings"`
}

type Column struct {
    ID     string       `json:"id"`
    Name   string       `json:"name"`
    Status TicketStatus `json:"status"` // Maps to ticket status
    Color  string       `json:"color"`  // Hex color for column header
    Limit  int          `json:"limit"`  // WIP limit (0 = unlimited)
}

type BoardSettings struct {
    DefaultAgent     string `json:"default_agent"`      // Default agent to spawn
    WorktreeBase     string `json:"worktree_base"`      // Base dir for worktrees
    AutoSpawnAgent   bool   `json:"auto_spawn_agent"`   // Spawn agent on move to in_progress
    AutoCreateBranch bool   `json:"auto_create_branch"` // Create branch on move to in_progress
    BranchPrefix     string `json:"branch_prefix"`      // e.g., "agent/" or "feature/"
    TmuxPrefix       string `json:"tmux_prefix"`        // e.g., "ab-" for session names
}
```

### Application State

Runtime state for the TUI application.

```go
type AppState struct {
    // Current board
    Board *Board
    
    // UI state
    ActiveColumn int        // Currently selected column index
    ActiveTicket int        // Currently selected ticket index within column
    Mode         UIMode     // Normal, Insert, Command, Help
    
    // Filtering/search
    FilterLabels []string
    SearchQuery  string
    
    // Cached views
    ColumnTickets [][]TicketID // Tickets per column (filtered/sorted)
    
    // Agent monitoring
    AgentStatuses map[TicketID]AgentStatus // Real-time status cache
    LastPoll      time.Time
}

type UIMode string

const (
    ModeNormal  UIMode = "normal"
    ModeInsert  UIMode = "insert"  // Editing ticket
    ModeCommand UIMode = "command" // : command mode
    ModeHelp    UIMode = "help"    // Help overlay
    ModeConfirm UIMode = "confirm" // Confirmation dialog
)
```

## Persistence Strategy

### File-Based Storage (Default)

Simple JSON files, human-readable and git-friendly.

```
~/.config/openkanban/
├── config.json           # Global configuration
└── boards/
    └── {board-id}/
        ├── board.json    # Board metadata + settings
        └── tickets/
            ├── {ticket-id}.json
            └── ...

# Or single-file approach (simpler):
~/.config/openkanban/
├── config.json
└── boards/
    └── {board-id}.json   # Everything in one file
```

#### Single-File Format (Recommended for <1000 tickets)

```json
{
  "id": "proj-abc123",
  "name": "My Project",
  "repo_path": "/home/user/projects/myproject",
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-16T14:30:00Z",
  "columns": [
    {"id": "backlog", "name": "Backlog", "status": "backlog", "color": "#89b4fa", "limit": 0},
    {"id": "in-progress", "name": "In Progress", "status": "in_progress", "color": "#f9e2af", "limit": 3},
    {"id": "done", "name": "Done", "status": "done", "color": "#a6e3a1", "limit": 0}
  ],
  "tickets": {
    "ticket-uuid-1": {
      "id": "ticket-uuid-1",
      "title": "Implement user authentication",
      "description": "Add JWT-based auth to the API",
      "status": "in_progress",
      "worktree_path": "/home/user/projects/myproject-worktrees/auth-feature",
      "branch_name": "agent/auth-feature",
      "agent_type": "claude",
      "agent_status": "working",
      "tmux_session": "ab-ticket-uuid-1",
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-16T14:30:00Z",
      "started_at": "2025-01-16T09:00:00Z",
      "labels": ["backend", "security"],
      "priority": 1
    }
  },
  "settings": {
    "default_agent": "claude",
    "worktree_base": "/home/user/projects/myproject-worktrees",
    "auto_spawn_agent": true,
    "auto_create_branch": true,
    "branch_prefix": "agent/",
    "tmux_prefix": "ab-"
  }
}
```

### SQLite Storage (Optional, for large boards)

For boards with >1000 tickets or complex querying needs.

```sql
-- Schema
CREATE TABLE boards (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    settings JSON NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE columns (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    color TEXT,
    wip_limit INTEGER DEFAULT 0,
    position INTEGER NOT NULL,
    UNIQUE(board_id, position)
);

CREATE TABLE tickets (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id),
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'backlog',
    worktree_path TEXT,
    branch_name TEXT,
    base_branch TEXT,
    agent_type TEXT,
    agent_status TEXT DEFAULT 'none',
    tmux_session TEXT,
    priority INTEGER DEFAULT 3,
    labels JSON DEFAULT '[]',
    meta JSON DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
);

CREATE INDEX idx_tickets_board_status ON tickets(board_id, status);
CREATE INDEX idx_tickets_agent_status ON tickets(agent_status);
```

### Storage Interface

Abstract storage to support both backends:

```go
type Storage interface {
    // Board operations
    CreateBoard(board *Board) error
    GetBoard(id string) (*Board, error)
    UpdateBoard(board *Board) error
    DeleteBoard(id string) error
    ListBoards() ([]*Board, error)
    
    // Ticket operations
    CreateTicket(boardID string, ticket *Ticket) error
    GetTicket(boardID string, ticketID TicketID) (*Ticket, error)
    UpdateTicket(boardID string, ticket *Ticket) error
    DeleteTicket(boardID string, ticketID TicketID) error
    ListTickets(boardID string, filter TicketFilter) ([]*Ticket, error)
    
    // Batch operations
    MoveTicket(boardID string, ticketID TicketID, newStatus TicketStatus) error
    ReorderTickets(boardID string, status TicketStatus, ticketIDs []TicketID) error
}

type TicketFilter struct {
    Status   []TicketStatus
    Labels   []string
    Priority *int
    Search   string
}
```

## State Transitions

### Ticket Lifecycle

```
                    ┌─────────────┐
                    │   Created   │
                    └──────┬──────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────┐
│                      BACKLOG                              │
│  - No worktree                                           │
│  - No agent session                                      │
│  - agent_status = "none"                                 │
└──────────────────────┬───────────────────────────────────┘
                       │ Move to In Progress
                       │ (triggers: create worktree, spawn agent)
                       ▼
┌──────────────────────────────────────────────────────────┐
│                   IN PROGRESS                             │
│  - Worktree created at {worktree_base}/{branch_name}     │
│  - Branch created: {branch_prefix}{ticket-id-short}      │
│  - Tmux session: {tmux_prefix}{ticket-id}                │
│  - agent_status cycles: idle → working → waiting → ...   │
└──────────────────────┬───────────────────────────────────┘
                       │ Move to Done
                       │ (triggers: optional cleanup prompt)
                       ▼
┌──────────────────────────────────────────────────────────┐
│                       DONE                                │
│  - Worktree can be kept or removed                       │
│  - Agent session terminated                              │
│  - agent_status = "completed" or "none"                  │
│  - Branch ready for PR                                   │
└──────────────────────┬───────────────────────────────────┘
                       │ Archive (optional)
                       ▼
┌──────────────────────────────────────────────────────────┐
│                     ARCHIVED                              │
│  - Hidden from default view                              │
│  - Worktree removed                                      │
│  - Historical record preserved                           │
└──────────────────────────────────────────────────────────┘
```

### Agent Status Transitions

```
     spawn agent
         │
         ▼
      ┌──────┐
      │ idle │◄─────────────────────┐
      └──┬───┘                      │
         │ activity detected        │ no activity (30s)
         ▼                          │
    ┌─────────┐                     │
    │ working │─────────────────────┘
    └────┬────┘
         │ waiting for input
         ▼
    ┌─────────┐
    │ waiting │ (prompt detected)
    └────┬────┘
         │ user responds OR
         │ agent continues
         ▼
    ┌─────────────┐
    │ working/idle│
    └─────────────┘

    Error states:
    - Process exits unexpectedly → "error"
    - Status file says "done" → "completed"
    - Session killed → "none"
```

## Global Configuration

```go
type Config struct {
    // Default board settings
    Defaults BoardSettings `json:"defaults"`
    
    // Agent configurations
    Agents map[string]AgentConfig `json:"agents"`
    
    // UI preferences
    UI UIConfig `json:"ui"`
    
    // Keybindings (optional overrides)
    Keys map[string]string `json:"keys,omitempty"`
}

type AgentConfig struct {
    Command     string            `json:"command"`      // e.g., "claude"
    Args        []string          `json:"args"`         // e.g., ["--dangerously-skip-permissions"]
    Env         map[string]string `json:"env"`          // Additional env vars
    StatusFile  string            `json:"status_file"`  // Relative path to status file
    InitPrompt  string            `json:"init_prompt"`  // Template for initial prompt
}

type UIConfig struct {
    Theme           string `json:"theme"`            // "catppuccin-mocha", "dracula", etc.
    ShowAgentStatus bool   `json:"show_agent_status"`
    RefreshInterval int    `json:"refresh_interval"` // Seconds between agent status polls
    ColumnWidth     int    `json:"column_width"`     // Characters
    TicketHeight    int    `json:"ticket_height"`    // Lines per ticket card
}
```

### Default Configuration File

```json
{
  "defaults": {
    "default_agent": "claude",
    "worktree_base": "",
    "auto_spawn_agent": true,
    "auto_create_branch": true,
    "branch_prefix": "agent/",
    "tmux_prefix": "ab-"
  },
  "agents": {
    "claude": {
      "command": "claude",
      "args": ["--dangerously-skip-permissions"],
      "env": {},
      "status_file": ".claude/status.json",
      "init_prompt": "You are working on: {{.Title}}\n\nDescription:\n{{.Description}}\n\nBranch: {{.BranchName}}\nBase: {{.BaseBranch}}"
    },
    "opencode": {
      "command": "opencode",
      "args": [],
      "env": {},
      "status_file": ".opencode/status.json",
      "init_prompt": "Task: {{.Title}}\n\n{{.Description}}"
    },
    "aider": {
      "command": "aider",
      "args": ["--yes"],
      "env": {},
      "status_file": "",
      "init_prompt": ""
    }
  },
  "ui": {
    "theme": "catppuccin-mocha",
    "show_agent_status": true,
    "refresh_interval": 5,
    "column_width": 40,
    "ticket_height": 4
  }
}
```

## File Paths

| Purpose | Path | Notes |
|---------|------|-------|
| Global config | `~/.config/openkanban/config.json` | User preferences |
| Board data | `~/.config/openkanban/boards/{id}.json` | Per-board state |
| Worktrees | `{repo}/../{repo}-worktrees/` | Default location |
| Agent status | `{worktree}/.claude/status.json` | Agent-specific |
| Logs | `~/.local/state/openkanban/logs/` | Debug logs |

## Concurrency Considerations

1. **File locking**: Use `flock` or similar when writing board JSON
2. **Atomic writes**: Write to temp file, then rename
3. **Agent polling**: Run in separate goroutine, update state via channels
4. **Tmux operations**: Serial execution to avoid race conditions

```go
// Atomic write pattern
func (s *JSONStorage) SaveBoard(board *Board) error {
    data, err := json.MarshalIndent(board, "", "  ")
    if err != nil {
        return err
    }
    
    tmpFile := s.boardPath(board.ID) + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0644); err != nil {
        return err
    }
    
    return os.Rename(tmpFile, s.boardPath(board.ID))
}
```
