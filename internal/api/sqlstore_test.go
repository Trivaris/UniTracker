package api

import (
	"context"
	"testing"
	"time"
)

func TestSQLiteStoreTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLStore(ctx, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	subject, err := store.StoreSubject(ctx, "Algorithms")
	if err != nil {
		t.Fatalf("store subject: %v", err)
	}

	start := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	deadline := time.Date(2026, time.October, 10, 0, 0, 0, 0, time.UTC)
	possible := 10.0
	achieved := 6.5

	task, err := store.StoreTask(ctx, Task{
		Name:           "Exercise Sheet 1",
		SubjectID:      subject.ID,
		StartDate:      &start,
		Deadline:       &deadline,
		IsRecurring:    false,
		State:          TaskStateNotStarted,
		SheetURL:       "https://example.com/sheet",
		PossiblePoints: &possible,
		AchievedPoints: &achieved,
	})
	if err != nil {
		t.Fatalf("store task: %v", err)
	}

	if task.State != TaskStateNotStarted {
		t.Fatalf("expected default state %q, got %q", TaskStateNotStarted, task.State)
	}

	stored, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if stored.Name != "Exercise Sheet 1" {
		t.Fatalf("expected name to persist, got %q", stored.Name)
	}
	if stored.SheetURL != "https://example.com/sheet" {
		t.Fatalf("expected sheet URL to persist, got %q", stored.SheetURL)
	}

	if err = store.UpdateTaskState(ctx, task.ID, TaskStateFinished); err != nil {
		t.Fatalf("update state: %v", err)
	}
	updated, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get updated task: %v", err)
	}
	if updated.State != TaskStateFinished {
		t.Fatalf("expected finished state, got %q", updated.State)
	}

	if err = store.UpdateTaskPoints(ctx, task.ID, nil, nil); err != nil {
		t.Fatalf("update points: %v", err)
	}
}

func TestSQLiteAssignmentSeriesGeneratesOccurrencesAndExercises(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLStore(ctx, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	subject, err := store.StoreSubject(ctx, "Algorithms")
	if err != nil {
		t.Fatalf("store subject: %v", err)
	}
	start := time.Now().UTC().AddDate(0, 0, -14)
	until := time.Now().UTC().AddDate(0, 0, 1)
	possible := 10.0
	series, err := store.StoreAssignmentSeries(ctx, AssignmentSeries{
		Name: "Exercise Sheet", SubjectID: subject.ID, Kind: AssignmentKindSheet,
		Requirement: AssignmentRequired, StartDate: &start, RecurrenceInterval: "weekly",
		RecurrenceUntil: &until, DeadlineWeekday: time.Friday, DeadlineTime: "18:00",
		Exercises: []Exercise{{Name: "Induction", Number: 1, PossiblePoints: &possible}},
	})
	if err != nil {
		t.Fatalf("store series: %v", err)
	}

	occurrences, err := store.GetAssignmentOccurrences(ctx, subject.ID)
	if err != nil {
		t.Fatalf("get occurrences: %v", err)
	}
	if len(occurrences) < 2 {
		t.Fatalf("expected at least two occurrences, got %d", len(occurrences))
	}
	if len(occurrences[0].Exercises) != 1 {
		t.Fatalf("expected one exercise, got %d", len(occurrences[0].Exercises))
	}
	if occurrences[0].SeriesID != series.ID {
		t.Fatalf("expected series %q, got %q", series.ID, occurrences[0].SeriesID)
	}

	achieved := 8.0
	if err := store.UpdateExercisePoints(ctx, occurrences[0].Exercises[0].ID, &achieved, nil); err != nil {
		t.Fatalf("update exercise: %v", err)
	}
	updated, err := store.GetAssignmentOccurrences(ctx, subject.ID)
	if err != nil {
		t.Fatalf("reload occurrences: %v", err)
	}
	if updated[0].Exercises[0].AchievedPoints == nil || *updated[0].Exercises[0].AchievedPoints != achieved {
		t.Fatalf("expected achieved points %.1f", achieved)
	}
}
