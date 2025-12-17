package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/techdufus/openkanban/internal/board"
)

// View implements tea.Model
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Board
	b.WriteString(m.renderBoard())

	// Overlays
	if m.showHelp {
		return m.renderHelp()
	}
	if m.showConfirm {
		return m.renderWithOverlay(b.String(), m.renderConfirmDialog())
	}
	if m.mode == ModeCreateTicket {
		return m.renderWithOverlay(b.String(), m.renderCreateTicketForm())
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderHeader renders the top header bar
func (m *Model) renderHeader() string {
	title := headerStyle.Render("OpenKanban")
	boardName := subtitleStyle.Render(m.board.Name)
	repoPath := dimStyle.Render("(" + m.board.RepoPath + ")")

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, " ", boardName, " ", repoPath)

	help := dimStyle.Render("? help  q quit")

	// Calculate spacing
	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(help)
	if spacing < 0 {
		spacing = 0
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, left, strings.Repeat(" ", spacing), help)
}

// renderBoard renders the kanban columns
func (m *Model) renderBoard() string {
	columnWidth := m.calcColumnWidth()
	visibleCols := m.visibleColumnCount(columnWidth)

	startCol := m.scrollOffset
	endCol := startCol + visibleCols
	if endCol > len(m.board.Columns) {
		endCol = len(m.board.Columns)
	}

	numVisible := endCol - startCol
	baseWidth, remainder := m.distributeWidth(numVisible)

	var columns []string

	if startCol > 0 {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Render("◀")
		columns = append(columns, indicator)
	}

	for i := startCol; i < endCol; i++ {
		col := m.board.Columns[i]
		isActive := i == m.activeColumn
		isLast := i == endCol-1

		colWidth := baseWidth
		if i-startCol < remainder {
			colWidth++
		}

		columns = append(columns, m.renderColumn(col, m.columnTickets[i], isActive, colWidth, isLast))
	}

	if endCol < len(m.board.Columns) {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Render("▶")
		columns = append(columns, indicator)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

// renderColumn renders a single kanban column
func (m *Model) renderColumn(col board.Column, tickets []*board.Ticket, isActive bool, width int, isLast bool) string {
	// Column header
	headerColor := lipgloss.Color(col.Color)
	header := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true).
		Render(fmt.Sprintf("%s (%d)", col.Name, len(tickets)))

	// WIP limit indicator
	if col.Limit > 0 {
		header += dimStyle.Render(fmt.Sprintf("/%d", col.Limit))
	}

	// Tickets
	var ticketViews []string
	for i, ticket := range tickets {
		isSelected := isActive && i == m.activeTicket
		ticketViews = append(ticketViews, m.renderTicket(ticket, isSelected, width-4, col.Color))
	}

	ticketsView := strings.Join(ticketViews, "\n")
	if len(tickets) == 0 {
		ticketsView = dimStyle.Render("  (empty)")
	}

	// Column container
	content := lipgloss.JoinVertical(lipgloss.Left, header, "", ticketsView)

	borderColor := lipgloss.Color("#313244")
	if isActive {
		borderColor = headerColor
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Padding(0, 1)

	if !isLast {
		style = style.MarginRight(1)
	}

	return style.Render(content)
}

// renderTicket renders a single ticket card
func (m *Model) renderTicket(ticket *board.Ticket, isSelected bool, width int, columnColor string) string {
	var statusIcon string
	switch ticket.AgentStatus {
	case board.AgentIdle:
		statusIcon = agentIdleStyle.Render("○")
	case board.AgentWorking:
		frames := []string{"●", "◐", "○", "◑"}
		statusIcon = agentWorkingStyle.Render(frames[m.animationFrame])
	case board.AgentWaiting:
		statusIcon = agentWaitingStyle.Render("◐")
	case board.AgentCompleted:
		statusIcon = agentCompletedStyle.Render("✓")
	case board.AgentError:
		statusIcon = agentErrorStyle.Render("✗")
	}

	sessionIndicator := ""
	if ticket.TmuxSession != "" {
		sessionIndicator = "▶ "
	}

	idStr := dimStyle.Render(fmt.Sprintf("#%s", string(ticket.ID)[:4]))
	headerLine := fmt.Sprintf("%s%s %s", sessionIndicator, idStr, statusIcon)

	titleStyle := lipgloss.NewStyle().Width(width).Inline(false)
	wrappedTitle := titleStyle.Render(ticket.Title)

	statusLine := ""
	if ticket.AgentStatus != board.AgentNone {
		statusLine = dimStyle.Render(string(ticket.AgentStatus))
	}

	var labelParts []string
	for _, label := range ticket.Labels {
		labelParts = append(labelParts, labelStyle.Render(label))
	}
	labelsLine := strings.Join(labelParts, " ")

	lines := []string{headerLine, wrappedTitle}
	if statusLine != "" {
		lines = append(lines, statusLine)
	}
	if labelsLine != "" {
		lines = append(lines, labelsLine)
	}

	content := strings.Join(lines, "\n")

	// Card style
	borderColor := lipgloss.Color(columnColor)
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		MarginBottom(1).
		Width(width)
	if isSelected {
		cardStyle = cardStyle.Border(lipgloss.DoubleBorder())
	}

	return cardStyle.Render(content)
}

// renderStatusBar renders the bottom status bar
func (m *Model) renderStatusBar() string {
	modeStr := modeStyle.Render(string(m.mode))
	hints := dimStyle.Render("h/l: columns │ n: new │ Space: move")

	notif := ""
	if m.notification != "" {
		notif = notificationStyle.Render(m.notification)
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, modeStr, " │ ", hints)
	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(notif)
	if spacing < 0 {
		spacing = 0
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, left, strings.Repeat(" ", spacing), notif)
}

// renderHelp renders the help overlay
func (m *Model) renderHelp() string {
	help := `
 Keyboard Shortcuts

 Navigation                     Actions
 ──────────────────────────     ────────────────────────────
 h/l     Move between columns   n       New ticket
 j/k     Move between tickets   Enter   Attach to agent session
 g       Go to first ticket     d       Delete ticket
 G       Go to last ticket      Space   Quick move to next column

 Agent                          Other
 ──────────────────────────     ────────────────────────────
 s       Spawn agent            ?       Toggle help
 S       Stop agent             :       Command mode
 r       Refresh status         q       Quit

                                        Press any key to close
`

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#89b4fa")).
		Padding(1, 2).
		Render(help)
}

// renderConfirmDialog renders a confirmation dialog
func (m *Model) renderConfirmDialog() string {
	content := fmt.Sprintf(`
  %s

  [y] Yes    [n] No    [Esc] Cancel
`, m.confirmMsg)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#f38ba8")).
		Padding(1, 2).
		Render(content)
}

func (m *Model) renderCreateTicketForm() string {
	content := fmt.Sprintf(`
  New Ticket

  Title: %s

  [Enter] Create    [Esc] Cancel
`, m.titleInput.View())

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#a6e3a1")).
		Padding(1, 2).
		Render(content)
}

// renderWithOverlay renders content with a centered overlay
func (m *Model) renderWithOverlay(background, overlay string) string {
	// Simple overlay - just return overlay for now
	// TODO: Proper overlay compositing
	return overlay
}

// Styles (Catppuccin Mocha)
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cdd6f4")).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#89b4fa"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086"))

	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1e1e2e")).
			Background(lipgloss.Color("#89b4fa")).
			Padding(0, 1)

	notificationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6e3a1"))

	ticketCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#313244")).
			Padding(0, 1).
			MarginBottom(1)

	ticketCardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("#89b4fa")).
				Padding(0, 1).
				MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1e1e2e")).
			Background(lipgloss.Color("#585b70")).
			Padding(0, 1)

	agentIdleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#89b4fa"))

	agentWorkingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f9e2af"))

	agentWaitingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cba6f7"))

	agentCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6e3a1"))

	agentErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f38ba8"))
)
