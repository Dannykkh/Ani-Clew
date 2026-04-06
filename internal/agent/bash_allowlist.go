package agent

import (
	"fmt"
	"regexp"
	"strings"
)

// FlagType defines how a flag consumes arguments.
type FlagType int

const (
	FlagNone   FlagType = iota // -v (no argument)
	FlagString                  // -o file (takes string argument)
	FlagNumber                  // -n 10 (takes numeric argument)
	FlagPath                    // -f path (takes file path argument)
	FlagChar                    // -d ',' (takes single char)
)

// CommandConfig defines allowed flags and behavior for a read-only command.
type CommandConfig struct {
	SafeFlags       map[string]FlagType // allowed flags with arg types
	DangerousFlags  []string            // flags that make it NOT read-only
	RespectsDoubleDash bool             // does -- end option parsing?
	CustomValidator func(args []string) bool // extra validation
}

// ReadOnlyAllowlist returns configs for commands considered read-only.
// Comprehensive read-only command allowlist.
func ReadOnlyAllowlist() map[string]CommandConfig {
	return map[string]CommandConfig{
		// ── File listing / info ──
		"ls": {
			SafeFlags: map[string]FlagType{
				"-l": FlagNone, "-a": FlagNone, "-h": FlagNone, "-R": FlagNone,
				"-t": FlagNone, "-S": FlagNone, "-r": FlagNone, "-1": FlagNone,
				"-d": FlagNone, "-i": FlagNone, "-F": FlagNone, "-p": FlagNone,
				"--color": FlagString, "--sort": FlagString, "--time": FlagString,
			},
			RespectsDoubleDash: true,
		},
		"stat": {
			SafeFlags: map[string]FlagType{"-f": FlagString, "-c": FlagString, "-L": FlagNone},
		},
		"file": {
			SafeFlags: map[string]FlagType{"-b": FlagNone, "-i": FlagNone, "-L": FlagNone, "-z": FlagNone},
		},
		"du": {
			SafeFlags: map[string]FlagType{
				"-h": FlagNone, "-s": FlagNone, "-a": FlagNone, "-d": FlagNumber,
				"--max-depth": FlagNumber, "-c": FlagNone, "--summarize": FlagNone,
			},
		},
		"df": {
			SafeFlags: map[string]FlagType{"-h": FlagNone, "-T": FlagNone, "-i": FlagNone},
		},
		"wc": {
			SafeFlags: map[string]FlagType{"-l": FlagNone, "-w": FlagNone, "-c": FlagNone, "-m": FlagNone},
		},

		// ── Text viewing ──
		"cat": {
			SafeFlags: map[string]FlagType{"-n": FlagNone, "-b": FlagNone, "-v": FlagNone, "-E": FlagNone, "-T": FlagNone, "-A": FlagNone},
		},
		"head": {
			SafeFlags: map[string]FlagType{"-n": FlagNumber, "-c": FlagNumber, "-q": FlagNone},
		},
		"tail": {
			SafeFlags: map[string]FlagType{"-n": FlagNumber, "-c": FlagNumber, "-f": FlagNone, "-q": FlagNone},
		},
		"less": {SafeFlags: map[string]FlagType{"-N": FlagNone, "-R": FlagNone, "-S": FlagNone}},
		"more": {SafeFlags: map[string]FlagType{}},
		"bat":  {SafeFlags: map[string]FlagType{"--style": FlagString, "--theme": FlagString, "-l": FlagString, "-n": FlagNone, "--paging": FlagString}},

		// ── Search ──
		"grep": {
			SafeFlags: map[string]FlagType{
				"-r": FlagNone, "-R": FlagNone, "-i": FlagNone, "-l": FlagNone,
				"-n": FlagNone, "-c": FlagNone, "-v": FlagNone, "-w": FlagNone,
				"-E": FlagNone, "-P": FlagNone, "-F": FlagNone, "-q": FlagNone,
				"-H": FlagNone, "-h": FlagNone, "-o": FlagNone, "-m": FlagNumber,
				"-A": FlagNumber, "-B": FlagNumber, "-C": FlagNumber,
				"--include": FlagString, "--exclude": FlagString, "--exclude-dir": FlagString,
				"--color": FlagString, "--binary-files": FlagString,
			},
			RespectsDoubleDash: true,
		},
		"rg": {
			SafeFlags: map[string]FlagType{
				"-i": FlagNone, "-l": FlagNone, "-n": FlagNone, "-c": FlagNone,
				"-w": FlagNone, "-v": FlagNone, "-F": FlagNone, "-U": FlagNone,
				"-g": FlagString, "--glob": FlagString, "-t": FlagString, "--type": FlagString,
				"-A": FlagNumber, "-B": FlagNumber, "-C": FlagNumber,
				"-m": FlagNumber, "--max-count": FlagNumber,
				"--json": FlagNone, "--no-heading": FlagNone, "--hidden": FlagNone,
				"--no-ignore": FlagNone, "-e": FlagString, "--color": FlagString,
			},
		},
		"ag": {
			SafeFlags: map[string]FlagType{"-i": FlagNone, "-l": FlagNone, "-c": FlagNone, "--hidden": FlagNone},
		},
		"ack": {
			SafeFlags: map[string]FlagType{"-i": FlagNone, "-l": FlagNone, "-c": FlagNone},
		},

		// ── Find ──
		"find": {
			SafeFlags: map[string]FlagType{
				"-name": FlagString, "-iname": FlagString, "-path": FlagString,
				"-type": FlagChar, "-maxdepth": FlagNumber, "-mindepth": FlagNumber,
				"-newer": FlagPath, "-size": FlagString, "-mtime": FlagString,
				"-atime": FlagString, "-ctime": FlagString, "-perm": FlagString,
				"-empty": FlagNone, "-not": FlagNone, "-print": FlagNone, "-print0": FlagNone,
			},
			DangerousFlags: []string{"-exec", "-execdir", "-delete", "-ok"},
		},
		"fd": {
			SafeFlags: map[string]FlagType{
				"-t": FlagChar, "-e": FlagString, "-E": FlagString, "-d": FlagNumber,
				"--max-depth": FlagNumber, "-H": FlagNone, "--hidden": FlagNone,
				"-I": FlagNone, "--no-ignore": FlagNone, "-a": FlagNone, "--absolute-path": FlagNone,
				"-l": FlagNone, "-0": FlagNone, "--color": FlagString,
			},
			DangerousFlags: []string{"-x", "--exec", "-X", "--exec-batch"},
		},
		"locate": {SafeFlags: map[string]FlagType{"-i": FlagNone, "-c": FlagNone, "-l": FlagNumber}},
		"which":  {SafeFlags: map[string]FlagType{"-a": FlagNone}},
		"where":  {SafeFlags: map[string]FlagType{}},
		"type":   {SafeFlags: map[string]FlagType{}},
		"whereis": {SafeFlags: map[string]FlagType{}},

		// ── Git (read-only) ──
		"git": {
			SafeFlags: map[string]FlagType{},
			CustomValidator: func(args []string) bool {
				if len(args) == 0 {
					return false
				}
				safeGitCommands := map[string]bool{
					"status": true, "log": true, "diff": true, "show": true,
					"branch": true, "tag": true, "remote": true, "stash": true,
					"describe": true, "rev-parse": true, "rev-list": true,
					"ls-files": true, "ls-tree": true, "cat-file": true,
					"blame": true, "shortlog": true, "reflog": true,
					"config": true, "help": true, "version": true,
				}
				return safeGitCommands[args[0]]
			},
		},

		// ── Process info ──
		"ps":     {SafeFlags: map[string]FlagType{"-e": FlagNone, "-f": FlagNone, "-u": FlagString, "aux": FlagNone, "-A": FlagNone}},
		"top":    {SafeFlags: map[string]FlagType{"-b": FlagNone, "-n": FlagNumber}},
		"htop":   {SafeFlags: map[string]FlagType{}},
		"uptime": {SafeFlags: map[string]FlagType{}},
		"free":   {SafeFlags: map[string]FlagType{"-h": FlagNone, "-m": FlagNone, "-g": FlagNone}},
		"lsof":   {SafeFlags: map[string]FlagType{"-i": FlagString, "-p": FlagNumber, "-t": FlagNone}},

		// ── Network (read-only) ──
		"ping":       {SafeFlags: map[string]FlagType{"-c": FlagNumber, "-W": FlagNumber}},
		"curl":       {SafeFlags: map[string]FlagType{"-s": FlagNone, "-S": FlagNone, "-L": FlagNone, "-I": FlagNone, "-v": FlagNone, "-H": FlagString, "--head": FlagNone}},
		"wget":       {SafeFlags: map[string]FlagType{"-q": FlagNone, "--spider": FlagNone}},
		"netstat":    {SafeFlags: map[string]FlagType{"-t": FlagNone, "-l": FlagNone, "-n": FlagNone, "-p": FlagNone, "-a": FlagNone}},
		"ss":         {SafeFlags: map[string]FlagType{"-t": FlagNone, "-l": FlagNone, "-n": FlagNone, "-p": FlagNone}},
		"dig":        {SafeFlags: map[string]FlagType{"+short": FlagNone}},
		"nslookup":   {SafeFlags: map[string]FlagType{}},
		"host":       {SafeFlags: map[string]FlagType{}},

		// ── Language/build info ──
		"go":     {SafeFlags: map[string]FlagType{}, CustomValidator: func(args []string) bool { return len(args) > 0 && (args[0] == "version" || args[0] == "env" || args[0] == "list" || args[0] == "doc" || args[0] == "help" || args[0] == "vet" || args[0] == "mod") }},
		"node":   {SafeFlags: map[string]FlagType{"--version": FlagNone, "-v": FlagNone, "-e": FlagString, "--eval": FlagString}},
		"python": {SafeFlags: map[string]FlagType{"--version": FlagNone, "-V": FlagNone, "-c": FlagString}},
		"python3": {SafeFlags: map[string]FlagType{"--version": FlagNone, "-V": FlagNone, "-c": FlagString}},
		"ruby":   {SafeFlags: map[string]FlagType{"--version": FlagNone, "-v": FlagNone}},
		"rustc":  {SafeFlags: map[string]FlagType{"--version": FlagNone}},
		"cargo":  {SafeFlags: map[string]FlagType{}, CustomValidator: func(args []string) bool { return len(args) > 0 && (args[0] == "version" || args[0] == "doc" || args[0] == "check" || args[0] == "clippy" || args[0] == "fmt" || args[0] == "test") }},
		"java":   {SafeFlags: map[string]FlagType{"-version": FlagNone, "--version": FlagNone}},
		"dotnet": {SafeFlags: map[string]FlagType{}, CustomValidator: func(args []string) bool { return len(args) > 0 && (args[0] == "--version" || args[0] == "--info" || args[0] == "list" || args[0] == "help") }},

		// ── Container (read-only) ──
		"docker": {
			SafeFlags: map[string]FlagType{},
			CustomValidator: func(args []string) bool {
				if len(args) == 0 { return false }
				safe := map[string]bool{"ps": true, "images": true, "inspect": true, "logs": true, "info": true, "version": true, "stats": true, "top": true, "port": true, "network": true, "volume": true}
				return safe[args[0]]
			},
		},
		"kubectl": {
			SafeFlags: map[string]FlagType{},
			CustomValidator: func(args []string) bool {
				if len(args) == 0 { return false }
				safe := map[string]bool{"get": true, "describe": true, "logs": true, "top": true, "api-resources": true, "version": true, "config": true, "cluster-info": true}
				return safe[args[0]]
			},
		},

		// ── Misc read-only ──
		"echo":     {SafeFlags: map[string]FlagType{"-n": FlagNone, "-e": FlagNone}},
		"printf":   {SafeFlags: map[string]FlagType{}},
		"date":     {SafeFlags: map[string]FlagType{"-u": FlagNone, "+%s": FlagNone}},
		"cal":      {SafeFlags: map[string]FlagType{}},
		"whoami":   {SafeFlags: map[string]FlagType{}},
		"id":       {SafeFlags: map[string]FlagType{}},
		"hostname": {SafeFlags: map[string]FlagType{}},
		"uname":    {SafeFlags: map[string]FlagType{"-a": FlagNone, "-r": FlagNone, "-m": FlagNone, "-s": FlagNone}},
		"env":      {SafeFlags: map[string]FlagType{}},
		"printenv": {SafeFlags: map[string]FlagType{}},
		"pwd":      {SafeFlags: map[string]FlagType{}},
		"realpath": {SafeFlags: map[string]FlagType{}},
		"basename": {SafeFlags: map[string]FlagType{}},
		"dirname":  {SafeFlags: map[string]FlagType{}},
		"md5sum":   {SafeFlags: map[string]FlagType{"-c": FlagPath}},
		"sha256sum":{SafeFlags: map[string]FlagType{"-c": FlagPath}},
		"diff":     {SafeFlags: map[string]FlagType{"-u": FlagNone, "-r": FlagNone, "-q": FlagNone, "--color": FlagString, "-y": FlagNone}},
		"sort":     {SafeFlags: map[string]FlagType{"-n": FlagNone, "-r": FlagNone, "-u": FlagNone, "-k": FlagString, "-t": FlagChar}},
		"uniq":     {SafeFlags: map[string]FlagType{"-c": FlagNone, "-d": FlagNone, "-u": FlagNone}},
		"tr":       {SafeFlags: map[string]FlagType{"-d": FlagNone, "-s": FlagNone, "-c": FlagNone}},
		"cut":      {SafeFlags: map[string]FlagType{"-d": FlagChar, "-f": FlagString, "-c": FlagString}},
		"awk":      {SafeFlags: map[string]FlagType{"-F": FlagChar, "-v": FlagString}},
		"sed":      {SafeFlags: map[string]FlagType{"-n": FlagNone, "-E": FlagNone, "-e": FlagString}},
		"jq":       {SafeFlags: map[string]FlagType{"-r": FlagNone, "-c": FlagNone, "-e": FlagNone, "--raw-output": FlagNone, "--slurp": FlagNone, "-s": FlagNone, "--arg": FlagString}},
		"yq":       {SafeFlags: map[string]FlagType{"-r": FlagNone, "-c": FlagNone}},
		"tree":     {SafeFlags: map[string]FlagType{"-L": FlagNumber, "-d": FlagNone, "-a": FlagNone, "-I": FlagString, "--gitignore": FlagNone}},
		"xargs":    {SafeFlags: map[string]FlagType{"-I": FlagString, "-0": FlagNone, "-n": FlagNumber, "-P": FlagNumber}},

		// ── gh CLI (read-only) ──
		"gh": {
			SafeFlags: map[string]FlagType{},
			CustomValidator: func(args []string) bool {
				if len(args) == 0 { return false }
				safe := map[string]bool{
					"auth":   true, // auth status is read-only
					"api":    true,
					"browse": true,
					"repo":   true, // repo view
					"pr":     true, // pr list/view
					"issue":  true, // issue list/view
					"run":    true, // run list/view
					"release":true, // release list/view
				}
				// First arg is safe subcommand
				if safe[args[0]] {
					// But block create/delete/merge subcommands
					if len(args) > 1 {
						dangerous := map[string]bool{"create": true, "delete": true, "merge": true, "close": true, "edit": true, "comment": true}
						return !dangerous[args[1]]
					}
					return true
				}
				return false
			},
		},
	}
}

// IsReadOnlyCommand checks if a command is in the read-only allowlist
// and only uses safe flags.
func IsReadOnlyCommand(command string) bool {
	command = strings.TrimSpace(command)
	base := ParseBaseCommand(command)
	allowlist := ReadOnlyAllowlist()

	config, ok := allowlist[base]
	if !ok {
		return false
	}

	// Parse arguments
	parts := splitCommandArgs(command)
	if len(parts) == 0 {
		return false
	}

	// Find the actual command args (skip the command name and any it matches in base)
	cmdArgs := parts[1:]
	// If base command was wrapped, need to find real args
	baseIdx := 0
	for i, p := range parts {
		if p == base {
			baseIdx = i
			break
		}
	}
	cmdArgs = parts[baseIdx+1:]

	// Custom validator
	if config.CustomValidator != nil {
		return config.CustomValidator(cmdArgs)
	}

	// Check for dangerous flags
	for _, arg := range cmdArgs {
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		for _, df := range config.DangerousFlags {
			if arg == df || strings.HasPrefix(arg, df+"=") {
				return false
			}
		}
	}

	// Validate all flags are in the safe list
	if len(config.SafeFlags) > 0 {
		for i := 0; i < len(cmdArgs); i++ {
			arg := cmdArgs[i]
			if !strings.HasPrefix(arg, "-") {
				continue // positional arg, skip
			}

			// Handle --flag=value
			flagName := arg
			if idx := strings.Index(arg, "="); idx > 0 {
				flagName = arg[:idx]
			}

			flagType, known := config.SafeFlags[flagName]
			if !known {
				// Try combined short flags: -nE → -n + -E
				if len(flagName) > 2 && flagName[0] == '-' && flagName[1] != '-' {
					allKnown := true
					for _, c := range flagName[1:] {
						if _, ok := config.SafeFlags["-"+string(c)]; !ok {
							allKnown = false
							break
						}
					}
					if allKnown {
						continue
					}
				}
				return false // unknown flag = not read-only
			}

			// Skip the flag's argument if it takes one
			if flagType != FlagNone && !strings.Contains(arg, "=") && i+1 < len(cmdArgs) {
				i++ // consume the argument
			}
		}
	}

	return true
}

// SplitCompoundCommand splits a command on ;, &&, ||, | operators.
// Returns individual commands that should each be validated separately.
func SplitCompoundCommand(command string) []string {
	var commands []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	depth := 0 // () depth

	for i := 0; i < len(command); i++ {
		c := command[i]

		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			current.WriteByte(c)
			continue
		}

		if c == '\'' && !inDouble {
			inSingle = !inSingle
			current.WriteByte(c)
			continue
		}

		if c == '"' && !inSingle {
			inDouble = !inDouble
			current.WriteByte(c)
			continue
		}

		if inSingle || inDouble {
			current.WriteByte(c)
			continue
		}

		if c == '(' {
			depth++
			current.WriteByte(c)
			continue
		}
		if c == ')' {
			depth--
			current.WriteByte(c)
			continue
		}

		if depth > 0 {
			current.WriteByte(c)
			continue
		}

		// Check for operators
		if c == ';' {
			if s := strings.TrimSpace(current.String()); s != "" {
				commands = append(commands, s)
			}
			current.Reset()
			continue
		}

		if c == '&' && i+1 < len(command) && command[i+1] == '&' {
			if s := strings.TrimSpace(current.String()); s != "" {
				commands = append(commands, s)
			}
			current.Reset()
			i++ // skip second &
			continue
		}

		if c == '|' && i+1 < len(command) && command[i+1] == '|' {
			if s := strings.TrimSpace(current.String()); s != "" {
				commands = append(commands, s)
			}
			current.Reset()
			i++ // skip second |
			continue
		}

		if c == '|' {
			if s := strings.TrimSpace(current.String()); s != "" {
				commands = append(commands, s)
			}
			current.Reset()
			continue
		}

		current.WriteByte(c)
	}

	if s := strings.TrimSpace(current.String()); s != "" {
		commands = append(commands, s)
	}

	return commands
}

// splitCommandArgs splits a command into tokens respecting quotes.
func splitCommandArgs(command string) []string {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, c := range command {
		if escaped {
			current.WriteRune(c)
			escaped = false
			continue
		}
		if c == '\\' && !inSingle {
			escaped = true
			continue
		}
		if c == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if c == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if c == ' ' && !inSingle && !inDouble {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(c)
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// ValidateCompoundCommand validates the full command AND each subcommand.
func ValidateCompoundCommand(command string, workDir string) *SecurityCheckResult {
	// First: validate the full command (catches cross-pipe patterns)
	if result := ValidateBashSecurity(command, workDir); result != nil {
		return result
	}
	// Then: validate each subcommand individually
	parts := SplitCompoundCommand(command)
	for _, part := range parts {
		if result := ValidateBashSecurity(part, workDir); result != nil {
			result.Reason = fmt.Sprintf("In subcommand '%s': %s",
				truncateForLog(part, 50), result.Reason)
			return result
		}
	}
	return nil
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// isCompoundCommand checks if a command contains operators.
func isCompoundCommand(cmd string) bool {
	return strings.ContainsAny(cmd, ";|&")
}

// Ensure regexp import is used
var _ = regexp.MustCompile
