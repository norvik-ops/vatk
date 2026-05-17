// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package scheduledreports

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// computeNextRunFrom is a testable variant of ComputeNextRun that accepts a
// reference time instead of calling time.Now(). Tests must use this function.
func computeNextRunFrom(schedule string, now time.Time) time.Time {
	switch schedule {
	case "weekly":
		daysUntilMonday := (8 - int(now.Weekday())) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		next := now.AddDate(0, 0, daysUntilMonday)
		return time.Date(next.Year(), next.Month(), next.Day(), 8, 0, 0, 0, time.UTC)
	case "monthly":
		return time.Date(now.Year(), now.Month()+1, 1, 8, 0, 0, 0, time.UTC)
	case "quarterly":
		month := now.Month()
		var nextQMonth time.Month
		var nextQYear int
		switch {
		case month < time.April:
			nextQMonth = time.April
			nextQYear = now.Year()
		case month < time.July:
			nextQMonth = time.July
			nextQYear = now.Year()
		case month < time.October:
			nextQMonth = time.October
			nextQYear = now.Year()
		default:
			nextQMonth = time.January
			nextQYear = now.Year() + 1
		}
		return time.Date(nextQYear, nextQMonth, 1, 8, 0, 0, 0, time.UTC)
	default:
		return now.AddDate(0, 0, 7).Truncate(time.Hour)
	}
}

// TestComputeNextRun_Weekly verifies that the next run for "weekly" is always
// the coming Monday at 08:00 UTC, regardless of the current day.
func TestComputeNextRun_Weekly(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		wantDay  time.Weekday
		wantHour int
	}{
		{
			name:     "from Wednesday should be next Monday",
			now:      time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC), // Wednesday
			wantDay:  time.Monday,
			wantHour: 8,
		},
		{
			name:     "from Monday should skip to following Monday",
			now:      time.Date(2026, 5, 11, 8, 0, 0, 0, time.UTC), // Monday
			wantDay:  time.Monday,
			wantHour: 8,
		},
		{
			name:     "from Sunday should be next Monday (tomorrow)",
			now:      time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC), // Sunday
			wantDay:  time.Monday,
			wantHour: 8,
		},
		{
			name:     "from Saturday should be next Monday (in 2 days)",
			now:      time.Date(2026, 5, 16, 6, 0, 0, 0, time.UTC), // Saturday
			wantDay:  time.Monday,
			wantHour: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeNextRunFrom("weekly", tt.now)
			assert.Equal(t, tt.wantDay, got.Weekday(),
				"next run should be on Monday")
			assert.Equal(t, tt.wantHour, got.Hour(),
				"next run should be at 08:00")
			assert.Equal(t, 0, got.Minute())
			assert.Equal(t, 0, got.Second())
			assert.True(t, got.After(tt.now),
				"next run must be strictly in the future")
		})
	}
}

// TestComputeNextRun_Monthly verifies that the next run for "monthly" is the
// 1st of the following month at 08:00 UTC.
func TestComputeNextRun_Monthly(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		wantYear  int
		wantMonth time.Month
		wantDay   int
	}{
		{
			name:      "mid-month should advance to 1st of next month",
			now:       time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC),
			wantYear:  2026,
			wantMonth: time.June,
			wantDay:   1,
		},
		{
			name:      "first of month should advance to 1st of next month",
			now:       time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC),
			wantYear:  2026,
			wantMonth: time.June,
			wantDay:   1,
		},
		{
			name:      "end of December should wrap to January next year",
			now:       time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.January,
			wantDay:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeNextRunFrom("monthly", tt.now)
			assert.Equal(t, tt.wantYear, got.Year())
			assert.Equal(t, tt.wantMonth, got.Month())
			assert.Equal(t, tt.wantDay, got.Day())
			assert.Equal(t, 8, got.Hour())
			assert.Equal(t, 0, got.Minute())
			assert.True(t, got.After(tt.now))
		})
	}
}

// TestComputeNextRun_Quarterly verifies that the next run for "quarterly" lands
// on the first day of the next calendar quarter at 08:00 UTC.
func TestComputeNextRun_Quarterly(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		wantYear  int
		wantMonth time.Month
		wantDay   int
	}{
		{
			name:      "Q1 (January) → next quarter is April",
			now:       time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC),
			wantYear:  2026,
			wantMonth: time.April,
			wantDay:   1,
		},
		{
			name:      "Q1 (March) → next quarter is April",
			now:       time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC),
			wantYear:  2026,
			wantMonth: time.April,
			wantDay:   1,
		},
		{
			name:      "Q2 (May) → next quarter is July",
			now:       time.Date(2026, 5, 17, 9, 0, 0, 0, time.UTC),
			wantYear:  2026,
			wantMonth: time.July,
			wantDay:   1,
		},
		{
			name:      "Q3 (August) → next quarter is October",
			now:       time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			wantYear:  2026,
			wantMonth: time.October,
			wantDay:   1,
		},
		{
			name:      "Q4 (November) → next quarter is January next year",
			now:       time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.January,
			wantDay:   1,
		},
		{
			name:      "Q4 (December) → next quarter is January next year",
			now:       time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.January,
			wantDay:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeNextRunFrom("quarterly", tt.now)
			assert.Equal(t, tt.wantYear, got.Year())
			assert.Equal(t, tt.wantMonth, got.Month())
			assert.Equal(t, tt.wantDay, got.Day())
			assert.Equal(t, 8, got.Hour())
			assert.Equal(t, 0, got.Minute())
			assert.True(t, got.After(tt.now))
		})
	}
}

// TestComputeNextRun_Unknown verifies that an unknown schedule returns a
// time approximately 7 days in the future.
func TestComputeNextRun_Unknown(t *testing.T) {
	now := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
	got := computeNextRunFrom("bogus", now)
	expected := now.AddDate(0, 0, 7).Truncate(time.Hour)
	assert.Equal(t, expected, got)
}
