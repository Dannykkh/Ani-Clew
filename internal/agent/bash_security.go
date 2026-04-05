package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// SecurityCheckResult holds the result of a security validation.
type SecurityCheckResult struct {
	Blocked bool   // true if command should be blocked
	Reason  string // human-readable reason
	Pattern string // which pattern matched
}

// ValidateBashSecurity runs all security validators against a command.
// Returns the first blocking result, or nil if all pass.
func ValidateBashSecurity(command string, workDir string) *SecurityCheckResult {
	validators := []struct {
		name string
		fn   func(string, string) *SecurityCheckResult
	}{
		{"command-substitution", checkCommandSubstitution},
		{"dangerous-variables", checkDangerousVariables},
		{"ifs-injection", checkIFSInjection},
		{"shell-metacharacters", checkShellMetacharacters},
		{"brace-expansion", checkBraceExpansion},
		{"unicode-whitespace", checkUnicodeWhitespace},
		{"heredoc-execution", checkHeredocExecution},
		{"dangerous-paths", checkDangerousPaths},
		{"dangerous-commands", checkDangerousCommands},
		{"process-substitution", checkProcessSubstitution},
		{"sed-execution", checkSedExecution},
		{"jq-system", checkJqSystem},
		{"env-var-hijack", checkEnvVarHijack},
		{"backslash-escape", checkBackslashEscape},
		{"redirect-to-system", checkRedirectToSystem},
	}

	for _, v := range validators {
		if result := v.fn(command, workDir); result != nil {
			result.Pattern = v.name
			return result
		}
	}
	return nil
}

// ── Validators ──

// checkCommandSubstitution detects $(...), `...`, and process substitution
func checkCommandSubstitution(cmd, _ string) *SecurityCheckResult {
	// Skip if inside single quotes (rough check)
	unquoted := stripSingleQuoted(cmd)

	// $(...) command substitution
	if regexp.MustCompile(`\$\(`).MatchString(unquoted) {
		return &SecurityCheckResult{Blocked: true, Reason: "Command substitution $() detected"}
	}
	// Backtick substitution
	if strings.Count(unquoted, "`")%2 != 0 || (strings.Count(unquoted, "`") >= 2 && !isInsideQuotes(cmd, strings.Index(cmd, "`"))) {
		return &SecurityCheckResult{Blocked: true, Reason: "Backtick command substitution detected"}
	}
	return nil
}

// checkDangerousVariables detects dangerous environment variable usage
func checkDangerousVariables(cmd, _ string) *SecurityCheckResult {
	dangerous := []string{
		"BASH_ENV", "ENV", "BASH_FUNC",
		"LD_PRELOAD", "LD_LIBRARY_PATH",
		"DYLD_INSERT_LIBRARIES", "DYLD_LIBRARY_PATH",
		"PROMPT_COMMAND",
		"SHELL", "BASH",
	}

	for _, v := range dangerous {
		// Check both $VAR and VAR= assignment at start
		if strings.Contains(cmd, "$"+v) || strings.Contains(cmd, "${"+v+"}") {
			return &SecurityCheckResult{Blocked: true, Reason: fmt.Sprintf("Dangerous variable $%s referenced", v)}
		}
		if regexp.MustCompile(`(?:^|\s)` + v + `=`).MatchString(cmd) {
			return &SecurityCheckResult{Blocked: true, Reason: fmt.Sprintf("Dangerous variable %s= being set", v)}
		}
	}
	return nil
}

// checkIFSInjection detects IFS manipulation attacks
func checkIFSInjection(cmd, _ string) *SecurityCheckResult {
	if regexp.MustCompile(`(?i)\bIFS\s*=`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "IFS variable manipulation detected"}
	}
	return nil
}

// checkShellMetacharacters detects unquoted dangerous metacharacters in compound commands
func checkShellMetacharacters(cmd, _ string) *SecurityCheckResult {
	// These are fine in normal usage — only block suspicious combinations
	unquoted := stripSingleQuoted(cmd)

	// Detect eval, source, exec being injected
	if regexp.MustCompile(`(?:^|\s|;|&&|\|\|)\s*eval\s`).MatchString(unquoted) {
		return &SecurityCheckResult{Blocked: true, Reason: "eval command detected"}
	}
	if regexp.MustCompile(`(?:^|\s|;|&&|\|\|)\s*exec\s+\d*[<>]`).MatchString(unquoted) {
		return &SecurityCheckResult{Blocked: true, Reason: "exec with file descriptor redirect detected"}
	}
	return nil
}

// checkBraceExpansion detects dangerous brace expansion
func checkBraceExpansion(cmd, _ string) *SecurityCheckResult {
	// {/etc/passwd,/etc/shadow} — can create unexpected file arguments
	if regexp.MustCompile(`\{[/~].*,.*\}`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "Brace expansion with paths detected"}
	}
	return nil
}

// checkUnicodeWhitespace detects non-ASCII whitespace that can bypass parsing
func checkUnicodeWhitespace(cmd, _ string) *SecurityCheckResult {
	for _, r := range cmd {
		if r > 127 && unicode.IsSpace(r) {
			return &SecurityCheckResult{Blocked: true, Reason: fmt.Sprintf("Non-ASCII whitespace U+%04X detected", r)}
		}
	}
	return nil
}

// checkHeredocExecution detects heredoc with command execution
func checkHeredocExecution(cmd, _ string) *SecurityCheckResult {
	if regexp.MustCompile(`<<\s*['"]*\w+['"]*[\s\S]*\$\(`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "Command substitution inside heredoc detected"}
	}
	return nil
}

// checkDangerousPaths detects operations on critical system paths
func checkDangerousPaths(cmd, workDir string) *SecurityCheckResult {
	dangerousPaths := []string{
		"/", "/etc", "/sys", "/proc", "/var", "/boot",
		"/usr/bin", "/usr/sbin", "/usr/lib",
		"/dev", "/root",
	}

	// Check rm/mv/cp targeting dangerous paths
	if regexp.MustCompile(`(?:^|\s)(?:rm|mv|cp|chmod|chown)\s`).MatchString(cmd) {
		for _, dp := range dangerousPaths {
			if regexp.MustCompile(`\s` + regexp.QuoteMeta(dp) + `(?:\s|$|/\s)`).MatchString(cmd) {
				return &SecurityCheckResult{Blocked: true, Reason: fmt.Sprintf("Dangerous operation on system path %s", dp)}
			}
		}
	}

	// Check rm -rf specifically
	if regexp.MustCompile(`rm\s+(-[rRf]+\s+)*(-[rRf]*\s+)*/\s*$`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "rm -rf / detected"}
	}

	// Check .git directory destruction
	if regexp.MustCompile(`(?:rm|mv).*\.git(?:\s|$|/)`).MatchString(cmd) {
		absGit := filepath.Join(workDir, ".git")
		if _, err := os.Stat(absGit); err == nil {
			return &SecurityCheckResult{Blocked: true, Reason: ".git directory operation detected"}
		}
	}

	return nil
}

// checkDangerousCommands detects inherently dangerous commands
func checkDangerousCommands(cmd, _ string) *SecurityCheckResult {
	patterns := []struct {
		re   string
		desc string
	}{
		{`(?:^|\s)mkfs\b`, "mkfs (format disk) detected"},
		{`(?:^|\s)dd\s+.*of=/dev/`, "dd writing to device detected"},
		{`:\(\)\s*\{`, "Fork bomb detected"},
		{`>\s*/dev/sd[a-z]`, "Writing to block device detected"},
		{`(?:^|\s)curl\s.*\|\s*(?:bash|sh|zsh)`, "Pipe from curl to shell detected"},
		{`(?:^|\s)wget\s.*\|\s*(?:bash|sh|zsh)`, "Pipe from wget to shell detected"},
		{`(?:^|\s)python\s+-c\s+.*(?:import\s+os|subprocess|shutil)`, "Python code execution with OS access"},
		{`(?:^|\s)node\s+-e\s+.*(?:child_process|exec|spawn)`, "Node code execution with child_process"},
	}

	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p.re, cmd); matched {
			return &SecurityCheckResult{Blocked: true, Reason: p.desc}
		}
	}
	return nil
}

// checkProcessSubstitution detects <() and >() Bash process substitution
func checkProcessSubstitution(cmd, _ string) *SecurityCheckResult {
	unquoted := stripSingleQuoted(cmd)
	if regexp.MustCompile(`[<>]\(`).MatchString(unquoted) {
		return &SecurityCheckResult{Blocked: true, Reason: "Process substitution <() or >() detected"}
	}
	return nil
}

// checkSedExecution detects dangerous sed commands
func checkSedExecution(cmd, _ string) *SecurityCheckResult {
	if !regexp.MustCompile(`(?:^|\s)sed\s`).MatchString(cmd) {
		return nil
	}
	// sed 'e' command executes shell — check for \de or standalone e
	if regexp.MustCompile(`sed\s.*'\d*e[\s']`).MatchString(cmd) || regexp.MustCompile(`sed\s.*-e\s*'e\b`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "sed 'e' command (shell execution) detected"}
	}
	// sed 'w' command writes to file
	if regexp.MustCompile(`sed\s.*'[^']*w\s+/`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "sed 'w' command (write to file) detected"}
	}
	return nil
}

// checkJqSystem detects jq system() calls
func checkJqSystem(cmd, _ string) *SecurityCheckResult {
	if strings.Contains(cmd, "jq") && strings.Contains(cmd, "system(") {
		return &SecurityCheckResult{Blocked: true, Reason: "jq system() call detected"}
	}
	return nil
}

// checkEnvVarHijack detects LD_PRELOAD-style binary hijacking
func checkEnvVarHijack(cmd, _ string) *SecurityCheckResult {
	if regexp.MustCompile(`LD_PRELOAD=\S+\s+\w`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "LD_PRELOAD binary hijacking detected"}
	}
	if regexp.MustCompile(`PATH=.*:\.\s`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "PATH injection with current directory detected"}
	}
	return nil
}

// checkBackslashEscape detects escaped operators hiding dangerous commands
func checkBackslashEscape(cmd, _ string) *SecurityCheckResult {
	// Detect \; \| used to hide operators from naive parsers
	// Only flag if combined with dangerous patterns
	if strings.Contains(cmd, "\\;") && regexp.MustCompile(`(?:rm|mv|chmod)\s`).MatchString(cmd) {
		return &SecurityCheckResult{Blocked: true, Reason: "Escaped semicolon with dangerous command detected"}
	}
	return nil
}

// checkRedirectToSystem detects output redirection to system files
func checkRedirectToSystem(cmd, _ string) *SecurityCheckResult {
	systemPaths := []string{"/etc/", "/usr/", "/var/", "/sys/", "/proc/", "/dev/"}
	if !strings.Contains(cmd, ">") {
		return nil
	}
	for _, sp := range systemPaths {
		pattern := `>\s*` + regexp.QuoteMeta(sp)
		if matched, _ := regexp.MatchString(pattern, cmd); matched {
			return &SecurityCheckResult{Blocked: true, Reason: fmt.Sprintf("Output redirect to system path %s detected", sp)}
		}
	}
	return nil
}

// ── Helpers ──

// stripSingleQuoted removes single-quoted sections for analysis
func stripSingleQuoted(s string) string {
	var result strings.Builder
	inSingle := false
	for _, r := range s {
		if r == '\'' {
			inSingle = !inSingle
			continue
		}
		if !inSingle {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// isInsideQuotes checks if position is inside quotes
func isInsideQuotes(s string, pos int) bool {
	singles := 0
	doubles := 0
	for i := 0; i < pos && i < len(s); i++ {
		if s[i] == '\'' && doubles%2 == 0 {
			singles++
		}
		if s[i] == '"' && singles%2 == 0 {
			doubles++
		}
	}
	return singles%2 != 0 || doubles%2 != 0
}

// ParseBaseCommand extracts the base command, stripping env vars and wrappers.
func ParseBaseCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)

	// Strip leading env var assignments: VAR=value cmd
	for regexp.MustCompile(`^\w+=\S+\s+`).MatchString(cmd) {
		cmd = regexp.MustCompile(`^\w+=\S+\s+`).ReplaceAllString(cmd, "")
	}

	// Strip common wrappers
	wrappers := []string{"timeout", "nice", "nohup", "time", "strace", "ltrace", "env"}
	for _, w := range wrappers {
		prefix := w + " "
		if strings.HasPrefix(cmd, prefix) {
			cmd = strings.TrimPrefix(cmd, prefix)
			cmd = strings.TrimSpace(cmd)
			// Skip wrapper's own arguments
			if w == "timeout" {
				// timeout VALUE cmd...
				parts := strings.SplitN(cmd, " ", 2)
				if len(parts) > 1 {
					cmd = parts[1]
				}
			} else if w == "nice" {
				// nice [-n VALUE] cmd... — skip -n and its value
				if strings.HasPrefix(cmd, "-n ") || strings.HasPrefix(cmd, "-n\t") {
					cmd = strings.TrimPrefix(cmd, "-n ")
					cmd = strings.TrimSpace(cmd)
					parts := strings.SplitN(cmd, " ", 2)
					if len(parts) > 1 {
						cmd = parts[1]
					}
				}
			}
		}
	}

	// Extract just the command name
	parts := strings.Fields(cmd)
	if len(parts) > 0 {
		return parts[0]
	}
	return cmd
}

// ExitCodeSemantics interprets exit codes for known commands.
func ExitCodeSemantics(baseCmd string, exitCode int) (string, bool) {
	if exitCode == 0 {
		return "", false
	}

	switch baseCmd {
	case "grep", "rg", "ag", "ack":
		if exitCode == 1 {
			return "No matches found (not an error)", false
		}
	case "diff":
		if exitCode == 1 {
			return "Files differ (not an error)", false
		}
	case "test", "[":
		if exitCode == 1 {
			return "Test condition false (not an error)", false
		}
	case "find":
		if exitCode == 1 {
			return "Some files/directories inaccessible (partial results)", false
		}
	}

	return "", true // actual error
}
