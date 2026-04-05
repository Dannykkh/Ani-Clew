package kairos

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// Notification represents an event to broadcast.
type Notification struct {
	Type      string    `json:"type"`      // "git-change", "task-done", "daemon-start", etc.
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Project   string    `json:"project,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Notifier manages webhook and SSE subscribers.
type Notifier struct {
	mu          sync.RWMutex
	webhookURL  string
	subscribers map[chan Notification]struct{}
	history     []Notification
}

func NewNotifier() *Notifier {
	return &Notifier{
		subscribers: make(map[chan Notification]struct{}),
		history:     make([]Notification, 0, 100),
	}
}

// SetWebhook configures the webhook URL.
func (n *Notifier) SetWebhook(url string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.webhookURL = url
}

// GetWebhook returns the current webhook URL.
func (n *Notifier) GetWebhook() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.webhookURL
}

// Subscribe returns a channel for SSE notifications.
func (n *Notifier) Subscribe() chan Notification {
	n.mu.Lock()
	defer n.mu.Unlock()
	ch := make(chan Notification, 16)
	n.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber.
func (n *Notifier) Unsubscribe(ch chan Notification) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.subscribers, ch)
	close(ch)
}

// Send broadcasts a notification to all subscribers and webhook.
func (n *Notifier) Send(notif Notification) {
	notif.Timestamp = time.Now()

	n.mu.Lock()
	n.history = append(n.history, notif)
	if len(n.history) > 100 {
		n.history = n.history[len(n.history)-50:]
	}
	subs := make([]chan Notification, 0, len(n.subscribers))
	for ch := range n.subscribers {
		subs = append(subs, ch)
	}
	webhookURL := n.webhookURL
	n.mu.Unlock()

	// Broadcast to SSE subscribers
	for _, ch := range subs {
		select {
		case ch <- notif:
		default: // skip if subscriber is slow
		}
	}

	// Send to webhook
	if webhookURL != "" {
		go n.sendWebhook(webhookURL, notif)
	}
}

// Recent returns recent notifications.
func (n *Notifier) Recent(limit int) []Notification {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if limit <= 0 || limit > len(n.history) {
		limit = len(n.history)
	}
	start := len(n.history) - limit
	result := make([]Notification, limit)
	copy(result, n.history[start:])
	return result
}

func (n *Notifier) sendWebhook(url string, notif Notification) {
	body, _ := json.Marshal(notif)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[KAIROS] Webhook error: %v", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("[KAIROS] Webhook returned %d", resp.StatusCode)
	}
}
