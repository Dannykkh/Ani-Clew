package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SessionMemory stores large tool results on disk to keep conversation context small.
type SessionMemory struct {
	mu      sync.RWMutex
	dir     string
	offsets map[string]int64 // resultID → file offset for resume
	nextID  int
}

// NewSessionMemory creates a disk-backed session memory.
func NewSessionMemory(baseDir, sessionID string) *SessionMemory {
	dir := filepath.Join(baseDir, "session-output", sessionID)
	os.MkdirAll(dir, 0755)
	return &SessionMemory{
		dir:     dir,
		offsets: make(map[string]int64),
	}
}

// StoreResult saves a large tool result to disk and returns a reference.
// Returns (reference, shouldReplace) — if shouldReplace is true, the caller
// should replace the full result with the reference in the conversation.
func (sm *SessionMemory) StoreResult(toolName string, result string) (string, bool) {
	const maxInlineSize = 2000 // keep results under 2KB inline

	if len(result) <= maxInlineSize {
		return result, false // small enough to keep inline
	}

	sm.mu.Lock()
	sm.nextID++
	id := fmt.Sprintf("result_%d", sm.nextID)
	sm.mu.Unlock()

	// Write to disk
	path := filepath.Join(sm.dir, id+".txt")
	os.WriteFile(path, []byte(result), 0644)

	// Return truncated reference
	preview := result[:500]
	if len(result) > 500 {
		preview += "..."
	}
	ref := fmt.Sprintf("[Result stored: %s (%d bytes)]\n%s\n[Full output: %s]",
		id, len(result), preview, path)

	return ref, true
}

// LoadResult reads a stored result from disk.
func (sm *SessionMemory) LoadResult(id string) (string, error) {
	path := filepath.Join(sm.dir, id+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("result %s not found", id)
	}
	return string(data), nil
}

// ListResults returns all stored result IDs with sizes.
func (sm *SessionMemory) ListResults() []map[string]interface{} {
	entries, _ := os.ReadDir(sm.dir)
	var results []map[string]interface{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, _ := e.Info()
		results = append(results, map[string]interface{}{
			"id":   e.Name()[:len(e.Name())-4], // strip .txt
			"size": info.Size(),
		})
	}
	return results
}

// Cleanup removes all stored results for this session.
func (sm *SessionMemory) Cleanup() {
	os.RemoveAll(sm.dir)
}

// TotalSize returns the total bytes stored on disk.
func (sm *SessionMemory) TotalSize() int64 {
	var total int64
	entries, _ := os.ReadDir(sm.dir)
	for _, e := range entries {
		info, _ := e.Info()
		if info != nil {
			total += info.Size()
		}
	}
	return total
}

// SessionMemoryStats returns summary statistics.
type SessionMemoryStats struct {
	ResultCount int   `json:"resultCount"`
	TotalBytes  int64 `json:"totalBytes"`
	SessionDir  string `json:"sessionDir"`
}

func (sm *SessionMemory) Stats() SessionMemoryStats {
	entries, _ := os.ReadDir(sm.dir)
	return SessionMemoryStats{
		ResultCount: len(entries),
		TotalBytes:  sm.TotalSize(),
		SessionDir:  sm.dir,
	}
}

// Integration: wrap ExecuteTool to auto-store large results
func ExecuteToolWithMemory(name string, input json.RawMessage, workDir string, mem *SessionMemory) (string, bool) {
	result, isError := ExecuteTool(name, input, workDir)

	if mem != nil && !isError {
		ref, replaced := mem.StoreResult(name, result)
		if replaced {
			return ref, false
		}
	}

	return result, isError
}
