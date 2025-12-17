package ui

import (
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/techdufus/openkanban/internal/agent"
	"github.com/techdufus/openkanban/internal/board"
	"github.com/techdufus/openkanban/internal/config"
	"github.com/techdufus/openkanban/internal/git"
)

// Mode represents the current UI mode
type Mode string

const (
	ModeNormal       Mode = "NORMAL"
	ModeInsert       Mode = "INSERT"
	ModeCommand      Mode = "COMMAND"
	ModeHelp         Mode = "HELP"
	ModeConfirm      Mode = "CONFIRM"
	ModeCreateTicket Mode = "CREATE"
)

const (
	minColumnWidth = 20
	columnOverhead = 5 // border (2) + padding (2) + margin (1)
)

// Model is the main Bubbletea model
type Model struct {
	// Configuration
	config *config.Config

	// Data
	board    *board.Board
	boardDir string

	// Managers
	agentMgr    *agent.Manager
	worktreeMgr *git.WorktreeManager

	// UI state
	mode           Mode
	activeColumn   int
	activeTicket   int
	width          int
	height         int
	animationFrame int
	scrollOffset   int

	// Cached column tickets
	columnTickets [][]*board.Ticket

	// Overlay state
	showHelp    bool
	showConfirm bool
	confirmMsg  string
	confirmFn   func() tea.Cmd

	// Create ticket form
	titleInput textinput.Model

	// Error/notification
	notification string
	notifyTime   time.Time
}

// NewModel creates a new UI model
func NewModel(cfg *config.Config, b *board.Board, boardDir string, agentMgr *agent.Manager, worktreeMgr *git.WorktreeManager) *Model {
	ti := textinput.New()
	ti.Placeholder = "Enter ticket title..."
	ti.CharLimit = 100
	ti.Width = 40

	m := &Model{
		config:      cfg,
		board:       b,
		boardDir:    boardDir,
		agentMgr:    agentMgr,
		worktreeMgr: worktreeMgr,
		mode:        ModeNormal,
		titleInput:  ti,
	}
	m.refreshColumnTickets()
	return m
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tickAgentStatus(m.agentMgr.StatusPollInterval()),
		tickAnimation(),
	)
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case agentStatusMsg:
		m.agentMgr.PollStatuses(m.board.Tickets)
		return m, tickAgentStatus(m.agentMgr.StatusPollInterval())

	case animationMsg:
		m.animationFrame = (m.animationFrame + 1) % 4
		return m, tickAnimation()

	case notificationMsg:
		if time.Since(m.notifyTime) > 3*time.Second {
			m.notification = ""
		}
		return m, nil
	}

	return m, nil
}

// handleKey processes keyboard input
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c", "q":
		if m.mode == ModeNormal {
			return m, tea.Quit
		}
	case "esc":
		m.mode = ModeNormal
		m.showHelp = false
		m.showConfirm = false
		m.titleInput.Blur()
		return m, nil
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	// Mode-specific handling
	if m.showHelp {
		// Any key closes help
		m.showHelp = false
		return m, nil
	}

	if m.showConfirm {
		return m.handleConfirm(msg)
	}

	switch m.mode {
	case ModeNormal:
		return m.handleNormalMode(msg)
	case ModeCommand:
		return m.handleCommandMode(msg)
	case ModeCreateTicket:
		return m.handleCreateTicketMode(msg)
	}

	return m, nil
}

// handleNormalMode processes keys in normal mode
func (m *Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Navigation
	case "h", "left":
		m.moveColumn(-1)
	case "l", "right":
		m.moveColumn(1)
	case "j", "down":
		m.moveTicket(1)
	case "k", "up":
		m.moveTicket(-1)
	case "g":
		m.activeTicket = 0
	case "G":
		if len(m.columnTickets) > m.activeColumn {
			m.activeTicket = len(m.columnTickets[m.activeColumn]) - 1
			if m.activeTicket < 0 {
				m.activeTicket = 0
			}
		}

	// Actions
	case "n":
		return m.createNewTicket()
	case "enter":
		return m.attachToAgent()
	case "d":
		return m.confirmDeleteTicket()
	case " ":
		return m.quickMoveTicket()
	case "s":
		return m.spawnAgent()
	case "S":
		return m.stopAgent()

	// Command mode
	case ":":
		m.mode = ModeCommand
	}

	return m, nil
}

// handleCommandMode processes keys in command mode
func (m *Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute command
		m.mode = ModeNormal
	case "esc":
		m.mode = ModeNormal
	}
	return m, nil
}

// handleConfirm processes keys in confirm dialog
func (m *Model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.showConfirm = false
		if m.confirmFn != nil {
			return m, m.confirmFn()
		}
	case "n", "N", "esc":
		m.showConfirm = false
	}
	return m, nil
}

func (m *Model) handleCreateTicketMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		title := strings.TrimSpace(m.titleInput.Value())
		if title == "" {
			m.notify("Title cannot be empty")
			return m, nil
		}
		ticket := board.NewTicket(title)
		ticket.Status = m.board.Columns[m.activeColumn].Status
		m.board.AddTicket(ticket)
		m.refreshColumnTickets()
		m.saveBoard()
		m.mode = ModeNormal
		m.titleInput.Blur()
		m.notify("Created: " + title)
		return m, nil
	case "esc":
		m.mode = ModeNormal
		m.titleInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.titleInput, cmd = m.titleInput.Update(msg)
	return m, cmd
}

// Navigation helpers
func (m *Model) moveColumn(delta int) {
	m.activeColumn += delta
	if m.activeColumn < 0 {
		m.activeColumn = 0
	}
	if m.activeColumn >= len(m.board.Columns) {
		m.activeColumn = len(m.board.Columns) - 1
	}
	m.activeTicket = 0
	m.ensureColumnVisible()
}

func (m *Model) ensureColumnVisible() {
	colWidth := m.calcColumnWidth()
	visibleCols := m.visibleColumnCount(colWidth)

	if m.activeColumn < m.scrollOffset {
		m.scrollOffset = m.activeColumn
	} else if m.activeColumn >= m.scrollOffset+visibleCols {
		m.scrollOffset = m.activeColumn - visibleCols + 1
	}

	maxOffset := len(m.board.Columns) - visibleCols
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
}

func (m *Model) calcColumnWidth() int {
	if m.width == 0 || len(m.board.Columns) == 0 {
		return minColumnWidth
	}

	numCols := len(m.board.Columns)
	totalOverhead := numCols * columnOverhead
	colWidth := (m.width - totalOverhead) / numCols

	if colWidth < minColumnWidth {
		colWidth = minColumnWidth
	}
	return colWidth
}

func (m *Model) visibleColumnCount(colWidth int) int {
	if m.width == 0 {
		return len(m.board.Columns)
	}
	visible := m.width / (colWidth + columnOverhead)
	if visible < 1 {
		visible = 1
	}
	if visible > len(m.board.Columns) {
		visible = len(m.board.Columns)
	}
	return visible
}

func (m *Model) distributeWidth(numCols int) (baseWidth, remainder int) {
	if numCols == 0 || m.width == 0 {
		return minColumnWidth, 0
	}
	// lipgloss Width() includes padding, so only border (2) and margin (1) are outside
	borders := numCols * 2
	margins := numCols - 1
	available := m.width - borders - margins
	baseWidth = available / numCols
	remainder = available % numCols
	if baseWidth < minColumnWidth {
		baseWidth = minColumnWidth
		remainder = 0
	}
	return baseWidth, remainder
}

func (m *Model) moveTicket(delta int) {
	if len(m.columnTickets) <= m.activeColumn {
		return
	}
	tickets := m.columnTickets[m.activeColumn]
	m.activeTicket += delta
	if m.activeTicket < 0 {
		m.activeTicket = 0
	}
	if m.activeTicket >= len(tickets) {
		m.activeTicket = len(tickets) - 1
		if m.activeTicket < 0 {
			m.activeTicket = 0
		}
	}
}

// Action implementations
func (m *Model) createNewTicket() (tea.Model, tea.Cmd) {
	m.mode = ModeCreateTicket
	m.titleInput.Reset()
	m.titleInput.Focus()
	return m, m.titleInput.Cursor.BlinkCmd()
}

func (m *Model) attachToAgent() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil || ticket.TmuxSession == "" {
		m.notify("No agent session for this ticket")
		return m, nil
	}

	// Detach from TUI and attach to tmux session
	return m, tea.ExecProcess(
		exec.Command("tmux", "attach-session", "-t", ticket.TmuxSession),
		func(err error) tea.Msg { return nil },
	)
}

func (m *Model) confirmDeleteTicket() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	m.showConfirm = true
	m.confirmMsg = "Delete ticket: " + ticket.Title + "?"
	m.confirmFn = func() tea.Cmd {
		m.board.DeleteTicket(ticket.ID)
		m.refreshColumnTickets()
		m.saveBoard()
		m.notify("Deleted ticket")
		return nil
	}
	return m, nil
}

func (m *Model) quickMoveTicket() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	// Move to next column
	nextStatus := m.nextStatus(ticket.Status)
	if nextStatus == ticket.Status {
		return m, nil
	}

	m.board.MoveTicket(ticket.ID, nextStatus)
	m.refreshColumnTickets()
	m.saveBoard()
	m.notify("Moved to " + string(nextStatus))

	return m, nil
}

func (m *Model) spawnAgent() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	if ticket.Status != board.StatusInProgress {
		m.notify("Move ticket to In Progress first")
		return m, nil
	}

	// Set up tmux session name
	ticket.TmuxSession = m.board.Settings.TmuxPrefix + string(ticket.ID)[:8]

	// Create worktree if needed
	if ticket.WorktreePath == "" {
		branch := m.board.Settings.BranchPrefix + string(ticket.ID)[:8]
		baseBranch, _ := m.worktreeMgr.GetDefaultBranch()

		path, err := m.worktreeMgr.CreateWorktree(branch, baseBranch)
		if err != nil {
			m.notify("Failed to create worktree: " + err.Error())
			return m, nil
		}

		ticket.WorktreePath = path
		ticket.BranchName = branch
		ticket.BaseBranch = baseBranch
	}

	// Spawn agent
	agentType := m.board.Settings.DefaultAgent
	if err := m.agentMgr.SpawnAgent(ticket, agentType); err != nil {
		m.notify("Failed to spawn agent: " + err.Error())
		return m, nil
	}

	m.saveBoard()
	m.notify("Spawned " + agentType + " agent")
	return m, nil
}

func (m *Model) stopAgent() (tea.Model, tea.Cmd) {
	ticket := m.selectedTicket()
	if ticket == nil {
		return m, nil
	}

	if err := m.agentMgr.StopAgent(ticket); err != nil {
		m.notify("Failed to stop agent: " + err.Error())
		return m, nil
	}

	m.saveBoard()
	m.notify("Agent stopped")
	return m, nil
}

// Helper methods
func (m *Model) selectedTicket() *board.Ticket {
	if len(m.columnTickets) <= m.activeColumn {
		return nil
	}
	tickets := m.columnTickets[m.activeColumn]
	if len(tickets) <= m.activeTicket {
		return nil
	}
	return tickets[m.activeTicket]
}

func (m *Model) refreshColumnTickets() {
	m.columnTickets = make([][]*board.Ticket, len(m.board.Columns))
	for i, col := range m.board.Columns {
		m.columnTickets[i] = m.board.GetTicketsByStatus(col.Status)
	}
}

func (m *Model) nextStatus(current board.TicketStatus) board.TicketStatus {
	switch current {
	case board.StatusBacklog:
		return board.StatusInProgress
	case board.StatusInProgress:
		return board.StatusDone
	default:
		return current
	}
}

func (m *Model) notify(msg string) {
	m.notification = msg
	m.notifyTime = time.Now()
}

func (m *Model) saveBoard() {
	if err := m.board.Save(m.boardDir); err != nil {
		m.notify("Failed to save: " + err.Error())
	}
}

// Messages
type agentStatusMsg time.Time
type animationMsg time.Time
type notificationMsg time.Time

// Commands
func tickAgentStatus(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return agentStatusMsg(t)
	})
}

func tickAnimation() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return animationMsg(t)
	})
}
