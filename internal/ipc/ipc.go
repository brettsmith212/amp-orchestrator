package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
)

// EventType represents the type of event being sent
type EventType string

const (
	EventTypeQueueUpdated   EventType = "queue_updated"
	EventTypeTicketEnqueued EventType = "ticket_enqueued"
	EventTypeTicketStarted  EventType = "ticket_started"
	EventTypeTicketComplete EventType = "ticket_complete"
	EventTypeWorkerStatus   EventType = "worker_status"
)

// Event represents a message sent over the IPC bus
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// QueueEvent represents queue-related events
type QueueEvent struct {
	QueueLength int            `json:"queue_length"`
	NextTicket  *ticket.Ticket `json:"next_ticket,omitempty"`
}

// TicketEvent represents ticket lifecycle events
type TicketEvent struct {
	Ticket   *ticket.Ticket `json:"ticket"`
	WorkerID int            `json:"worker_id,omitempty"`
	Message  string         `json:"message,omitempty"`
}

// WorkerStatusEvent represents worker status updates
type WorkerStatusEvent struct {
	WorkerID      int            `json:"worker_id"`
	Status        string         `json:"status"` // "idle", "working", "error"
	CurrentTicket *ticket.Ticket `json:"current_ticket,omitempty"`
	Message       string         `json:"message,omitempty"`
}

// Server represents the IPC server that publishes events
type Server struct {
	socketPath string
	listener   net.Listener
	clients    map[net.Conn]bool
	clientsMux sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewServer creates a new IPC server
func NewServer(socketPath string) *Server {
	// Expand tilde in socket path
	if strings.HasPrefix(socketPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Warning: Failed to get home directory: %v", err)
		} else {
			socketPath = filepath.Join(home, socketPath[2:])
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		socketPath: socketPath,
		clients:    make(map[net.Conn]bool),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins listening for client connections
func (s *Server) Start() error {
	// Remove existing socket file if it exists
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create directory for socket if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket: %w", err)
	}

	s.listener = listener
	log.Printf("IPC server listening on %s", s.socketPath)

	// Accept connections in a goroutine
	go s.acceptConnections()

	return nil
}

// Stop shuts down the IPC server
func (s *Server) Stop() error {
	s.cancel()

	// Close all client connections
	s.clientsMux.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clientsMux.Unlock()

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Remove socket file
	return os.Remove(s.socketPath)
}

// PublishEvent sends an event to all connected clients
func (s *Server) PublishEvent(eventType EventType, data interface{}) {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	// Add newline for easier parsing by clients
	eventJSON = append(eventJSON, '\n')

	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	// Send to all connected clients
	for conn := range s.clients {
		_, err := conn.Write(eventJSON)
		if err != nil {
			log.Printf("Failed to write to client: %v", err)
			// Remove client on write error
			go s.removeClient(conn)
		}
	}
}

// Helper methods for common events
func (s *Server) PublishQueueUpdated(queueLength int, nextTicket *ticket.Ticket) {
	s.PublishEvent(EventTypeQueueUpdated, QueueEvent{
		QueueLength: queueLength,
		NextTicket:  nextTicket,
	})
}

func (s *Server) PublishTicketEnqueued(t *ticket.Ticket) {
	s.PublishEvent(EventTypeTicketEnqueued, TicketEvent{
		Ticket:  t,
		Message: fmt.Sprintf("Ticket %s enqueued", t.ID),
	})
}

func (s *Server) PublishTicketStarted(t *ticket.Ticket, workerID int) {
	s.PublishEvent(EventTypeTicketStarted, TicketEvent{
		Ticket:   t,
		WorkerID: workerID,
		Message:  fmt.Sprintf("Worker %d started processing ticket %s", workerID, t.ID),
	})
}

func (s *Server) PublishTicketComplete(t *ticket.Ticket, workerID int) {
	s.PublishEvent(EventTypeTicketComplete, TicketEvent{
		Ticket:   t,
		WorkerID: workerID,
		Message:  fmt.Sprintf("Worker %d completed ticket %s", workerID, t.ID),
	})
}

func (s *Server) PublishWorkerStatus(workerID int, status string, currentTicket *ticket.Ticket, message string) {
	s.PublishEvent(EventTypeWorkerStatus, WorkerStatusEvent{
		WorkerID:      workerID,
		Status:        status,
		CurrentTicket: currentTicket,
		Message:       message,
	})
}

// acceptConnections handles incoming client connections
func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.ctx.Err() != nil {
					// Server is shutting down
					return
				}
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			s.addClient(conn)
		}
	}
}

// addClient adds a new client connection
func (s *Server) addClient(conn net.Conn) {
	s.clientsMux.Lock()
	s.clients[conn] = true
	s.clientsMux.Unlock()

	log.Printf("New IPC client connected: %s", conn.RemoteAddr())

	// Handle client connection in a goroutine
	go s.handleClient(conn)
}

// removeClient removes a client connection
func (s *Server) removeClient(conn net.Conn) {
	s.clientsMux.Lock()
	delete(s.clients, conn)
	s.clientsMux.Unlock()

	conn.Close()
	log.Printf("IPC client disconnected: %s", conn.RemoteAddr())
}

// handleClient manages a client connection
func (s *Server) handleClient(conn net.Conn) {
	defer s.removeClient(conn)

	// Set read timeout to detect dead connections
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// Set a read deadline to periodically check if context is cancelled
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			// Try to read from connection (clients might send keepalive)
			buf := make([]byte, 1024)
			_, err := conn.Read(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout is expected, continue
					continue
				}
				// Connection closed or other error
				return
			}
		}
	}
}

// Client represents an IPC client that receives events
type Client struct {
	socketPath string
	conn       net.Conn
	events     chan Event
	ctx        context.Context
	cancel     context.CancelFunc
	closeOnce  sync.Once
}

// NewClient creates a new IPC client
func NewClient(socketPath string) *Client {
	// Expand tilde in socket path
	if strings.HasPrefix(socketPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Warning: Failed to get home directory: %v", err)
		} else {
			socketPath = filepath.Join(home, socketPath[2:])
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		socketPath: socketPath,
		events:     make(chan Event, 100), // Buffer events
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Connect establishes connection to the IPC server
func (c *Client) Connect() error {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to unix socket: %w", err)
	}

	c.conn = conn

	// Start reading events in a goroutine
	go c.readEvents()

	return nil
}

// Events returns the channel of received events
func (c *Client) Events() <-chan Event {
	return c.events
}

// Close disconnects from the IPC server
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.cancel()
		close(c.events)

		if c.conn != nil {
			err = c.conn.Close()
		}
	})

	return err
}

// readEvents reads events from the connection
func (c *Client) readEvents() {
	defer c.Close()

	decoder := json.NewDecoder(c.conn)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var event Event
			if err := decoder.Decode(&event); err != nil {
				log.Printf("Failed to decode event: %v", err)
				return
			}

			select {
			case c.events <- event:
			case <-c.ctx.Done():
				return
			default:
				// Channel is full, drop the event
				log.Printf("Event channel full, dropping event")
			}
		}
	}
}
