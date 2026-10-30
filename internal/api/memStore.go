package api

import (
	"context"
	"maps"
	"slices"

	"github.com/google/uuid"
)

type InMemoryStore struct {
	tasks    map[TaskID]*Task
	subjects map[SubjectID]*Subject
}

func NewMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tasks:    map[TaskID]*Task{},
		subjects: map[SubjectID]*Subject{},
	}
}

func (s *InMemoryStore) StoreTask(ctx context.Context, task Task) (*Task, error) {
	task.Normalize()
	id := s.newTaskID()
	task.ID = id
	s.tasks[id] = &task
	return &task, nil
}

func (s *InMemoryStore) StoreSubject(ctx context.Context, name string) (*Subject, error) {
	id := s.newSubjectID()
	subject := &Subject{
		ID:   id,
		Name: name,
	}
	s.subjects[id] = subject
	return subject, nil
}

func (s *InMemoryStore) GetTask(ctx context.Context, id TaskID) (*Task, error) {
	task, ok := s.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (s *InMemoryStore) GetSubject(ctx context.Context, id SubjectID) (*Subject, error) {
	subject, ok := s.subjects[id]
	if !ok {
		return nil, ErrSubjectNotFound
	}
	return subject, nil
}

func (s *InMemoryStore) GetAllSubjects(ctx context.Context) ([]*Subject, error) {
	return slices.Collect(maps.Values(s.subjects)), nil
}

func (s *InMemoryStore) GetAllTasks(ctx context.Context) ([]*Task, error) {
	return slices.Collect(maps.Values(s.tasks)), nil
}

func (s *InMemoryStore) DeleteSubject(ctx context.Context, id SubjectID) error {
	delete(s.subjects, id)
	for taskID, task := range s.tasks {
		if task.SubjectID == id {
			delete(s.tasks, taskID)
		}
	}
	return nil
}

func (s *InMemoryStore) DeleteTask(ctx context.Context, id TaskID) error {
	delete(s.tasks, id)
	return nil
}

func (s *InMemoryStore) UpdateTaskState(ctx context.Context, id TaskID, state TaskState) error {
	task, ok := s.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	task.State = NormalizeTaskState(string(state))
	return nil
}

func (s *InMemoryStore) UpdateTaskPoints(ctx context.Context, id TaskID, achieved *float64, possible *float64) error {
	task, ok := s.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}
	if achieved != nil {
		task.AchievedPoints = achieved
	}
	if possible != nil {
		task.PossiblePoints = possible
	}
	return nil
}

func (s *InMemoryStore) newTaskID() TaskID {
	id, err := uuid.NewV7()
	if err != nil {
		return "-1"
	}
	return TaskID(id.String())
}

func (s *InMemoryStore) newSubjectID() SubjectID {
	id, err := uuid.NewV7()
	if err != nil {
		return "-1"
	}
	return SubjectID(id.String())
}
