package router

import (
	"encoding/json"
	"strings"

	"github.com/aniclew/aniclew/internal/types"
)

// Classify analyzes an incoming request and returns the best routing role.
func Classify(req *types.MessagesRequest, cfg *RouterConfig) (RoleID, string) {
	toolNames := make(map[string]bool)
	for _, t := range req.Tools {
		toolNames[t.Name] = true
	}

	// ── 1. Agent spawn ──
	if toolNames["Agent"] || toolNames["Task"] {
		return RoleAgentSpawn, "Agent/Task tool present"
	}

	// ── 2. User intent from text ──
	userText := strings.ToLower(extractLastUserText(req))

	if matchesAny(userText, "commit", "커밋") {
		return RoleCommit, "Commit request"
	}
	if matchesAny(userText, "explain", "what is", "what does", "how does", "why does", "설명", "뭐야", "어떻게", "왜", "알려줘", "?") {
		return RoleExplain, "Explanation intent"
	}
	if matchesAny(userText, "debug", "fix", "error", "bug", "broken", "crash", "fail", "not working", "디버그", "에러", "버그", "수정", "안 돼", "오류") {
		return RoleDebug, "Debug intent"
	}
	if matchesAny(userText, "review", "check", "audit", "리뷰", "검토", "확인") {
		return RoleReview, "Review intent"
	}
	if matchesAny(userText, "test", "spec", "coverage", "테스트", "검증") {
		return RoleTest, "Test intent"
	}
	if matchesAny(userText, "refactor", "restructure", "clean up", "simplify", "architecture", "리팩토링", "정리", "구조", "재설계") {
		return RoleRefactor, "Refactor intent"
	}

	// ── 3. Tool-based ──
	hasBash := toolNames["Bash"]
	hasEdit := toolNames["Edit"] || toolNames["FileEditTool"]
	hasWrite := toolNames["Write"] || toolNames["FileWriteTool"]
	hasRead := toolNames["Read"]
	hasGlob := toolNames["Glob"]
	hasGrep := toolNames["Grep"]
	isReadOnly := !hasEdit && !hasWrite && !hasBash

	if hasEdit && hasWrite && hasGlob && matchesAny(userText, "all files", "across", "every", "global", "project-wide", "전체", "모든 파일") {
		return RoleMultiFileEdit, "Multi-file edit predicted"
	}
	if hasEdit || hasWrite {
		return RoleFileEdit, "File edit/write tools"
	}
	if isReadOnly && (hasRead || hasGlob || hasGrep) {
		return RoleFileRead, "Read-only tools"
	}
	if hasBash && !hasEdit && !hasWrite {
		return RoleBashOnly, "Bash-only tools"
	}

	// ── 4. Context length ──
	tokens := estimateTokens(req)
	if tokens > cfg.ContextThresholds.Long {
		return RoleLongCtx, "Long context"
	}
	if tokens < cfg.ContextThresholds.Short {
		return RoleShortCtx, "Short context"
	}

	return RoleDefault, "Default"
}

func extractLastUserText(req *types.MessagesRequest) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			// Try string
			var s string
			if json.Unmarshal(req.Messages[i].Content, &s) == nil {
				return s
			}
			// Try blocks
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(req.Messages[i].Content, &blocks) == nil {
				var parts []string
				for _, b := range blocks {
					if b.Type == "text" {
						parts = append(parts, b.Text)
					}
				}
				return strings.Join(parts, " ")
			}
		}
	}
	return ""
}

func estimateTokens(req *types.MessagesRequest) int {
	chars := len(req.System)
	for _, m := range req.Messages {
		chars += len(m.Content)
	}
	return chars / 4
}

func matchesAny(text string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}
