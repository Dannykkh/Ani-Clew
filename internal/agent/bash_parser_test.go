package agent

import (
	"testing"
)

func TestParseCommand_QuoteDetection(t *testing.T) {
	tests := []struct {
		cmd         string
		quoteErrors int
		subshell    bool
		redirect    bool
		pipe        bool
		compound    bool
	}{
		// Simple
		{"ls -la", 0, false, false, false, false},
		{"echo hello", 0, false, false, false, false},

		// Quoted
		{"echo 'hello world'", 0, false, false, false, false},
		{`echo "hello world"`, 0, false, false, false, false},

		// Subshell
		{"echo $(whoami)", 0, true, false, false, false},
		{"echo `date`", 0, true, false, false, false},

		// Redirect
		{"echo foo > file.txt", 0, false, true, false, false},
		{"cat < input.txt", 0, false, true, false, false},

		// Pipe
		{"ls | grep foo", 0, false, false, true, false},

		// Compound
		{"ls && echo done", 0, false, false, false, true},
		{"ls || echo fail", 0, false, false, true, true},
		{"ls; echo done", 0, false, false, false, true},

		// Unterminated quotes
		{"echo 'hello", 1, false, false, false, false},
		{`echo "hello`, 1, false, false, false, false},

		// Subshell in double quotes (should detect)
		{`echo "$(whoami)"`, 0, true, false, false, false},

		// Semicolon in single quotes (NOT compound)
		{"echo 'a;b'", 0, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			parsed := ParseCommand(tt.cmd)
			if len(parsed.QuoteErrors) != tt.quoteErrors {
				t.Errorf("QuoteErrors: got %d, want %d (%v)", len(parsed.QuoteErrors), tt.quoteErrors, parsed.QuoteErrors)
			}
			if parsed.HasSubshell != tt.subshell {
				t.Errorf("HasSubshell: got %v, want %v", parsed.HasSubshell, tt.subshell)
			}
			if parsed.HasRedirect != tt.redirect {
				t.Errorf("HasRedirect: got %v, want %v", parsed.HasRedirect, tt.redirect)
			}
			if parsed.HasPipe != tt.pipe {
				t.Errorf("HasPipe: got %v, want %v", parsed.HasPipe, tt.pipe)
			}
			if parsed.IsCompound != tt.compound {
				t.Errorf("IsCompound: got %v, want %v", parsed.IsCompound, tt.compound)
			}
		})
	}
}

func TestParseCommand_UnquotedContent(t *testing.T) {
	tests := []struct {
		cmd      string
		unquoted string
	}{
		{"echo hello", "echo hello"},
		{"echo 'hello world'", "echo "},
		{`echo "a" b`, "echo  b"},
		{"cat 'file;rm -rf /'", "cat "},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			parsed := ParseCommand(tt.cmd)
			got := parsed.UnquotedContent()
			if got != tt.unquoted {
				t.Errorf("UnquotedContent: got %q, want %q", got, tt.unquoted)
			}
		})
	}
}

func TestParseCommand_HasUnquotedPattern(t *testing.T) {
	tests := []struct {
		cmd     string
		pattern string
		found   bool
	}{
		{"echo $(whoami)", "$(", true},
		{"echo '$(whoami)'", "$(", false}, // inside single quotes
		{"rm -rf /", "rm", true},
		{"echo 'rm -rf /'", "rm", false}, // inside quotes
	}

	for _, tt := range tests {
		t.Run(tt.cmd+"_"+tt.pattern, func(t *testing.T) {
			parsed := ParseCommand(tt.cmd)
			got := parsed.HasUnquotedPattern(tt.pattern)
			if got != tt.found {
				t.Errorf("HasUnquotedPattern(%q): got %v, want %v", tt.pattern, got, tt.found)
			}
		})
	}
}
