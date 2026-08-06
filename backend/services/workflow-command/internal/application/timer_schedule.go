package application

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// timerSchedule describes when a timer should fire and whether it should repeat.
// RepeatCount: 1 = one-shot, N>1 = fire N times total, -1 = infinite (R/PT…).
type timerSchedule struct {
	DueDate     time.Time
	Interval    time.Duration
	RepeatCount int
}

func resolveTimerSchedule(props map[string]any, now time.Time) (*timerSchedule, error) {
	if props == nil {
		return nil, fmt.Errorf("timer properties required")
	}

	if cycle, _ := props["timer_cycle"].(string); strings.TrimSpace(cycle) != "" {
		return parseISO8601Cycle(strings.TrimSpace(cycle), now)
	}

	if date, _ := props["timer_date"].(string); strings.TrimSpace(date) != "" {
		due, err := parseAbsoluteTime(strings.TrimSpace(date))
		if err != nil {
			return nil, err
		}
		return &timerSchedule{DueDate: due, RepeatCount: 1}, nil
	}

	durationStr, _ := props["timer_duration"].(string)
	durationStr = strings.TrimSpace(durationStr)
	if durationStr == "" {
		return nil, fmt.Errorf("no timer_duration, timer_date, or timer_cycle")
	}

	// Compat: parser previously stuffed timeDate into timer_duration.
	if strings.HasPrefix(durationStr, "R") {
		return parseISO8601Cycle(durationStr, now)
	}
	if due, err := parseAbsoluteTime(durationStr); err == nil {
		return &timerSchedule{DueDate: due, RepeatCount: 1}, nil
	}
	d, err := parseISO8601Duration(durationStr)
	if err != nil {
		return nil, err
	}
	return &timerSchedule{DueDate: now.Add(d), Interval: d, RepeatCount: 1}, nil
}

// parseISO8601Cycle supports R[n]/PT… and R/PT… (infinite).
func parseISO8601Cycle(cycle string, now time.Time) (*timerSchedule, error) {
	cycle = strings.TrimSpace(cycle)
	if cycle == "" || cycle[0] != 'R' {
		return nil, fmt.Errorf("unsupported timeCycle (must start with R): %s", cycle)
	}
	rest := cycle[1:]
	if rest == "" {
		return nil, fmt.Errorf("unsupported timeCycle: %s", cycle)
	}
	if rest[0] == '/' {
		// R/PT10S
		interval, err := parseISO8601Duration(rest[1:])
		if err != nil {
			return nil, err
		}
		return &timerSchedule{DueDate: now.Add(interval), Interval: interval, RepeatCount: -1}, nil
	}
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return nil, fmt.Errorf("unsupported timeCycle (expected R[n]/PT…): %s", cycle)
	}
	n, err := strconv.Atoi(rest[:slash])
	if err != nil || n < 1 {
		return nil, fmt.Errorf("invalid timeCycle repeat count: %s", cycle)
	}
	interval, err := parseISO8601Duration(rest[slash+1:])
	if err != nil {
		return nil, err
	}
	return &timerSchedule{DueDate: now.Add(interval), Interval: interval, RepeatCount: n}, nil
}

func parseAbsoluteTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty timeDate")
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	// Common BPMN export without timezone → treat as UTC.
	if t, err := time.Parse("2006-01-02T15:04:05", value); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unsupported timeDate format: %s", value)
}

func cycleIntervalFromProps(props map[string]any) time.Duration {
	if props == nil {
		return 0
	}
	if cycle, _ := props["timer_cycle"].(string); strings.TrimSpace(cycle) != "" {
		if sched, err := parseISO8601Cycle(strings.TrimSpace(cycle), time.Now()); err == nil {
			return sched.Interval
		}
	}
	if dur, _ := props["timer_duration"].(string); strings.HasPrefix(strings.TrimSpace(dur), "R") {
		if sched, err := parseISO8601Cycle(strings.TrimSpace(dur), time.Now()); err == nil {
			return sched.Interval
		}
	}
	return 0
}
