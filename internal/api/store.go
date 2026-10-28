package api

import (
	"context"
	"errors"
)

var ErrSubjectNotFound error = errors.New("subject not found")
var ErrTaskNotFound error = errors.New("task not found")

type Store interface {
	StoreTask(context.Context, string, SubjectID) (*Task, error)
	StoreSubject(context.Context, string) (*Subject, error)
	GetSubject(context.Context, SubjectID) (*Subject, error)
	GetTask(context.Context, TaskID) (*Task, error)
	GetAllTasks(context.Context) ([]*Task, error)
	GetAllSubjects(context.Context) ([]*Subject, error)
}
