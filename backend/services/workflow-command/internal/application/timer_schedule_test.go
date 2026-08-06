package application

import (
	"testing"
	"time"
)

func TestResolveTimerSchedule_DurationDateCycle(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

	dur, err := resolveTimerSchedule(map[string]any{"timer_duration": "PT10S"}, now)
	if err != nil {
		t.Fatalf("duration: %v", err)
	}
	if !dur.DueDate.Equal(now.Add(10*time.Second)) || dur.RepeatCount != 1 {
		t.Fatalf("unexpected duration schedule: %+v", dur)
	}

	past := "2020-01-01T00:00:00Z"
	date, err := resolveTimerSchedule(map[string]any{"timer_date": past}, now)
	if err != nil {
		t.Fatalf("date: %v", err)
	}
	if date.DueDate.Year() != 2020 || date.RepeatCount != 1 {
		t.Fatalf("unexpected date schedule: %+v", date)
	}

	// Compat: absolute value stored in timer_duration
	compat, err := resolveTimerSchedule(map[string]any{"timer_duration": past}, now)
	if err != nil || compat.DueDate.Year() != 2020 {
		t.Fatalf("compat date-in-duration: err=%v sched=%+v", err, compat)
	}

	cycle, err := resolveTimerSchedule(map[string]any{"timer_cycle": "R3/PT5S"}, now)
	if err != nil {
		t.Fatalf("cycle: %v", err)
	}
	if cycle.RepeatCount != 3 || cycle.Interval != 5*time.Second {
		t.Fatalf("unexpected cycle: %+v", cycle)
	}

	inf, err := parseISO8601Cycle("R/PT1S", now)
	if err != nil || inf.RepeatCount != -1 {
		t.Fatalf("infinite cycle: err=%v sched=%+v", err, inf)
	}
}
