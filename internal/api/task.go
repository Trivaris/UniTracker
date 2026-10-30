package api

import (
	"strings"
	"time"
)

type TaskID string

type TaskState string

const (
	TaskStateNotStarted TaskState = "not_started"
	TaskStateInProgress TaskState = "in_progress"
	TaskStateFinished   TaskState = "finished"
)

type Task struct {
	ID                 TaskID     `json:"id"`
	Name               string     `json:"name"`
	SubjectID          SubjectID  `json:"subject_id"`
	StartDate          *time.Time `json:"start_date,omitempty"`
	Deadline           *time.Time `json:"deadline,omitempty"`
	IsRecurring        bool       `json:"is_recurring"`
	RecurrenceInterval string     `json:"recurrence_interval,omitempty"`
	RecurrenceUntil    *time.Time `json:"recurrence_until,omitempty"`
	State              TaskState  `json:"state"`
	SheetURL           string     `json:"sheet_url,omitempty"`
	SolutionURL        string     `json:"solution_url,omitempty"`
	PossiblePoints     *float64   `json:"possible_points,omitempty"`
	AchievedPoints     *float64   `json:"achieved_points,omitempty"`
}

func NormalizeTaskState(value string) TaskState {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(TaskStateInProgress):
		return TaskStateInProgress
	case string(TaskStateFinished):
		return TaskStateFinished
	default:
		return TaskStateNotStarted
	}
}

func (t *Task) Normalize() {
	if t == nil {
		return
	}
	if t.State == "" {
		t.State = TaskStateNotStarted
	} else {
		t.State = NormalizeTaskState(string(t.State))
	}
	if t.RecurrenceInterval != "" {
		t.RecurrenceInterval = strings.ToLower(strings.TrimSpace(t.RecurrenceInterval))
	}
}

func (t Task) RecurrenceLabel() string {
	if !t.IsRecurring || t.RecurrenceInterval == "" {
		return ""
	}
	label := strings.ToLower(strings.TrimSpace(t.RecurrenceInterval))
	if t.RecurrenceUntil != nil {
		return label + " until " + t.RecurrenceUntil.Format("2006-01-02")
	}
	return label
}

type TaskView struct {
	*Task
	SubjectName string
}

func DateOnlyString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateOnly)
}
