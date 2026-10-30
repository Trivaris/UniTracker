package api

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
)

type InMemoryStore struct {
	tasks     map[TaskID]*Task
	subjects  map[SubjectID]*Subject
	series    map[AssignmentSeriesID]*AssignmentSeries
	exercises map[ExerciseID]*Exercise
}

func NewMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tasks:     map[TaskID]*Task{},
		subjects:  map[SubjectID]*Subject{},
		series:    map[AssignmentSeriesID]*AssignmentSeries{},
		exercises: map[ExerciseID]*Exercise{},
	}
}

func (s *InMemoryStore) StoreAssignmentSeries(ctx context.Context, series AssignmentSeries) (*AssignmentSeries, error) {
	series.Normalize()
	series.ID = AssignmentSeriesID(s.newID())
	s.series[series.ID] = &series
	return &series, nil
}

func (s *InMemoryStore) GetAssignmentSeries(ctx context.Context, id AssignmentSeriesID) (*AssignmentSeries, error) {
	series, ok := s.series[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return series, nil
}

func (s *InMemoryStore) GetAssignmentOccurrences(ctx context.Context, subjectID SubjectID) ([]AssignmentOccurrence, error) {
	var result []AssignmentOccurrence
	for _, series := range s.series {
		if subjectID != "" && series.SubjectID != subjectID {
			continue
		}
		for number, release := range series.OccurrenceDates(time.Now()) {
			var task *Task
			for _, candidate := range s.tasks {
				if candidate.SeriesID == series.ID && candidate.OccurrenceNumber == number+1 {
					task = candidate
					break
				}
			}
			if task == nil {
				deadline := occurrenceDeadline(release, series.DeadlineWeekday, series.DeadlineTime)
				created, err := s.StoreTask(ctx, Task{Name: occurrenceName(series.Name, number+1), SubjectID: series.SubjectID, SeriesID: series.ID, OccurrenceNumber: number + 1, StartDate: &release, Deadline: &deadline, State: TaskStateNotStarted, SheetURL: series.SheetURL, SolutionURL: series.SolutionURL})
				if err != nil {
					return nil, err
				}
				task = created
				for _, template := range series.Exercises {
					copy := template
					copy.ID = ExerciseID(s.newID())
					copy.OccurrenceID = task.ID
					copy.State = TaskStateNotStarted
					s.exercises[copy.ID] = &copy
				}
			}
			var exercises []Exercise
			for _, exercise := range s.exercises {
				if exercise.OccurrenceID == task.ID {
					exercises = append(exercises, *exercise)
				}
			}
			subject, _ := s.GetSubject(ctx, task.SubjectID)
			result = append(result, AssignmentOccurrence{TaskView: TaskView{Task: task, SubjectName: subject.Name}, SeriesID: series.ID, OccurrenceNumber: number + 1, Kind: series.Kind, Requirement: series.Requirement, Exercises: exercises})
		}
	}
	return result, nil
}

func (s *InMemoryStore) UpdateExercisePoints(ctx context.Context, id ExerciseID, achieved *float64, possible *float64) error {
	exercise, ok := s.exercises[id]
	if !ok {
		return ErrTaskNotFound
	}
	if achieved != nil {
		exercise.AchievedPoints = achieved
	}
	if possible != nil {
		exercise.PossiblePoints = possible
	}
	return nil
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

func (s *InMemoryStore) newID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return "-1"
	}
	return id.String()
}

func (s *InMemoryStore) newSubjectID() SubjectID {
	id, err := uuid.NewV7()
	if err != nil {
		return "-1"
	}
	return SubjectID(id.String())
}
