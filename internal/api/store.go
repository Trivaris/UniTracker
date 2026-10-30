package api

import (
	"context"
	"errors"
)

var ErrSubjectNotFound = errors.New("subject not found")
var ErrTaskNotFound = errors.New("task not found")

type Store interface {
	StoreTask(ctx context.Context, task Task) (*Task, error)
	StoreAssignmentSeries(ctx context.Context, series AssignmentSeries) (*AssignmentSeries, error)
	GetAssignmentSeries(ctx context.Context, id AssignmentSeriesID) (*AssignmentSeries, error)
	GetAssignmentOccurrences(ctx context.Context, subjectID SubjectID) ([]AssignmentOccurrence, error)
	UpdateExercisePoints(ctx context.Context, id ExerciseID, achieved *float64, possible *float64) error
	StoreSubject(ctx context.Context, name string) (*Subject, error)
	GetSubject(ctx context.Context, id SubjectID) (*Subject, error)
	GetTask(ctx context.Context, id TaskID) (*Task, error)
	GetAllTasks(ctx context.Context) ([]*Task, error)
	GetAllSubjects(ctx context.Context) ([]*Subject, error)
	DeleteSubject(ctx context.Context, id SubjectID) error
	DeleteTask(ctx context.Context, id TaskID) error
	UpdateTaskState(ctx context.Context, id TaskID, state TaskState) error
	UpdateTaskPoints(ctx context.Context, id TaskID, achieved *float64, possible *float64) error
}
