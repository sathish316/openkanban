# Agent Integration

This document describes how Agent Board spawns, monitors, and manages AI coding agents.

## Overview

Agent Board treats AI agents as external processes managed through tmux sessions. This approach provides:

- **Isolation**: Each ticket gets its own session
- **Persistence**: Sessions survive Agent Board restarts
- **Flexibility**: Any CLI-based agent works
- **Familiarity**: Users can manage sessions with standard tmux commands

## Supported Agents

### Tier 1: Full Support
Agents with native status detection and tested integration.

| Agent | Command | Status Detection | Notes |
|-------|---------|------------------|-------|
| OpenCode | `opencode` | Process + hooks | Your primary target |
| Claude Code | `claude` | Process + hooks | Anthropic's CLI |
| Aider | `aider` | Process only | Use `--yes --no-auto-commits` |

### Tier 2: Basic Support
Agents that work but may have limited status detection.

| Agent | Command | Notes |
|-------|---------|-------|
| Cursor | `cursor` | Opens GUI, limited TUI integration |
| Continue | `continue` | TUI mode supported |
| Codex | `codex` | When available |

### Tier 3: Generic Support
Any CLI tool that runs interactively.

```yaml
agents:
  custom-agent:
    command: /path/to/agent
    args: ["--interactive"]
```

## Spawning Agents

### Lifecycle

```
┌─────────────┐
│ Ticket in   │
│  Backlog    │
└─────────────┘
       │
       │ User moves to "In Progress"
       ▼
┌─────────────────────────────────────────┐
│ 1. Create git worktree                  │
│    git worktree add .worktrees/slug     │
│    task/ticket-slug                     │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ 2. Create tmux session                  │
│    tmux new-session -d -s ab-slug       │
│    -c .worktrees/slug                   │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ 3. Send agent command                   │
│    tmux send-keys -t ab-slug            │
│    'opencode --continue' Enter          │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────┐
│ Agent is    │
│  running    │
└─────────────┘
```

### Implementation

```go
// internal/agent/spawn.go

type SpawnOptions struct {
    Ticket      *core.Ticket
    WorktreePath string
    AgentConfig  AgentConfig
}

func (m *Manager) Spawn(opts SpawnOptions) error {
    sessionName := m.sessionName(opts.Ticket)
    
    // Create tmux session in worktree directory
    createCmd := exec.Command("tmux", "new-session",
        "-d",                      // Detached
        "-s", sessionName,         // Session name
        "-c", opts.WorktreePath,   // Working directory
    )
    if err := createCmd.Run(); err != nil {
        return fmt.Errorf("creating tmux session: %w", err)
    }
    
    // Build agent command
    agentCmd := opts.AgentConfig.Command
    if len(opts.AgentConfig.Args) > 0 {
        agentCmd += " " + strings.Join(opts.AgentConfig.Args, " ")
    }
    
    // Send the command to start the agent
    sendCmd := exec.Command("tmux", "send-keys",
        "-t", sessionName,
        agentCmd, "Enter",
    )
    if err := sendCmd.Run(); err != nil {
        return fmt.Errorf("starting agent: %w", err)
    }
    
    // Register for status tracking
    m.registerSession(opts.Ticket.ID, sessionName)
    
    return nil
}

func (m *Manager) sessionName(ticket *core.Ticket) string {
    return fmt.Sprintf("%s%s", m.config.SessionPrefix, ticket.Slug)
}
```

## Status Detection

Agent Board uses multiple methods to detect agent status:

### Method 1: Status Files (Preferred)

Some agents (like Claude Code with hooks) write status files:

```
~/.cache/claude-status/
├── ab-auth-system.status    # "working" or "done"
├── ab-api-endpoints.status
└── ...
```

```go
// internal/agent/status_file.go

func (m *Manager) readStatusFile(ticket *core.Ticket) (AgentStatus, error) {
    pattern := m.config.Agents[ticket.Agent].StatusFile
    // Replace {session} with actual session name
    path := strings.Replace(pattern, "{session}", m.sessionName(ticket), 1)
    
    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return StatusUnknown, nil
    }
    if err != nil {
        return StatusUnknown, err
    }
    
    content := strings.TrimSpace(string(data))
    switch content {
    case "working":
        return StatusWorking, nil
    case "done", "idle":
        return StatusIdle, nil
    default:
        return StatusUnknown, nil
    }
}
```

### Method 2: Process Detection

Fall back to checking if agent process is running in the session:

```go
// internal/agent/status_process.go

func (m *Manager) detectFromProcess(ticket *core.Ticket) AgentStatus {
    sessionName := m.sessionName(ticket)
    
    // Get the TTY for this tmux session
    ttyCmd := exec.Command("tmux", "list-panes",
        "-t", sessionName,
        "-F", "#{pane_tty}",
    )
    ttyOut, err := ttyCmd.Output()
    if err != nil {
        return StatusNotRunning
    }
    tty := strings.TrimSpace(string(ttyOut))
    
    // Check if agent process is running on that TTY
    agentName := m.config.Agents[ticket.Agent].Command
    psCmd := exec.Command("ps", "aux")
    psOut, _ := psCmd.Output()
    
    // Parse ps output, look for agent on this TTY
    for _, line := range strings.Split(string(psOut), "\n") {
        if strings.Contains(line, agentName) && strings.Contains(line, tty) {
            // Agent is running
            // Check if it's actively working (heuristics)
            if strings.Contains(line, "R") || strings.Contains(line, "S+") {
                return StatusWorking
            }
            return StatusIdle
        }
    }
    
    return StatusNotRunning
}
```

### Method 3: Activity Heuristics

For agents without status files, detect activity from tmux pane:

```go
// internal/agent/status_activity.go

func (m *Manager) detectFromActivity(ticket *core.Ticket) AgentStatus {
    sessionName := m.sessionName(ticket)
    
    // Capture recent pane content
    captureCmd := exec.Command("tmux", "capture-pane",
        "-t", sessionName,
        "-p",           // Print to stdout
        "-S", "-5",     // Last 5 lines
    )
    out, err := captureCmd.Output()
    if err != nil {
        return StatusUnknown
    }
    
    content := string(out)
    
    // Heuristics for "working" state
    workingIndicators := []string{
        "Thinking...",
        "Writing...",
        "Reading...",
        "⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏", // Spinners
    }
    
    for _, indicator := range workingIndicators {
        if strings.Contains(content, indicator) {
            return StatusWorking
        }
    }
    
    // Check for prompt (agent waiting for input)
    promptIndicators := []string{
        "> ",
        "$ ",
        "❯ ",
    }
    
    lines := strings.Split(strings.TrimSpace(content), "\n")
    if len(lines) > 0 {
        lastLine := lines[len(lines)-1]
        for _, prompt := range promptIndicators {
            if strings.HasSuffix(lastLine, prompt) {
                return StatusIdle
            }
        }
    }
    
    return StatusUnknown
}
```

### Combined Status Detection

```go
// internal/agent/status.go

type AgentStatus int

const (
    StatusNotRunning AgentStatus = iota
    StatusWorking
    StatusIdle
    StatusUnknown
)

func (m *Manager) GetStatus(ticket *core.Ticket) AgentStatus {
    sessionName := m.sessionName(ticket)
    
    // First, check if session exists
    hasSession := exec.Command("tmux", "has-session", "-t", sessionName)
    if hasSession.Run() != nil {
        return StatusNotRunning
    }
    
    // Try status file first (most reliable)
    if m.config.UseStatusFiles {
        if status, err := m.readStatusFile(ticket); err == nil && status != StatusUnknown {
            return status
        }
    }
    
    // Fall back to process detection
    if m.config.FallbackToProcess {
        if status := m.detectFromProcess(ticket); status != StatusUnknown {
            return status
        }
    }
    
    // Last resort: activity heuristics
    return m.detectFromActivity(ticket)
}
```

## Session Management

### Attaching to Sessions

When user opens a ticket, suspend the TUI and attach:

```go
// internal/agent/attach.go

func (m *Manager) Attach(ticket *core.Ticket) tea.Cmd {
    sessionName := m.sessionName(ticket)
    
    // Use tea.ExecProcess to suspend TUI and run tmux attach
    return tea.ExecProcess(
        exec.Command("tmux", "attach-session", "-t", sessionName),
        func(err error) tea.Msg {
            if err != nil {
                return ErrorMsg{Err: err}
            }
            return SessionDetachedMsg{TicketID: ticket.ID}
        },
    )
}
```

### Killing Sessions

When user deletes a ticket or cleans up:

```go
// internal/agent/cleanup.go

func (m *Manager) Kill(ticket *core.Ticket) error {
    sessionName := m.sessionName(ticket)
    
    // Kill the tmux session
    cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
    return cmd.Run()
}
```

### Listing Sessions

For status overview:

```go
// internal/agent/list.go

type SessionInfo struct {
    Name      string
    TicketID  string
    Created   time.Time
    Attached  bool
}

func (m *Manager) ListSessions() ([]SessionInfo, error) {
    cmd := exec.Command("tmux", "list-sessions",
        "-F", "#{session_name}:#{session_created}:#{session_attached}",
    )
    out, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    var sessions []SessionInfo
    for _, line := range strings.Split(string(out), "\n") {
        if !strings.HasPrefix(line, m.config.SessionPrefix) {
            continue
        }
        parts := strings.Split(line, ":")
        // Parse and append...
    }
    return sessions, nil
}
```

## Status Hooks Integration

### For Claude Code

Create a hook that writes status files:

```bash
# ~/.claude/hooks/openkanban-hook
#!/bin/bash
# Called by Claude Code hooks system

SESSION=$(tmux display-message -p '#S' 2>/dev/null)
if [[ "$SESSION" == ab-* ]]; then
    STATUS_DIR="$HOME/.cache/openkanban-status"
    mkdir -p "$STATUS_DIR"
    
    case "$1" in
        PreToolUse|PreAskUser)
            echo "working" > "$STATUS_DIR/${SESSION}.status"
            ;;
        PostToolUse|Stop)
            echo "idle" > "$STATUS_DIR/${SESSION}.status"
            ;;
    esac
fi
```

### For OpenCode

OpenCode can be configured to emit status:

```yaml
# ~/.config/opencode/opencode.json
{
  "hooks": {
    "onStart": "echo working > ~/.cache/openkanban-status/${TMUX_SESSION}.status",
    "onIdle": "echo idle > ~/.cache/openkanban-status/${TMUX_SESSION}.status"
  }
}
```

## Polling Strategy

Agent Board polls status at configurable intervals:

```go
// internal/agent/poller.go

func (m *Manager) StartPolling(interval time.Duration) tea.Cmd {
    return tea.Every(interval, func(t time.Time) tea.Msg {
        statuses := make(map[string]AgentStatus)
        for ticketID := range m.running {
            ticket := m.store.GetTicket(ticketID)
            statuses[ticketID] = m.GetStatus(ticket)
        }
        return AgentStatusBatchMsg{Statuses: statuses}
    })
}
```

Default: Poll every 2 seconds for active tickets.

## Error Handling

### Spawn Failures

```go
func (m *Manager) Spawn(opts SpawnOptions) error {
    // ... spawn logic ...
    
    if err != nil {
        // Clean up partial state
        m.Kill(opts.Ticket) // Kill session if it was created
        return &SpawnError{
            Ticket: opts.Ticket,
            Phase:  "agent_start",
            Err:    err,
        }
    }
}
```

### Agent Crashes

Detected via status polling:

```go
func (m Model) handleStatusUpdate(msg AgentStatusBatchMsg) (tea.Model, tea.Cmd) {
    for ticketID, status := range msg.Statuses {
        ticket := m.board.GetTicket(ticketID)
        
        if status == StatusNotRunning && ticket.Status == core.StatusInProgress {
            // Agent died unexpectedly
            m.notifications = append(m.notifications, Notification{
                Level:   Warning,
                Message: fmt.Sprintf("Agent for '%s' stopped", ticket.Title),
                Action:  "Press 'r' to restart",
            })
        }
    }
    return m, nil
}
```

## Adding New Agents

To add support for a new agent:

1. **Add configuration**:

```yaml
# config.yaml
agents:
  new-agent:
    command: new-agent-cli
    args: ["--mode", "interactive"]
    status_file: ""  # Optional
    detect_patterns:
      working: ["Processing", "Analyzing"]
      idle: ["> ", "Ready"]
```

2. **Test spawning**:

```bash
# Manual test
tmux new-session -d -s test-agent -c /tmp 'new-agent-cli --mode interactive'
tmux attach -t test-agent
# Verify agent works, then detach
```

3. **Verify status detection**:

```go
// In tests
func TestNewAgentStatusDetection(t *testing.T) {
    mgr := agent.NewManager(config)
    
    // Start agent
    ticket := &core.Ticket{Slug: "test", Agent: "new-agent"}
    mgr.Spawn(SpawnOptions{Ticket: ticket, WorktreePath: "/tmp"})
    
    // Check status detection
    time.Sleep(2 * time.Second)
    status := mgr.GetStatus(ticket)
    
    if status == StatusUnknown {
        t.Error("Could not detect agent status")
    }
}
```

## Security Considerations

### Command Injection

Agent commands come from config, not user input:

```go
// SAFE: Config-defined command
agentCmd := config.Agents[ticket.Agent].Command

// NEVER: User-provided command
// agentCmd := userInput  // DON'T DO THIS
```

### Session Isolation

Each ticket's session is named with a prefix:

```go
// Only manage our sessions
if !strings.HasPrefix(sessionName, config.SessionPrefix) {
    return ErrNotOurSession
}
```

### Worktree Paths

Validate worktree paths are within expected directory:

```go
func (m *Manager) validateWorktreePath(path string) error {
    abs, err := filepath.Abs(path)
    if err != nil {
        return err
    }
    
    expected := filepath.Join(m.repoRoot, m.config.WorktreeDir)
    if !strings.HasPrefix(abs, expected) {
        return ErrInvalidWorktreePath
    }
    return nil
}
```
