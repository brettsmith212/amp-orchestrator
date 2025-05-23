package queue

import (
	"testing"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
)

func TestPushPopPriorities(t *testing.T) {
	q := New()
	
	// Create tickets with different priorities
	// Priority 1 = highest, 5 = lowest
	ticket1 := &ticket.Ticket{
		ID:          "low-priority",
		Title:       "Low priority task",
		Description: "This can wait",
		Priority:    5,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	ticket2 := &ticket.Ticket{
		ID:          "high-priority",
		Title:       "High priority task",
		Description: "This is urgent",
		Priority:    1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	ticket3 := &ticket.Ticket{
		ID:          "medium-priority",
		Title:       "Medium priority task",
		Description: "This is normal",
		Priority:    3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Push in random order
	q.Push(ticket1) // Priority 5
	q.Push(ticket2) // Priority 1
	q.Push(ticket3) // Priority 3
	
	// Verify queue length
	if q.Len() != 3 {
		t.Errorf("Expected queue length 3, got %d", q.Len())
	}
	
	// Pop should yield highest priority first (1, then 3, then 5)
	first := q.Pop()
	if first == nil || first.Priority != 1 {
		t.Errorf("Expected first pop to be priority 1, got %v", first)
	}
	if first.ID != "high-priority" {
		t.Errorf("Expected first pop to be high-priority ticket, got %s", first.ID)
	}
	
	second := q.Pop()
	if second == nil || second.Priority != 3 {
		t.Errorf("Expected second pop to be priority 3, got %v", second)
	}
	if second.ID != "medium-priority" {
		t.Errorf("Expected second pop to be medium-priority ticket, got %s", second.ID)
	}
	
	third := q.Pop()
	if third == nil || third.Priority != 5 {
		t.Errorf("Expected third pop to be priority 5, got %v", third)
	}
	if third.ID != "low-priority" {
		t.Errorf("Expected third pop to be low-priority ticket, got %s", third.ID)
	}
	
	// Queue should be empty now
	if !q.IsEmpty() {
		t.Error("Expected queue to be empty after popping all items")
	}
	
	// Pop from empty queue should return nil
	empty := q.Pop()
	if empty != nil {
		t.Error("Expected pop from empty queue to return nil")
	}
}

func TestFIFOWithinSamePriority(t *testing.T) {
	q := New()
	
	baseTime := time.Now()
	
	// Create tickets with same priority but different creation times
	ticket1 := &ticket.Ticket{
		ID:          "first",
		Title:       "First ticket",
		Description: "Created first",
		Priority:    2,
		CreatedAt:   baseTime,
		UpdatedAt:   baseTime,
	}
	
	ticket2 := &ticket.Ticket{
		ID:          "second",
		Title:       "Second ticket",
		Description: "Created second",
		Priority:    2,
		CreatedAt:   baseTime.Add(1 * time.Second),
		UpdatedAt:   baseTime.Add(1 * time.Second),
	}
	
	// Push in order
	q.Push(ticket1)
	q.Push(ticket2)
	
	// Should pop in FIFO order for same priority
	first := q.Pop()
	if first.ID != "first" {
		t.Errorf("Expected first ticket to be popped first, got %s", first.ID)
	}
	
	second := q.Pop()
	if second.ID != "second" {
		t.Errorf("Expected second ticket to be popped second, got %s", second.ID)
	}
}

func TestPeek(t *testing.T) {
	q := New()
	
	// Peek empty queue
	if q.Peek() != nil {
		t.Error("Expected peek on empty queue to return nil")
	}
	
	ticket1 := &ticket.Ticket{
		ID:          "test",
		Title:       "Test ticket",
		Description: "Test",
		Priority:    1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	q.Push(ticket1)
	
	// Peek should return the ticket without removing it
	peeked := q.Peek()
	if peeked == nil || peeked.ID != "test" {
		t.Error("Peek should return the ticket without removing it")
	}
	
	// Queue length should still be 1
	if q.Len() != 1 {
		t.Error("Peek should not remove the ticket from queue")
	}
	
	// Pop should still return the same ticket
	popped := q.Pop()
	if popped.ID != "test" {
		t.Error("Pop after peek should return the same ticket")
	}
}

func TestRemove(t *testing.T) {
	q := New()
	
	ticket1 := &ticket.Ticket{
		ID:          "remove-me",
		Title:       "To be removed",
		Description: "This will be removed",
		Priority:    2,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	ticket2 := &ticket.Ticket{
		ID:          "keep-me",
		Title:       "To be kept",
		Description: "This will stay",
		Priority:    1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	q.Push(ticket1)
	q.Push(ticket2)
	
	// Remove existing ticket
	if !q.Remove("remove-me") {
		t.Error("Expected Remove to return true for existing ticket")
	}
	
	// Queue should have 1 item left
	if q.Len() != 1 {
		t.Errorf("Expected queue length 1 after removal, got %d", q.Len())
	}
	
	// Remaining ticket should be the correct one
	remaining := q.Pop()
	if remaining.ID != "keep-me" {
		t.Errorf("Expected remaining ticket to be 'keep-me', got %s", remaining.ID)
	}
	
	// Remove non-existent ticket
	if q.Remove("not-exists") {
		t.Error("Expected Remove to return false for non-existent ticket")
	}
}

func TestClear(t *testing.T) {
	q := New()
	
	// Add some tickets
	for i := 0; i < 5; i++ {
		ticket := &ticket.Ticket{
			ID:          "test-" + string(rune('0'+i)),
			Title:       "Test ticket",
			Description: "Test",
			Priority:    i + 1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		q.Push(ticket)
	}
	
	if q.Len() != 5 {
		t.Errorf("Expected 5 tickets in queue, got %d", q.Len())
	}
	
	// Clear the queue
	q.Clear()
	
	if !q.IsEmpty() {
		t.Error("Expected queue to be empty after clear")
	}
	
	if q.Len() != 0 {
		t.Errorf("Expected queue length 0 after clear, got %d", q.Len())
	}
}

func TestList(t *testing.T) {
	q := New()
	
	// Empty queue should return empty slice
	list := q.List()
	if len(list) != 0 {
		t.Error("Expected empty list for empty queue")
	}
	
	// Add tickets
	for i := 1; i <= 3; i++ {
		ticket := &ticket.Ticket{
			ID:          "test-" + string(rune('0'+i)),
			Title:       "Test ticket",
			Description: "Test",
			Priority:    i,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		q.Push(ticket)
	}
	
	list = q.List()
	if len(list) != 3 {
		t.Errorf("Expected list length 3, got %d", len(list))
	}
	
	// Modifying the returned list should not affect the queue
	list[0] = nil
	if q.Len() != 3 {
		t.Error("Modifying returned list should not affect queue")
	}
}

func TestConcurrentAccess(t *testing.T) {
	q := New()
	
	// This test verifies that the queue is thread-safe
	// Run multiple goroutines that push and pop
	done := make(chan bool)
	
	// Producer goroutine
	go func() {
		for i := 0; i < 10; i++ {
			ticket := &ticket.Ticket{
				ID:          "concurrent-" + string(rune('0'+i)),
				Title:       "Concurrent ticket",
				Description: "Test concurrency",
				Priority:    (i % 3) + 1,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			q.Push(ticket)
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()
	
	// Consumer goroutine
	go func() {
		count := 0
		for count < 10 {
			if ticket := q.Pop(); ticket != nil {
				count++
			}
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()
	
	// Wait for both goroutines to complete
	<-done
	<-done
	
	// Queue should be empty
	if !q.IsEmpty() {
		t.Error("Expected queue to be empty after concurrent operations")
	}
}