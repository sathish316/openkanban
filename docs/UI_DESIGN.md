# UI Design & TUI Wireframes

This document defines the visual design, component hierarchy, and styling for Agent Board's terminal user interface.

## Design Philosophy

- **Information density**: Show maximum useful info without clutter
- **Vim-native**: hjkl navigation, modal editing, command mode
- **Status at a glance**: Agent states visible via color/icons
- **Keyboard-first**: Every action has a keybinding

## Main Board View

### Wireframe: Default Layout

```
┌─ Agent Board ─────────────────────────────────────────────────────────────────┐
│ myproject (~/projects/myproject)                              ? help  q quit │
├───────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌─ Backlog (4) ──────┐  ┌─ In Progress (2/3) ┐  ┌─ Done (7) ─────────┐     │
│  │                    │  │                    │  │                    │     │
│  │ ┌────────────────┐ │  │ ┌────────────────┐ │  │ ┌────────────────┐ │     │
│  │ │ #3 Add user    │ │  │ │▶#1 Auth system │ │  │ │ #5 Setup CI/CD │ │     │
│  │ │ profile page   │ │  │ │ ● Working      │ │  │ │ ✓ Completed    │ │     │
│  │ │ [frontend]     │ │  │ │ [backend,auth] │ │  │ │ [devops]       │ │     │
│  │ └────────────────┘ │  │ └────────────────┘ │  │ └────────────────┘ │     │
│  │                    │  │                    │  │                    │     │
│  │ ┌────────────────┐ │  │ ┌────────────────┐ │  │ ┌────────────────┐ │     │
│  │ │ #4 API rate    │ │  │ │▶#2 Database    │ │  │ │ #6 Add README  │ │     │
│  │ │ limiting       │ │  │ │ ○ Idle         │ │  │ │ ✓ Completed    │ │     │
│  │ │ [backend]      │ │  │ │ [backend,db]   │ │  │ │ [docs]         │ │     │
│  │ └────────────────┘ │  │ └────────────────┘ │  │ └────────────────┘ │     │
│  │                    │  │                    │  │                    │     │
│  │ ┌────────────────┐ │  │                    │  │ ┌────────────────┐ │     │
│  │ │ #7 Write tests │ │  │                    │  │ │ #8 Fix login   │ │     │
│  │ │                │ │  │                    │  │ │ ✓ Completed    │ │     │
│  │ │ [testing]      │ │  │                    │  │ │ [bugfix]       │ │     │
│  │ └────────────────┘ │  │                    │  │ └────────────────┘ │     │
│  │                    │  │                    │  │                    │     │
│  │ ┌────────────────┐ │  │                    │  │        ...         │     │
│  │ │ #9 Refactor    │ │  │                    │  │   (+4 more)        │     │
│  │ │ auth module    │ │  │                    │  │                    │     │
│  │ │ [refactor]     │ │  │                    │  │                    │     │
│  │ └────────────────┘ │  │                    │  │                    │     │
│  │                    │  │                    │  │                    │     │
│  └────────────────────┘  └────────────────────┘  └────────────────────┘     │
│                                                                               │
├───────────────────────────────────────────────────────────────────────────────┤
│ NORMAL │ j/k: navigate │ h/l: columns │ Enter: attach │ n: new │ m: move     │
└───────────────────────────────────────────────────────────────────────────────┘
```

### Wireframe: Ticket Selected (Highlighted)

```
  ┌─ In Progress (2/3) ┐
  │                    │
  │ ╔════════════════╗ │   ← Selected ticket has double border
  │ ║▶#1 Auth system ║ │
  │ ║ ● Working      ║ │
  │ ║ [backend,auth] ║ │
  │ ╚════════════════╝ │
  │                    │
  │ ┌────────────────┐ │
  │ │▶#2 Database    │ │
  │ │ ○ Idle         │ │
```

### Wireframe: Compact Mode (Small Terminal)

```
┌─ Agent Board ── myproject ──────────────────────────┐
├─────────────────────────────────────────────────────┤
│ Backlog(4)     In Progress(2)  Done(7)             │
│ ────────────   ───────────────  ─────────          │
│ #3 Profile     ▶#1 Auth ●      #5 CI/CD ✓         │
│ #4 Rate limit  ▶#2 DB ○        #6 README ✓        │
│ #7 Tests                        #8 Login ✓        │
│ #9 Refactor                     (+4)              │
├─────────────────────────────────────────────────────┤
│ j/k:nav h/l:col Enter:attach n:new m:move ?:help   │
└─────────────────────────────────────────────────────┘
```

## Component Hierarchy

```
App
├── Header
│   ├── Logo/Title
│   ├── BoardName
│   ├── RepoPath
│   └── QuickHelp (? q)
│
├── BoardView
│   ├── Column (repeated)
│   │   ├── ColumnHeader
│   │   │   ├── Title
│   │   │   ├── Count
│   │   │   └── WIPIndicator
│   │   │
│   │   └── TicketList
│   │       └── TicketCard (repeated)
│   │           ├── ID
│   │           ├── Title
│   │           ├── AgentIndicator
│   │           ├── StatusBadge
│   │           └── Labels
│   │
│   └── ScrollIndicator
│
├── StatusBar
│   ├── Mode
│   ├── KeyHints
│   └── Notifications
│
└── Overlays (conditional)
    ├── HelpModal
    ├── TicketDetailModal
    ├── NewTicketForm
    ├── CommandPalette
    └── ConfirmDialog
```

## Ticket Card States

### Agent Status Indicators

| Status | Icon | Color | Description |
|--------|------|-------|-------------|
| None | (none) | Gray | No agent spawned |
| Idle | ○ | Blue | Session exists, no activity |
| Working | ● | Yellow (animated) | Active processing |
| Waiting | ◐ | Magenta | Awaiting user input |
| Completed | ✓ | Green | Agent finished |
| Error | ✗ | Red | Agent crashed |

### Card Variations

```
Standard (backlog):          With Agent (in progress):     Completed:
┌────────────────┐           ┌────────────────┐           ┌────────────────┐
│ #3 Add profile │           │▶#1 Auth system │           │ #5 Setup CI/CD │
│                │           │ ● Working      │           │ ✓ Completed    │
│ [frontend]     │           │ [backend,auth] │           │ [devops]       │
└────────────────┘           └────────────────┘           └────────────────┘

Selected:                    Error state:                  High priority:
╔════════════════╗           ┌────────────────┐           ┌────────────────┐
║ #3 Add profile ║           │▶#4 API rate    │           │!#7 Critical    │
║                ║           │ ✗ Error        │           │                │
║ [frontend]     ║           │ [backend]      │           │ [urgent]       │
╚════════════════╝           └────────────────┘           └────────────────┘
```

## Modal Dialogs

### New Ticket Form

```
┌─ New Ticket ──────────────────────────────────────────┐
│                                                        │
│  Title:                                                │
│  ┌──────────────────────────────────────────────────┐ │
│  │ Implement user authentication                    │ │
│  └──────────────────────────────────────────────────┘ │
│                                                        │
│  Description:                                          │
│  ┌──────────────────────────────────────────────────┐ │
│  │ Add JWT-based auth to the API with:              │ │
│  │ - Login/logout endpoints                         │ │
│  │ - Token refresh                                  │ │
│  │ - Role-based permissions                         │ │
│  └──────────────────────────────────────────────────┘ │
│                                                        │
│  Labels: [backend] [auth] [+]                          │
│                                                        │
│  Priority: ● High  ○ Medium  ○ Low                     │
│                                                        │
│  Agent: [claude ▼]  ☑ Auto-spawn when started         │
│                                                        │
│  ┌─────────┐  ┌──────────┐                            │
│  │ Create  │  │  Cancel  │                            │
│  └─────────┘  └──────────┘                            │
│                                                        │
│  Tab: next field │ Shift+Tab: prev │ Enter: submit    │
└────────────────────────────────────────────────────────┘
```

### Ticket Detail View

```
┌─ Ticket #1 ─────────────────────────────────────────────────────────────────┐
│                                                                              │
│  Auth system implementation                                     [In Progress]│
│  ═══════════════════════════════════════════════════════════════════════════│
│                                                                              │
│  Description:                                                                │
│  Add JWT-based authentication to the API including login, logout,           │
│  token refresh, and role-based permissions.                                 │
│                                                                              │
│  ───────────────────────────────────────────────────────────────────────────│
│                                                                              │
│  Agent: claude           Status: ● Working                                  │
│  Branch: agent/auth-1    Session: ab-ticket-1                               │
│  Worktree: ~/projects/myproject-worktrees/auth-1                           │
│                                                                              │
│  ───────────────────────────────────────────────────────────────────────────│
│                                                                              │
│  Labels: [backend] [auth] [security]                                        │
│  Priority: High                                                              │
│  Created: 2025-01-15 10:30                                                  │
│  Started: 2025-01-16 09:00 (5h 30m ago)                                     │
│                                                                              │
│  ───────────────────────────────────────────────────────────────────────────│
│                                                                              │
│  [Enter] Attach to session  [e] Edit  [m] Move  [d] Delete  [Esc] Close    │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Help Modal

```
┌─ Keyboard Shortcuts ────────────────────────────────────────────────────────┐
│                                                                              │
│  Navigation                      Actions                                    │
│  ──────────────────────────────  ────────────────────────────────────────  │
│  h/l      Move between columns   n         New ticket                       │
│  j/k      Move between tickets   Enter     Attach to agent session          │
│  g        Go to first ticket     e         Edit ticket                      │
│  G        Go to last ticket      d         Delete ticket (with confirm)     │
│  1-9      Jump to column N       m         Move ticket (then h/l)           │
│                                  Space     Quick move to next column        │
│  Views                                                                      │
│  ──────────────────────────────  Agent                                      │
│  Tab      Cycle focus            ────────────────────────────────────────   │
│  /        Search/filter          s         Spawn/restart agent              │
│  Esc      Clear filter/cancel    S         Stop agent                       │
│  ?        Toggle help            r         Refresh agent status             │
│  q        Quit                   a         Change agent type                │
│                                                                              │
│  Command Mode                    Git                                        │
│  ──────────────────────────────  ────────────────────────────────────────   │
│  :        Enter command mode     p         Create PR from ticket            │
│  :w       Save board             b         Show git branch                  │
│  :q       Quit                   c         Show recent commits              │
│  :board   Switch board                                                      │
│                                                                              │
│                                                      Press any key to close │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Confirmation Dialog

```
                    ┌─ Confirm ─────────────────────────┐
                    │                                    │
                    │  Delete ticket #3?                 │
                    │                                    │
                    │  This will also:                   │
                    │  • Kill tmux session ab-ticket-3   │
                    │  • Remove worktree (optional)      │
                    │                                    │
                    │  ☐ Also delete git worktree        │
                    │  ☐ Also delete branch              │
                    │                                    │
                    │    [y] Yes    [n] No    [Esc]     │
                    │                                    │
                    └────────────────────────────────────┘
```

## Color Scheme (Catppuccin Mocha)

### Base Colors

```go
var colors = struct {
    // Base
    Base     lipgloss.Color // #1e1e2e - Main background
    Mantle   lipgloss.Color // #181825 - Darker background
    Crust    lipgloss.Color // #11111b - Darkest background
    Surface0 lipgloss.Color // #313244 - Surface
    Surface1 lipgloss.Color // #45475a - Lighter surface
    Surface2 lipgloss.Color // #585b70 - Even lighter
    
    // Text
    Text     lipgloss.Color // #cdd6f4 - Main text
    Subtext0 lipgloss.Color // #a6adc8 - Dimmed text
    Subtext1 lipgloss.Color // #bac2de - Less dimmed
    Overlay0 lipgloss.Color // #6c7086 - Muted text
    
    // Accent colors
    Blue     lipgloss.Color // #89b4fa - Primary accent
    Green    lipgloss.Color // #a6e3a1 - Success
    Yellow   lipgloss.Color // #f9e2af - Warning/Working
    Red      lipgloss.Color // #f38ba8 - Error
    Mauve    lipgloss.Color // #cba6f7 - Purple accent
    Peach    lipgloss.Color // #fab387 - Orange accent
    Teal     lipgloss.Color // #94e2d5 - Teal accent
    Pink     lipgloss.Color // #f5c2e7 - Pink accent
}{
    Base:     lipgloss.Color("#1e1e2e"),
    Mantle:   lipgloss.Color("#181825"),
    Crust:    lipgloss.Color("#11111b"),
    Surface0: lipgloss.Color("#313244"),
    Surface1: lipgloss.Color("#45475a"),
    Surface2: lipgloss.Color("#585b70"),
    Text:     lipgloss.Color("#cdd6f4"),
    Subtext0: lipgloss.Color("#a6adc8"),
    Subtext1: lipgloss.Color("#bac2de"),
    Overlay0: lipgloss.Color("#6c7086"),
    Blue:     lipgloss.Color("#89b4fa"),
    Green:    lipgloss.Color("#a6e3a1"),
    Yellow:   lipgloss.Color("#f9e2af"),
    Red:      lipgloss.Color("#f38ba8"),
    Mauve:    lipgloss.Color("#cba6f7"),
    Peach:    lipgloss.Color("#fab387"),
    Teal:     lipgloss.Color("#94e2d5"),
    Pink:     lipgloss.Color("#f5c2e7"),
}
```

### Semantic Color Mapping

```go
var semantic = struct {
    // Column headers
    ColumnBacklog    lipgloss.Color // Blue
    ColumnInProgress lipgloss.Color // Yellow
    ColumnDone       lipgloss.Color // Green
    
    // Agent status
    AgentIdle      lipgloss.Color // Blue
    AgentWorking   lipgloss.Color // Yellow
    AgentWaiting   lipgloss.Color // Mauve
    AgentCompleted lipgloss.Color // Green
    AgentError     lipgloss.Color // Red
    
    // Priority
    PriorityHigh   lipgloss.Color // Red
    PriorityMedium lipgloss.Color // Yellow
    PriorityLow    lipgloss.Color // Text (default)
    
    // UI elements
    Selected lipgloss.Color // Blue
    Border   lipgloss.Color // Surface1
    Dimmed   lipgloss.Color // Overlay0
}{
    ColumnBacklog:    colors.Blue,
    ColumnInProgress: colors.Yellow,
    ColumnDone:       colors.Green,
    
    AgentIdle:      colors.Blue,
    AgentWorking:   colors.Yellow,
    AgentWaiting:   colors.Mauve,
    AgentCompleted: colors.Green,
    AgentError:     colors.Red,
    
    PriorityHigh:   colors.Red,
    PriorityMedium: colors.Yellow,
    PriorityLow:    colors.Text,
    
    Selected: colors.Blue,
    Border:   colors.Surface1,
    Dimmed:   colors.Overlay0,
}
```

## Lipgloss Style Definitions

```go
package styles

import "github.com/charmbracelet/lipgloss"

// Layout styles
var (
    App = lipgloss.NewStyle().
        Background(colors.Base)
    
    Header = lipgloss.NewStyle().
        Foreground(colors.Text).
        Background(colors.Mantle).
        Padding(0, 1).
        Bold(true)
    
    StatusBar = lipgloss.NewStyle().
        Foreground(colors.Subtext0).
        Background(colors.Mantle).
        Padding(0, 1)
)

// Column styles
var (
    Column = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(colors.Surface1).
        Padding(0, 1).
        MarginRight(1)
    
    ColumnHeader = lipgloss.NewStyle().
        Bold(true).
        Padding(0, 1).
        MarginBottom(1)
    
    ColumnHeaderBacklog = ColumnHeader.Copy().
        Foreground(colors.Blue)
    
    ColumnHeaderInProgress = ColumnHeader.Copy().
        Foreground(colors.Yellow)
    
    ColumnHeaderDone = ColumnHeader.Copy().
        Foreground(colors.Green)
)

// Ticket card styles
var (
    TicketCard = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(colors.Surface0).
        Padding(0, 1).
        MarginBottom(1).
        Width(36)
    
    TicketCardSelected = TicketCard.Copy().
        Border(lipgloss.DoubleBorder()).
        BorderForeground(colors.Blue)
    
    TicketTitle = lipgloss.NewStyle().
        Foreground(colors.Text).
        Bold(true)
    
    TicketID = lipgloss.NewStyle().
        Foreground(colors.Subtext0)
    
    TicketLabel = lipgloss.NewStyle().
        Foreground(colors.Mantle).
        Background(colors.Surface2).
        Padding(0, 1)
)

// Agent status styles
var (
    AgentIndicator = lipgloss.NewStyle().
        MarginRight(1)
    
    AgentIdle = AgentIndicator.Copy().
        Foreground(colors.Blue)
    
    AgentWorking = AgentIndicator.Copy().
        Foreground(colors.Yellow).
        Blink(true)  // If terminal supports
    
    AgentWaiting = AgentIndicator.Copy().
        Foreground(colors.Mauve)
    
    AgentCompleted = AgentIndicator.Copy().
        Foreground(colors.Green)
    
    AgentError = AgentIndicator.Copy().
        Foreground(colors.Red)
)

// Modal styles
var (
    Modal = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(colors.Blue).
        Background(colors.Mantle).
        Padding(1, 2)
    
    ModalTitle = lipgloss.NewStyle().
        Foreground(colors.Blue).
        Bold(true).
        MarginBottom(1)
    
    ModalButton = lipgloss.NewStyle().
        Foreground(colors.Text).
        Background(colors.Surface0).
        Padding(0, 2)
    
    ModalButtonFocused = ModalButton.Copy().
        Background(colors.Blue).
        Foreground(colors.Mantle)
)

// Input styles
var (
    Input = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(colors.Surface1).
        Padding(0, 1)
    
    InputFocused = Input.Copy().
        BorderForeground(colors.Blue)
    
    InputLabel = lipgloss.NewStyle().
        Foreground(colors.Subtext0).
        MarginBottom(1)
)
```

## Responsive Behavior

### Terminal Size Breakpoints

| Width | Layout | Adjustments |
|-------|--------|-------------|
| <80 | Compact | Single column, abbreviated cards |
| 80-120 | Standard | 3 columns, full cards |
| 120-160 | Wide | 3 columns + expanded cards with description |
| >160 | Ultra-wide | Optional 4th column (Archive) or wider cards |

### Height Adjustments

| Height | Behavior |
|--------|----------|
| <20 | Hide descriptions, minimal chrome |
| 20-40 | Standard layout |
| >40 | More visible tickets, expanded details |

### Resize Handler

```go
func (m Model) handleResize(width, height int) Model {
    m.width = width
    m.height = height
    
    // Calculate column widths
    if width < 80 {
        m.columnWidth = width - 4
        m.visibleColumns = 1
    } else if width < 120 {
        m.columnWidth = (width - 8) / 3
        m.visibleColumns = 3
    } else {
        m.columnWidth = 40
        m.visibleColumns = 3
    }
    
    // Calculate visible tickets per column
    cardHeight := 4 // title + status + labels + margin
    headerHeight := 4 // app header + column header
    footerHeight := 2 // status bar
    m.ticketsPerColumn = (height - headerHeight - footerHeight) / cardHeight
    
    return m
}
```

## Animation Considerations

### Agent Working Animation

Since terminal blink support is inconsistent, implement via polling:

```go
type agentAnimationMsg time.Time

func tickAgentAnimation() tea.Cmd {
    return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
        return agentAnimationMsg(t)
    })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case agentAnimationMsg:
        m.animationFrame = (m.animationFrame + 1) % 4
        return m, tickAgentAnimation()
    }
    // ...
}

// Render working indicator with animation
func (m Model) renderAgentWorking() string {
    frames := []string{"●", "◐", "○", "◑"}
    return AgentWorking.Render(frames[m.animationFrame])
}
```

### Smooth Scrolling

Use viewport component from Bubbles:

```go
import "github.com/charmbracelet/bubbles/viewport"

type Column struct {
    viewport viewport.Model
    tickets  []Ticket
}
```

## Accessibility Considerations

1. **High contrast mode**: Alternative color scheme with higher contrast ratios
2. **No color-only indicators**: Icons accompany all status colors
3. **Screen reader hints**: Status bar announces current position
4. **Configurable refresh rate**: Reduce animations for sensitive users

```go
// High contrast alternative
var highContrastColors = struct {
    Background lipgloss.Color
    Text       lipgloss.Color
    Selected   lipgloss.Color
    // ...
}{
    Background: lipgloss.Color("#000000"),
    Text:       lipgloss.Color("#ffffff"),
    Selected:   lipgloss.Color("#00ff00"),
}
```
