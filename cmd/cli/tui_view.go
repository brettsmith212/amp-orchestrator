package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Styles for the TUI
var (
	// Panel styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Margin(1, 0)

	// Status styles
	idleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	workingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	completedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("76"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	queuedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

	// Text styles
	boldStyle = lipgloss.NewStyle().Bold(true)
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	
	// Event styles
	eventTimeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	eventTypeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
)

// View renders the TUI
func (m Model) View() string {
	if m.quitting {
		return "Goodbye! 👋\n"
	}

	// Calculate dimensions for panels
	width := m.width
	height := m.height
	
	// Set sensible defaults if dimensions not yet received
	if width == 0 {
		width = 100
	}
	if height == 0 {
		height = 30
	}
	
	if width < 80 {
		width = 80
	}
	if height < 20 {
		height = 20
	}

	// Header
	header := titleStyle.Render("🤖 Amp Orchestrator - Real-time Status")
	
	// Calculate panel dimensions
	panelWidth := (width - 6) / 2 // Account for borders and margins
	panelHeight := height - 12     // Account for header, footer, and events panel
	
	// Ensure minimum panel dimensions
	if panelWidth < 30 {
		panelWidth = 30
	}
	if panelHeight < 8 {
		panelHeight = 8
	}

	// Render tickets panel
	ticketsPanel := m.renderTicketsPanel(panelWidth, panelHeight)
	
	// Render agents panel
	agentsPanel := m.renderAgentsPanel(panelWidth, panelHeight)
	
	// Render events panel (full width, shorter)
	eventsPanel := m.renderEventsPanel(width-4, 8)
	
	// Arrange panels side by side
	topPanels := lipgloss.JoinHorizontal(lipgloss.Top, ticketsPanel, agentsPanel)
	
	// Footer with help text
	footer := dimStyle.Render("Press q or Ctrl+C to quit")
	
	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		topPanels,
		eventsPanel,
		footer,
	)
}

// renderTicketsPanel renders the tickets panel
func (m Model) renderTicketsPanel(width, height int) string {
	title := "📋 Tickets"
	
	var content strings.Builder
	
	if len(m.tickets) == 0 {
		content.WriteString(dimStyle.Render("No tickets yet..."))
	} else {
		// Show recent tickets (limit to fit in panel)
		maxTickets := height - 5 // Account for title, borders, and padding
		if maxTickets < 1 {
			maxTickets = 1
		}
		start := 0
		if len(m.tickets) > maxTickets {
			start = len(m.tickets) - maxTickets
		}
		
		for i := start; i < len(m.tickets); i++ {
			ticket := m.tickets[i]
			content.WriteString(m.renderTicketLine(ticket))
			if i < len(m.tickets)-1 {
				content.WriteString("\n")
			}
		}
	}
	
	panelContent := fmt.Sprintf("%s\n\n%s", boldStyle.Render(title), content.String())
	
	return panelStyle.
		Width(width).
		Render(panelContent)
}

// renderAgentsPanel renders the agents panel
func (m Model) renderAgentsPanel(width, height int) string {
	title := "🤖 Agents"
	
	var content strings.Builder
	
	if len(m.agents) == 0 {
		content.WriteString(dimStyle.Render("No agents connected..."))
	} else {
		for i, agent := range m.agents {
			content.WriteString(m.renderAgentLine(agent))
			if i < len(m.agents)-1 {
				content.WriteString("\n")
			}
		}
	}
	
	panelContent := fmt.Sprintf("%s\n\n%s", boldStyle.Render(title), content.String())
	
	return panelStyle.
		Width(width).
		Render(panelContent)
}

// renderEventsPanel renders the events panel
func (m Model) renderEventsPanel(width, height int) string {
	title := "📡 Recent Events"
	
	var content strings.Builder
	
	if len(m.events) == 0 {
		content.WriteString(dimStyle.Render("Waiting for events..."))
	} else {
		// Show recent events (limit to fit in panel)
		maxEvents := height - 5 // Account for title, borders, and padding
		if maxEvents < 1 {
			maxEvents = 1
		}
		start := 0
		if len(m.events) > maxEvents {
			start = len(m.events) - maxEvents
		}
		
		for i := start; i < len(m.events); i++ {
			event := m.events[i]
			content.WriteString(m.renderEventLine(event))
			if i < len(m.events)-1 {
				content.WriteString("\n")
			}
		}
	}
	
	panelContent := fmt.Sprintf("%s\n\n%s", boldStyle.Render(title), content.String())
	
	return panelStyle.
		Width(width).
		Render(panelContent)
}

// renderTicketLine renders a single ticket line
func (m Model) renderTicketLine(ticket TicketInfo) string {
	var statusIcon, statusText string
	var style lipgloss.Style
	
	switch ticket.Status {
	case "queued":
		statusIcon = "⏳"
		statusText = "Queued"
		style = queuedStyle
	case "processing":
		statusIcon = "⚙️"
		statusText = "Processing"
		style = workingStyle
	case "completed":
		statusIcon = "✅"
		statusText = "Completed"
		style = completedStyle
	default:
		statusIcon = "❓"
		statusText = "Unknown"
		style = dimStyle
	}
	
	// Format ticket line
	idPart := boldStyle.Render(ticket.ID)
	statusPart := style.Render(statusIcon + " " + statusText)
	
	// Add worker info if assigned
	workerPart := ""
	if ticket.AssignedTo > 0 {
		workerPart = dimStyle.Render(" (Worker " + strconv.Itoa(ticket.AssignedTo) + ")")
	}
	
	// Format title (truncate if too long)
	title := ticket.Title
	if len(title) > 30 {
		title = title[:27] + "..."
	}
	
	return fmt.Sprintf("%s %s%s\n  %s", 
		statusPart, 
		idPart, 
		workerPart, 
		dimStyle.Render(title))
}

// renderAgentLine renders a single agent line
func (m Model) renderAgentLine(agent AgentInfo) string {
	var statusIcon string
	var style lipgloss.Style
	
	switch agent.Status {
	case "idle":
		statusIcon = "😴"
		style = idleStyle
	case "working":
		statusIcon = "⚙️"
		style = workingStyle
	case "error":
		statusIcon = "❌"
		style = errorStyle
	default:
		statusIcon = "🤖"
		style = dimStyle
	}
	
	// Format agent line
	agentID := boldStyle.Render("Agent " + strconv.Itoa(agent.ID))
	status := style.Render(statusIcon + " " + strings.Title(agent.Status))
	
	// Current activity
	activity := ""
	if agent.CurrentTicket != nil {
		activity = "\n  " + dimStyle.Render("Working on: " + *agent.CurrentTicket)
	} else if agent.Status == "idle" {
		activity = "\n  " + dimStyle.Render("Ready for work")
	}
	
	// Last activity time
	timeSince := time.Since(agent.LastActivity)
	timeStr := ""
	if timeSince < time.Minute {
		timeStr = "now"
	} else if timeSince < time.Hour {
		timeStr = fmt.Sprintf("%dm ago", int(timeSince.Minutes()))
	} else {
		timeStr = agent.LastActivity.Format("15:04")
	}
	
	return fmt.Sprintf("%s %s\n  %s%s", 
		status, 
		agentID, 
		dimStyle.Render("Last activity: " + timeStr),
		activity)
}

// renderEventLine renders a single event line
func (m Model) renderEventLine(event EventInfo) string {
	timestamp := eventTimeStyle.Render(event.Timestamp.Format("15:04:05"))
	eventType := eventTypeStyle.Render("[" + event.Type + "]")
	message := event.Message
	
	// Truncate long messages
	if len(message) > 60 {
		message = message[:57] + "..."
	}
	
	return fmt.Sprintf("%s %s %s", timestamp, eventType, message)
}