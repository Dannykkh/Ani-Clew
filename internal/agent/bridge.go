package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aniclew/aniclew/internal/types"
)

// Bridge provides remote control of the agent via HTTP.
// Allows external tools (IDE extensions, scripts) to send commands
// and receive results without the web UI.
type Bridge struct {
	mu       sync.RWMutex
	sessions map[string]*BridgeSession
	provider types.Provider
	model    string
	workDir  string
}

// BridgeSession is a single remote agent session.
type BridgeSession struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"` // idle, running, done
	Messages  []types.Message `json:"-"`
	LastInput string    `json:"lastInput"`
	LastOutput string   `json:"lastOutput"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewBridge creates a bridge controller.
func NewBridge(provider types.Provider, model, workDir string) *Bridge {
	return &Bridge{
		sessions: make(map[string]*BridgeSession),
		provider: provider,
		model:    model,
		workDir:  workDir,
	}
}

// CreateSession starts a new bridge session.
func (b *Bridge) CreateSession() *BridgeSession {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := fmt.Sprintf("bridge_%d", time.Now().UnixMilli())
	sess := &BridgeSession{
		ID:        id,
		Status:    "idle",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	b.sessions[id] = sess
	log.Printf("[Bridge] Session created: %s", id)
	return sess
}

// Send sends a message to a bridge session and returns the response.
func (b *Bridge) Send(sessionID string, input string) (string, error) {
	b.mu.Lock()
	sess, ok := b.sessions[sessionID]
	if !ok {
		b.mu.Unlock()
		return "", fmt.Errorf("session %s not found", sessionID)
	}
	sess.Status = "running"
	sess.LastInput = input
	sess.UpdatedAt = time.Now()

	// Build messages
	userContent, _ := json.Marshal(input)
	sess.Messages = append(sess.Messages, types.Message{
		Role: "user", Content: userContent,
	})
	messages := make([]types.Message, len(sess.Messages))
	copy(messages, sess.Messages)
	b.mu.Unlock()

	// Run agent
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	eventCh := make(chan Event, 100)
	go RunLoop(ctx, b.provider, b.model, messages, b.workDir, "auto", eventCh)

	// Collect response
	var result string
	for event := range eventCh {
		if text, ok := event.Data.(string); ok && event.Type == "text" {
			result += text
		}
	}

	b.mu.Lock()
	sess.Status = "idle"
	sess.LastOutput = result
	sess.UpdatedAt = time.Now()
	assistantContent, _ := json.Marshal(result)
	sess.Messages = append(sess.Messages, types.Message{
		Role: "assistant", Content: assistantContent,
	})
	b.mu.Unlock()

	return result, nil
}

// GetSession returns a session by ID.
func (b *Bridge) GetSession(id string) *BridgeSession {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.sessions[id]
}

// ListSessions returns all bridge sessions.
func (b *Bridge) ListSessions() []*BridgeSession {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]*BridgeSession, 0, len(b.sessions))
	for _, s := range b.sessions {
		result = append(result, s)
	}
	return result
}

// CloseSession removes a bridge session.
func (b *Bridge) CloseSession(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.sessions, id)
	log.Printf("[Bridge] Session closed: %s", id)
}
