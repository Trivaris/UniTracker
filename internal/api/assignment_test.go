package api

import (
	"testing"
	"time"
)

func TestOccurrenceDatesUseIntervalAndUntilDate(t *testing.T) {
	start := time.Date(2026, time.October, 5, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, time.October, 30, 0, 0, 0, 0, time.UTC)
	series := AssignmentSeries{StartDate: &start, RecurrenceInterval: "weekly", RecurrenceUntil: &until}

	dates := series.OccurrenceDates(time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC))
	if len(dates) != 4 {
		t.Fatalf("expected four weekly releases before the until date, got %d", len(dates))
	}
	if !dates[0].Equal(time.Date(2026, time.October, 5, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected the default release time on the first date, got %s", dates[0])
	}

	series.RecurrenceInterval = "biweekly"
	dates = series.OccurrenceDates(time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC))
	if len(dates) != 2 {
		t.Fatalf("expected two biweekly releases before the until date, got %d", len(dates))
	}
}
