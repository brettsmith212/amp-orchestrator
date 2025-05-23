package queue

import (
	"container/heap"
	"fmt"
	"sync"

	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
)

// Queue represents a thread-safe priority queue for tickets
type Queue struct {
	heap *ticketHeap
	mu   sync.RWMutex
}

// New creates a new priority queue
func New() *Queue {
	return &Queue{
		heap: newTicketHeap(),
	}
}

// Push adds a ticket to the queue with priority ordering
func (q *Queue) Push(t *ticket.Ticket) {
	if t == nil {
		return
	}
	
	q.mu.Lock()
	defer q.mu.Unlock()
	
	heap.Push(q.heap, t)
}

// Pop removes and returns the highest priority ticket
// Returns nil if the queue is empty
func (q *Queue) Pop() *ticket.Ticket {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if q.heap.Len() == 0 {
		return nil
	}
	
	return heap.Pop(q.heap).(*ticket.Ticket)
}

// Peek returns the highest priority ticket without removing it
// Returns nil if the queue is empty
func (q *Queue) Peek() *ticket.Ticket {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	return q.heap.peek()
}

// Len returns the number of tickets in the queue
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	return q.heap.Len()
}

// IsEmpty returns true if the queue has no tickets
func (q *Queue) IsEmpty() bool {
	return q.Len() == 0
}

// List returns a copy of all tickets in the queue ordered by priority
func (q *Queue) List() []*ticket.Ticket {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	result := make([]*ticket.Ticket, len(*q.heap))
	copy(result, *q.heap)
	return result
}

// Remove removes a ticket with the given ID from the queue
// Returns true if the ticket was found and removed
func (q *Queue) Remove(ticketID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	// Find the ticket in the heap
	for i, t := range *q.heap {
		if t.ID == ticketID {
			// Remove the item at index i
			heap.Remove(q.heap, i)
			return true
		}
	}
	
	return false
}

// Clear removes all tickets from the queue
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	*q.heap = (*q.heap)[:0]
	heap.Init(q.heap)
}

// String returns a string representation of the queue
func (q *Queue) String() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	if q.heap.Len() == 0 {
		return "Queue: empty"
	}
	
	result := fmt.Sprintf("Queue (%d tickets):\n", q.heap.Len())
	for i, t := range *q.heap {
		result += fmt.Sprintf("  %d. [P%d] %s: %s\n", i+1, t.Priority, t.ID, t.Title)
	}
	
	return result
}