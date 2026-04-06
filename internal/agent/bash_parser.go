package agent

import (
	"strings"
)

// QuoteState tracks the quoting context during bash command parsing.
type QuoteState int

const (
	QuoteNone   QuoteState = iota
	QuoteSingle            // inside '...'
	QuoteDouble            // inside "..."
	QuoteDollar            // inside $'...' (ANSI-C quoting)
)

// ParsedCommand represents a parsed bash command with quote-aware analysis.
type ParsedCommand struct {
	Raw          string
	Segments     []CommandSegment
	QuoteErrors  []string // mismatched quotes, unterminated strings
	HasSubshell  bool     // $(), backticks
	HasRedirect  bool     // >, >>, <
	HasPipe      bool     // |
	HasBackground bool    // &
	IsCompound   bool     // ;, &&, ||
}

// CommandSegment is a parsed portion of the command.
type CommandSegment struct {
	Text      string
	Quoted    bool
	QuoteType QuoteState
	Start     int
	End       int
}

// ParseCommand performs quote-aware parsing of a bash command.
// This is a pure Go implementation that handles the cases tree-sitter would catch.
func ParseCommand(cmd string) ParsedCommand {
	result := ParsedCommand{Raw: cmd}

	state := QuoteNone
	escaped := false
	var segments []CommandSegment
	var current strings.Builder
	segStart := 0

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]

		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}

		switch state {
		case QuoteNone:
			switch c {
			case '\\':
				escaped = true
				current.WriteByte(c)
			case '\'':
				if current.Len() > 0 {
					segments = append(segments, CommandSegment{
						Text: current.String(), Quoted: false, QuoteType: QuoteNone,
						Start: segStart, End: i,
					})
					current.Reset()
				}
				state = QuoteSingle
				segStart = i
			case '"':
				if current.Len() > 0 {
					segments = append(segments, CommandSegment{
						Text: current.String(), Quoted: false, QuoteType: QuoteNone,
						Start: segStart, End: i,
					})
					current.Reset()
				}
				state = QuoteDouble
				segStart = i
			case '$':
				if i+1 < len(cmd) && cmd[i+1] == '\'' {
					state = QuoteDollar
					segStart = i
					i++ // skip the '
				} else if i+1 < len(cmd) && cmd[i+1] == '(' {
					result.HasSubshell = true
					current.WriteByte(c)
				} else {
					current.WriteByte(c)
				}
			case '`':
				result.HasSubshell = true
				current.WriteByte(c)
			case '|':
				result.HasPipe = true
				if i+1 < len(cmd) && cmd[i+1] == '|' {
					result.IsCompound = true
				}
				current.WriteByte(c)
			case '&':
				if i+1 < len(cmd) && cmd[i+1] == '&' {
					result.IsCompound = true
				} else {
					result.HasBackground = true
				}
				current.WriteByte(c)
			case ';':
				result.IsCompound = true
				current.WriteByte(c)
			case '>', '<':
				result.HasRedirect = true
				current.WriteByte(c)
			default:
				current.WriteByte(c)
			}

		case QuoteSingle:
			if c == '\'' {
				segments = append(segments, CommandSegment{
					Text: current.String(), Quoted: true, QuoteType: QuoteSingle,
					Start: segStart, End: i + 1,
				})
				current.Reset()
				state = QuoteNone
				segStart = i + 1
			} else {
				current.WriteByte(c)
			}

		case QuoteDouble:
			if c == '\\' && i+1 < len(cmd) {
				next := cmd[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' {
					escaped = true
					current.WriteByte(c)
					continue
				}
			}
			if c == '"' {
				segments = append(segments, CommandSegment{
					Text: current.String(), Quoted: true, QuoteType: QuoteDouble,
					Start: segStart, End: i + 1,
				})
				current.Reset()
				state = QuoteNone
				segStart = i + 1
			} else {
				if c == '$' && i+1 < len(cmd) && cmd[i+1] == '(' {
					result.HasSubshell = true
				}
				current.WriteByte(c)
			}

		case QuoteDollar:
			if c == '\'' {
				segments = append(segments, CommandSegment{
					Text: current.String(), Quoted: true, QuoteType: QuoteDollar,
					Start: segStart, End: i + 1,
				})
				current.Reset()
				state = QuoteNone
				segStart = i + 1
			} else {
				current.WriteByte(c)
			}
		}
	}

	// Remaining content
	if current.Len() > 0 {
		segments = append(segments, CommandSegment{
			Text: current.String(), Quoted: false, QuoteType: QuoteNone,
			Start: segStart, End: len(cmd),
		})
	}

	// Check for unterminated quotes
	if state != QuoteNone {
		switch state {
		case QuoteSingle:
			result.QuoteErrors = append(result.QuoteErrors, "Unterminated single quote")
		case QuoteDouble:
			result.QuoteErrors = append(result.QuoteErrors, "Unterminated double quote")
		case QuoteDollar:
			result.QuoteErrors = append(result.QuoteErrors, "Unterminated $' quote")
		}
	}

	result.Segments = segments
	return result
}

// UnquotedContent returns only the unquoted portions of the command.
func (p *ParsedCommand) UnquotedContent() string {
	var sb strings.Builder
	for _, seg := range p.Segments {
		if !seg.Quoted {
			sb.WriteString(seg.Text)
		}
	}
	return sb.String()
}

// HasUnquotedPattern checks if a pattern appears in unquoted context.
func (p *ParsedCommand) HasUnquotedPattern(pattern string) bool {
	return strings.Contains(p.UnquotedContent(), pattern)
}

// IsFullyQuoted returns true if the entire command is within quotes.
func (p *ParsedCommand) IsFullyQuoted() bool {
	for _, seg := range p.Segments {
		if !seg.Quoted && strings.TrimSpace(seg.Text) != "" {
			return false
		}
	}
	return true
}
