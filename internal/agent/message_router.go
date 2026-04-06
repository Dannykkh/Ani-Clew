package agent

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MessageRouter handles real-time message delivery between agents.
// Combines mailbox persistence with in-memory channel delivery.
type MessageRouter struct {
	mu          sync.RWMutex
	subscribers map[string]chan TeamMessage // agentID → live channel
	mailbox     *Mailbox                    // persistent fallback
}

// NewMessageRouter creates a router with persistent mailbox backing.
func NewMessageRouter(mailbox *Mailbox) *MessageRouter {
	return &MessageRouter{
		subscribers: make(map[string]chan TeamMessage),
		mailbox:     mailbox,
	}
}

// Subscribe registers an agent for live message delivery.
func (r *MessageRouter) Subscribe(agentID string) <-chan TeamMessage {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan TeamMessage, 16)
	r.subscribers[agentID] = ch

	// Ensure mailbox inbox exists
	if r.mailbox != nil {
		r.mailbox.EnsureInbox(agentID)
	}

	log.Printf("[MessageRouter] %s subscribed", agentID)
	return ch
}

// Unsubscribe removes an agent from live delivery.
func (r *MessageRouter) Unsubscribe(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ch, ok := r.subscribers[agentID]; ok {
		close(ch)
		delete(r.subscribers, agentID)
	}
	log.Printf("[MessageRouter] %s unsubscribed", agentID)
}

// Send delivers a message to the target agent.
// Uses live channel if available, falls back to mailbox.
func (r *MessageRouter) Send(msg TeamMessage) {
	msg.Timestamp = time.Now()
	if msg.ID == "" {
		msg.ID = generateMsgID()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if msg.To == "*" {
		// Broadcast to all subscribers except sender
		for id, ch := range r.subscribers {
			if id != msg.From {
				r.deliverToChannel(ch, msg)
			}
		}
		// Also persist to mailbox for offline agents
		if r.mailbox != nil {
			r.mailbox.Send(msg)
		}
		return
	}

	// Direct message
	if ch, ok := r.subscribers[msg.To]; ok {
		// Live delivery
		r.deliverToChannel(ch, msg)
	} else if r.mailbox != nil {
		// Offline: save to mailbox
		r.mailbox.Send(msg)
		log.Printf("[MessageRouter] %s offline, saved to mailbox", msg.To)
	}
}

// deliverToChannel sends with non-blocking fallback.
func (r *MessageRouter) deliverToChannel(ch chan TeamMessage, msg TeamMessage) {
	select {
	case ch <- msg:
	default:
		log.Printf("[MessageRouter] Channel full for %s, dropping message", msg.To)
	}
}

// PollMailbox checks for unread mailbox messages and delivers them.
// Call periodically to catch messages sent while agent was offline.
func (r *MessageRouter) PollMailbox(agentID string) []TeamMessage {
	if r.mailbox == nil {
		return nil
	}
	return r.mailbox.Receive(agentID)
}

// IsOnline checks if an agent is currently subscribed.
func (r *MessageRouter) IsOnline(agentID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.subscribers[agentID]
	return ok
}

// OnlineAgents returns list of currently subscribed agent IDs.
func (r *MessageRouter) OnlineAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.subscribers))
	for id := range r.subscribers {
		ids = append(ids, id)
	}
	return ids
}

func generateMsgID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
