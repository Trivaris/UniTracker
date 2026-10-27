package api

import (
	"maps"
	"slices"

	"github.com/google/uuid"
)

type Store struct {
	tasks map[TaskID]*Task
	subjects map[SubjectID]*Subject
}

func NewStore() *Store {
	return &Store{
		tasks: map[TaskID]*Task{},
		subjects: map[SubjectID]*Subject{},
	}
}

func (s *Store) StoreTask(name string, subjectid SubjectID) (*Task, error) {
	id := s.newTaskID()
	task := &Task{
		ID: id,
		Name: name,
		SubjectId: subjectid,
	}
	s.tasks[id] = task
	return task, nil
}

func (s *Store) StoreSubject(name string) (*Subject, error) {
	id := s.newSubjectID()
	subject := &Subject{
		ID: id,
		Name: name,
	}
	s.subjects[id] = subject
	return subject, nil
}

func (s *Store) GetTask(id TaskID) *Task {
	task, ok := s.tasks[id]
	if !ok {
		return nil
	}

	return task
}

func (s *Store) GetSubject(id SubjectID) *Subject {
	subject, ok := s.subjects[id]
	if !ok {
		return nil
	}

	return subject
}

func (s *Store) GetAllSubjects() []*Subject {
	return slices.Collect(maps.Values(s.subjects))
}

func (s *Store) GetAllTasks() []*Task {
	return slices.Collect(maps.Values(s.tasks))
}

func (s *Store) newTaskID() TaskID {
	id, err := uuid.NewV7()
	if err != nil {
		return "-1"
	}
	return TaskID(id.String())
}
func (s *Store) newSubjectID() SubjectID {
	id, err := uuid.NewV7()
	if err != nil {
		return "-1"
	}
	return SubjectID(id.String())
}
