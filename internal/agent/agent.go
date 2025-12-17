package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/techdufus/openkanban/internal/board"
	"github.com/techdufus/openkanban/internal/config"
)

// Manager handles AI agent lifecycle
type Manager struct {
	config *config.Config
}

// NewManager creates a new agent manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{config: cfg}
}

// SpawnAgent starts an AI agent for a ticket in a tmux session
func (m *Manager) SpawnAgent(ticket *board.Ticket, agentType string) error {
	agentCfg, ok := m.config.Agents[agentType]
	if !ok {
		return fmt.Errorf("unknown agent type: %s", agentType)
	}

	sessionName := ticket.TmuxSession
	if sessionName == "" {
		return fmt.Errorf("ticket has no tmux session name")
	}

	workdir := ticket.WorktreePath
	if workdir == "" {
		return fmt.Errorf("ticket has no worktree path")
	}

	// Check if session already exists
	if m.SessionExists(sessionName) {
		return fmt.Errorf("tmux session already exists: %s", sessionName)
	}

	// Build the agent command
	cmdParts := []string{agentCfg.Command}
	cmdParts = append(cmdParts, agentCfg.Args...)

	// Prepare initial prompt if configured
	initPrompt := ""
	if agentCfg.InitPrompt != "" {
		tmpl, err := template.New("prompt").Parse(agentCfg.InitPrompt)
		if err == nil {
			var buf bytes.Buffer
			tmpl.Execute(&buf, ticket)
			initPrompt = buf.String()
		}
	}

	// Create tmux session
	args := []string{
		"new-session",
		"-d",
		"-s", sessionName,
		"-c", workdir,
	}

	// Start the agent command in the session
	agentCmd := strings.Join(cmdParts, " ")
	if initPrompt != "" {
		// Some agents accept initial prompt via stdin or argument
		// This is agent-specific and may need customization
		agentCmd = fmt.Sprintf("%s", agentCmd)
	}
	args = append(args, agentCmd)

	cmd := exec.Command("tmux", args...)

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range agentCfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	ticket.AgentType = agentType
	ticket.AgentStatus = board.AgentIdle

	return nil
}

// StopAgent terminates the agent session for a ticket
func (m *Manager) StopAgent(ticket *board.Ticket) error {
	if ticket.TmuxSession == "" {
		return nil
	}

	if !m.SessionExists(ticket.TmuxSession) {
		ticket.AgentStatus = board.AgentNone
		return nil
	}

	cmd := exec.Command("tmux", "kill-session", "-t", ticket.TmuxSession)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}

	ticket.AgentStatus = board.AgentNone
	return nil
}

// AttachSession attaches to a ticket's tmux session
func (m *Manager) AttachSession(sessionName string) error {
	if !m.SessionExists(sessionName) {
		return fmt.Errorf("session does not exist: %s", sessionName)
	}

	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// SessionExists checks if a tmux session exists
func (m *Manager) SessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

// GetStatus determines the current status of an agent
func (m *Manager) GetStatus(ticket *board.Ticket) board.AgentStatus {
	if ticket.TmuxSession == "" {
		return board.AgentNone
	}

	// Check if session exists
	if !m.SessionExists(ticket.TmuxSession) {
		return board.AgentNone
	}

	// Try to read status file
	if ticket.AgentType != "" {
		agentCfg, ok := m.config.Agents[ticket.AgentType]
		if ok && agentCfg.StatusFile != "" {
			statusPath := filepath.Join(ticket.WorktreePath, agentCfg.StatusFile)
			if status := m.readStatusFile(statusPath); status != "" {
				return board.AgentStatus(status)
			}
		}
	}

	// Fall back to activity detection
	return m.detectActivity(ticket.TmuxSession)
}

// readStatusFile reads agent status from a status file
func (m *Manager) readStatusFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var status struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(data, &status); err != nil {
		return ""
	}

	return status.Status
}

// detectActivity detects agent activity by checking tmux pane output
func (m *Manager) detectActivity(sessionName string) board.AgentStatus {
	// Get pane PID
	cmd := exec.Command("tmux", "list-panes", "-t", sessionName, "-F", "#{pane_pid}")
	output, err := cmd.Output()
	if err != nil {
		return board.AgentIdle
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
		return board.AgentIdle
	}

	// Check if process has children (indicating active work)
	cmd = exec.Command("pgrep", "-P", pid)
	if err := cmd.Run(); err == nil {
		return board.AgentWorking
	}

	return board.AgentIdle
}

// PollStatuses updates agent statuses for all tickets
func (m *Manager) PollStatuses(tickets map[board.TicketID]*board.Ticket) {
	for _, ticket := range tickets {
		if ticket.Status == board.StatusInProgress {
			ticket.AgentStatus = m.GetStatus(ticket)
		}
	}
}

// StatusPollInterval returns the configured polling interval
func (m *Manager) StatusPollInterval() time.Duration {
	interval := m.config.UI.RefreshInterval
	if interval <= 0 {
		interval = 5
	}
	return time.Duration(interval) * time.Second
}
