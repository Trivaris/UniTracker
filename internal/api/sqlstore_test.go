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
