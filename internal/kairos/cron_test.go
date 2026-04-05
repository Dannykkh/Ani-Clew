package kairos

import (
	"testing"
	"time"
)

func TestParseCron(t *testing.T) {
	tests := []struct {
		expr    string
		wantErr bool
	}{
		{"* * * * *", false},
		{"0 * * * *", false},
		{"*/5 * * * *", false},
		{"0 0 * * *", false},
		{"30 2 * * 1-5", false},
		{"0,15,30,45 * * * *", false},
		{"@hourly", false},
		{"@daily", false},
		{"@weekly", false},
		{"@monthly", false},
		{"@every 5m", false},
		{"@every 2h", false},
		{"bad", true},
		{"* * *", true},
		{"60 * * * *", true}, // minute > 59
		{"* 25 * * *", true}, // hour > 23
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.expr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.expr, err)
			}
		})
	}
}

func TestCronMatches(t *testing.T) {
	// Monday 2026-04-06 14:30:00
	testTime := time.Date(2026, 4, 6, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		expr    string
		matches bool
	}{
		{"* * * * *", true},           // every minute
		{"30 * * * *", true},          // minute 30
		{"0 * * * *", false},          // minute 0 only
		{"30 14 * * *", true},         // 14:30
		{"30 14 6 * *", true},         // 14:30 on 6th
		{"30 14 6 4 *", true},         // 14:30 on Apr 6
		{"30 14 * * 1", true},         // 14:30 on Monday (1)
		{"30 14 * * 0", false},        // 14:30 on Sunday
		{"*/15 * * * *", true},        // every 15 min (0,15,30,45)
		{"*/10 * * * *", true},        // every 10 min (0,10,20,30,40,50)
		{"*/7 * * * *", false},        // every 7 min (0,7,14,21,28,35,...) — 30 not in list
		{"0 0 * * *", false},          // midnight only
		{"@hourly", false},            // minute 0 only
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			cron, err := ParseCron(tt.expr)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if cron.Matches(testTime) != tt.matches {
				t.Errorf("CronExpr(%q).Matches(%v) = %v, want %v", tt.expr, testTime, !tt.matches, tt.matches)
			}
		})
	}
}

func TestIsDue(t *testing.T) {
	now := time.Date(2026, 4, 6, 14, 30, 30, 0, time.UTC) // 14:30:30
	justRan := time.Date(2026, 4, 6, 14, 30, 5, 0, time.UTC) // 14:30:05 — same minute
	ranBefore := now.Add(-5 * time.Minute) // ran 5 min ago

	tests := []struct {
		name     string
		cron     string
		lastRun  time.Time
		expected bool
	}{
		{"matching cron, never ran", "30 14 * * *", time.Time{}, true},
		{"matching cron, ran same minute", "30 14 * * *", justRan, false},
		{"matching cron, ran before", "30 14 * * *", ranBefore, true},
		{"non-matching cron", "0 15 * * *", time.Time{}, false},
		{"empty cron, never ran", "", time.Time{}, true},
		{"empty cron, already ran", "", ranBefore, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDue(tt.cron, tt.lastRun, now)
			if result != tt.expected {
				t.Errorf("IsDue(%q, %v, %v) = %v, want %v", tt.cron, tt.lastRun, now, result, tt.expected)
			}
		})
	}
}
