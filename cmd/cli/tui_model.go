package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/brettsmith212/amp-orchestrator/internal/ipc"
)

// Model represents the TUI application state
type Model struct {
	tickets   []TicketInfo
	agents    []AgentInfo
	events    []EventInfo
	ipcClient *ipc.Client
	quitting  bool
	width     int
	height    int
}

// TicketInfo represents a ticket in the UI
type TicketInfo struct {
	ID          string
	Title       string
	Priority    int
	Status      string // "queued", "processing", "completed"
	AssignedTo  int    // Worker ID, 0 if not assigned
	EnqueuedAt  time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// AgentInfo represents an agent/worker in the UI
type AgentInfo struct {
	ID            int
	Status        string // "idle", "working", "error"
	CurrentTicket *string
	LastActivity  time.Time
	Message       string
}

// EventInfo represents a recent event
type EventInfo struct {
	Timestamp time.Time
	Type      string
	Message   string
}

// eventMsg wraps IPC events for Bubble Tea
type eventMsg struct {
	event ipc.Event
}

// tickMsg is sent periodically to update the UI
type tickMsg time.Time

// NewModel creates a new TUI model
func NewModel(client *ipc.Client) Model {
	return Model{
		tickets:   make([]TicketInfo, 0),
		agents:    make([]AgentInfo, 0),
		events:    make([]EventInfo, 0),
		ipcClient: client,
		quitting:  false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		listenForEvents(m.ipcClient),
		tickCmd(),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case eventMsg:
		m = m.handleIPCEvent(msg.event)
		return m, listenForEvents(m.ipcClient)

	case tickMsg:
		// Clean up old events (keep last 50)
		if len(m.events) > 50 {
			m.events = m.events[len(m.events)-50:]
		}
		return m, tickCmd()

	case tea.QuitMsg:
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// handleIPCEvent processes incoming IPC events and updates the model
func (m Model) handleIPCEvent(event ipc.Event) Model {
	timestamp := event.Timestamp
	
	// Add to events log
	eventInfo := EventInfo{
		Timestamp: timestamp,
		Type:      string(event.Type),
	}

	switch event.Type {
	case ipc.EventTypeQueueUpdated:
		if queueEvent, ok := event.Data.(map[string]interface{}); ok {
			queueLength := int(queueEvent["queue_length"].(float64))
			eventInfo.Message = formatQueueMessage(queueLength)
		}

	case ipc.EventTypeTicketEnqueued:
		if ticketEvent, ok := event.Data.(map[string]interface{}); ok {
			if ticket, ok := ticketEvent["ticket"].(map[string]interface{}); ok {
				ticketInfo := TicketInfo{
					ID:         ticket["id"].(string),
					Title:      ticket["title"].(string),
					Priority:   int(ticket["priority"].(float64)),
					Status:     "queued",
					EnqueuedAt: timestamp,
				}
				m.tickets = append(m.tickets, ticketInfo)
				eventInfo.Message = formatTicketEnqueuedMessage(ticketInfo)
			}
		}

	case ipc.EventTypeTicketStarted:
		if ticketEvent, ok := event.Data.(map[string]interface{}); ok {
			if ticket, ok := ticketEvent["ticket"].(map[string]interface{}); ok {
				ticketID := ticket["id"].(string)
				workerID := int(ticketEvent["worker_id"].(float64))
				
				// Update ticket status
				for i := range m.tickets {
					if m.tickets[i].ID == ticketID {
						m.tickets[i].Status = "processing"
						m.tickets[i].AssignedTo = workerID
						m.tickets[i].StartedAt = &timestamp
						break
					}
				}
				
				eventInfo.Message = formatTicketStartedMessage(ticketID, workerID)
			}
		}

	case ipc.EventTypeTicketComplete:
		if ticketEvent, ok := event.Data.(map[string]interface{}); ok {
			if ticket, ok := ticketEvent["ticket"].(map[string]interface{}); ok {
				ticketID := ticket["id"].(string)
				workerID := int(ticketEvent["worker_id"].(float64))
				
				// Update ticket status
				for i := range m.tickets {
					if m.tickets[i].ID == ticketID {
						m.tickets[i].Status = "completed"
						m.tickets[i].CompletedAt = &timestamp
						break
					}
				}
				
				eventInfo.Message = formatTicketCompleteMessage(ticketID, workerID)
			}
		}

	case ipc.EventTypeWorkerStatus:
		if workerEvent, ok := event.Data.(map[string]interface{}); ok {
			workerID := int(workerEvent["worker_id"].(float64))
			status := workerEvent["status"].(string)
			message := workerEvent["message"].(string)
			
			// Update or create agent info
			agentFound := false
			for i := range m.agents {
				if m.agents[i].ID == workerID {
					m.agents[i].Status = status
					m.agents[i].Message = message
					m.agents[i].LastActivity = timestamp
					
					if status == "idle" {
						m.agents[i].CurrentTicket = nil
					} else if status == "working" {
						// Find current ticket for this worker
						for _, t := range m.tickets {
							if t.AssignedTo == workerID && t.Status == "processing" {
								m.agents[i].CurrentTicket = &t.ID
								break
							}
						}
					}
					agentFound = true
					break
				}
			}
			
			if !agentFound {
				agent := AgentInfo{
					ID:           workerID,
					Status:       status,
					Message:      message,
					LastActivity: timestamp,
				}
				m.agents = append(m.agents, agent)
			}
			
			eventInfo.Message = formatWorkerStatusMessage(workerID, status, message)
		}
	}

	// Add to events log
	m.events = append(m.events, eventInfo)
	
	return m
}

// listenForEvents creates a command to listen for the next IPC event
func listenForEvents(client *ipc.Client) tea.Cmd {
	return func() tea.Msg {
		select {
		case event := <-client.Events():
			return eventMsg{event: event}
		}
	}
}

// tickCmd creates a command for periodic updates
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Formatting helper functions
func formatQueueMessage(queueLength int) string {
	if queueLength == 0 {
		return "Queue is empty"
	}
	return formatPlural(queueLength, "ticket", "tickets") + " in queue"
}

func formatTicketEnqueuedMessage(ticket TicketInfo) string {
	return "Enqueued: " + ticket.ID + " - " + ticket.Title
}

func formatTicketStartedMessage(ticketID string, workerID int) string {
	return formatWorker(workerID) + " started: " + ticketID
}

func formatTicketCompleteMessage(ticketID string, workerID int) string {
	return formatWorker(workerID) + " completed: " + ticketID
}

func formatWorkerStatusMessage(workerID int, status, message string) string {
	return formatWorker(workerID) + " " + status + ": " + message
}

func formatWorker(id int) string {
	return "Worker " + formatInt(id)
}

func formatInt(n int) string {
	return fmt.Sprintf("%d", n)
}

func formatPlural(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return formatInt(count) + " " + plural
}