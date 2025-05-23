package queue

import (
	"container/heap"

	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
)

// ticketHeap implements heap.Interface for tickets based on priority
// Lower priority numbers (1) have higher precedence than higher numbers (5)
type ticketHeap []*ticket.Ticket

func (h ticketHeap) Len() int { return len(h) }

func (h ticketHeap) Less(i, j int) bool {
	// Priority 1 is highest, priority 5 is lowest
	// So we want smaller priority numbers to come first
	if h[i].Priority != h[j].Priority {
		return h[i].Priority < h[j].Priority
	}
	// If priorities are equal, use creation time (FIFO within same priority)
	return h[i].CreatedAt.Before(h[j].CreatedAt)
}

func (h ticketHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *ticketHeap) Push(x interface{}) {
	*h = append(*h, x.(*ticket.Ticket))
}

func (h *ticketHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// peek returns the highest priority ticket without removing it
func (h ticketHeap) peek() *ticket.Ticket {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}

// newTicketHeap creates a new ticket heap
func newTicketHeap() *ticketHeap {
	h := &ticketHeap{}
	heap.Init(h)
	return h
}