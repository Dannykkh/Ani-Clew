package kairos

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronExpr represents a parsed cron expression.
// Format: "minute hour day month weekday"
// Supports: *, */N, N, N-M, N,M,O
type CronExpr struct {
	Minutes  []int // 0-59
	Hours    []int // 0-23
	Days     []int // 1-31
	Months   []int // 1-12
	Weekdays []int // 0-6 (0=Sunday)
}

// ParseCron parses a 5-field cron expression.
func ParseCron(expr string) (*CronExpr, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty cron expression")
	}

	// Handle presets
	switch expr {
	case "@hourly":
		expr = "0 * * * *"
	case "@daily", "@midnight":
		expr = "0 0 * * *"
	case "@weekly":
		expr = "0 0 * * 0"
	case "@monthly":
		expr = "0 0 1 * *"
	}

	// Handle @every Nd/Nh/Nm
	if strings.HasPrefix(expr, "@every ") {
		return parseEvery(expr[7:])
	}

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}

	c := &CronExpr{}
	var err error

	c.Minutes, err = parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	c.Hours, err = parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	c.Days, err = parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day: %w", err)
	}
	c.Months, err = parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	c.Weekdays, err = parseField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("weekday: %w", err)
	}

	return c, nil
}

// Matches checks if the given time matches the cron expression.
func (c *CronExpr) Matches(t time.Time) bool {
	return contains(c.Minutes, t.Minute()) &&
		contains(c.Hours, t.Hour()) &&
		contains(c.Days, t.Day()) &&
		contains(c.Months, int(t.Month())) &&
		contains(c.Weekdays, int(t.Weekday()))
}

// IsDue checks if the task should run at the given time, considering the last run.
func IsDue(cronExpr string, lastRun time.Time, now time.Time) bool {
	if cronExpr == "" {
		// No cron = run every tick if never run
		return lastRun.IsZero()
	}

	cron, err := ParseCron(cronExpr)
	if err != nil {
		return false
	}

	// Check if current minute matches and hasn't run in this minute
	if !cron.Matches(now) {
		return false
	}

	// Don't re-run within the same minute
	if !lastRun.IsZero() && lastRun.Truncate(time.Minute) == now.Truncate(time.Minute) {
		return false
	}

	return true
}

// ── Field parsing ──

func parseField(field string, min, max int) ([]int, error) {
	if field == "*" {
		return makeRange(min, max), nil
	}

	// */N — every N
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step: %s", field)
		}
		var result []int
		for i := min; i <= max; i += step {
			result = append(result, i)
		}
		return result, nil
	}

	// Comma-separated: 1,5,10
	if strings.Contains(field, ",") {
		var result []int
		for _, part := range strings.Split(field, ",") {
			v, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || v < min || v > max {
				return nil, fmt.Errorf("invalid value: %s", part)
			}
			result = append(result, v)
		}
		return result, nil
	}

	// Range: 1-5
	if strings.Contains(field, "-") {
		parts := strings.SplitN(field, "-", 2)
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || start < min || end > max || start > end {
			return nil, fmt.Errorf("invalid range: %s", field)
		}
		return makeRange(start, end), nil
	}

	// Single value
	v, err := strconv.Atoi(field)
	if err != nil || v < min || v > max {
		return nil, fmt.Errorf("invalid value: %s", field)
	}
	return []int{v}, nil
}

func parseEvery(duration string) (*CronExpr, error) {
	// @every 5m, @every 1h, @every 30s
	duration = strings.TrimSpace(duration)
	if strings.HasSuffix(duration, "m") {
		mins, err := strconv.Atoi(duration[:len(duration)-1])
		if err != nil || mins <= 0 {
			return nil, fmt.Errorf("invalid @every duration: %s", duration)
		}
		var minutes []int
		for i := 0; i < 60; i += mins {
			minutes = append(minutes, i)
		}
		return &CronExpr{
			Minutes:  minutes,
			Hours:    makeRange(0, 23),
			Days:     makeRange(1, 31),
			Months:   makeRange(1, 12),
			Weekdays: makeRange(0, 6),
		}, nil
	}
	if strings.HasSuffix(duration, "h") {
		hours, err := strconv.Atoi(duration[:len(duration)-1])
		if err != nil || hours <= 0 {
			return nil, fmt.Errorf("invalid @every duration: %s", duration)
		}
		var hrs []int
		for i := 0; i < 24; i += hours {
			hrs = append(hrs, i)
		}
		return &CronExpr{
			Minutes:  []int{0},
			Hours:    hrs,
			Days:     makeRange(1, 31),
			Months:   makeRange(1, 12),
			Weekdays: makeRange(0, 6),
		}, nil
	}
	return nil, fmt.Errorf("unsupported @every duration: %s (use m or h)", duration)
}

func makeRange(min, max int) []int {
	r := make([]int, max-min+1)
	for i := range r {
		r[i] = min + i
	}
	return r
}

func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
