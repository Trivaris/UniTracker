package api

import (
	"fmt"
	"time"
)

type AssignmentSeriesID string
type ExerciseID string

type AssignmentKind string

const (
	AssignmentKindSheet AssignmentKind = "sheet"
	AssignmentKindQuiz  AssignmentKind = "quiz"
)

type AssignmentRequirement string

const (
	AssignmentRequired AssignmentRequirement = "required"
	AssignmentOptional AssignmentRequirement = "optional"
)

type AssignmentSeries struct {
	ID                 AssignmentSeriesID
	Name               string
	SubjectID          SubjectID
	Kind               AssignmentKind
	Requirement        AssignmentRequirement
	StartDate          *time.Time
	RecurrenceInterval string
	RecurrenceUntil    *time.Time
	ReleaseTime        string
	DeadlineWeekday    time.Weekday
	DeadlineTime       string
	SheetURL           string
	SolutionURL        string
	Exercises          []Exercise
}

type Exercise struct {
	ID             ExerciseID
	OccurrenceID   TaskID
	Number         int
	Name           string
	LinkURL        string
	PossiblePoints *float64
	AchievedPoints *float64
	State          TaskState
}

type AssignmentOccurrence struct {
	TaskView
	SeriesID         AssignmentSeriesID
	OccurrenceNumber int
	Kind             AssignmentKind
	Requirement      AssignmentRequirement
	Exercises        []Exercise
}

func (s *AssignmentSeries) Normalize() {
	if s == nil {
		return
	}
	if s.Kind != AssignmentKindQuiz {
		s.Kind = AssignmentKindSheet
	}
	if s.Requirement != AssignmentOptional {
		s.Requirement = AssignmentRequired
	}
	if s.RecurrenceInterval != "biweekly" {
		s.RecurrenceInterval = "weekly"
	}
	if s.ReleaseTime == "" {
		s.ReleaseTime = "08:00"
	}
	if s.DeadlineTime == "" {
		s.DeadlineTime = "18:00"
	}
}

func (s *AssignmentSeries) OccurrenceDates(now time.Time) []time.Time {
	if s == nil || s.StartDate == nil {
		return nil
	}
	s.Normalize()
	start := dateOnly(*s.StartDate)
	until := now
	if s.RecurrenceUntil != nil && dateOnly(*s.RecurrenceUntil).Before(until) {
		until = dateOnly(*s.RecurrenceUntil)
	}
	step := 7
	if s.RecurrenceInterval == "biweekly" {
		step = 14
	}
	var dates []time.Time
	for occurrence := 0; ; occurrence++ {
		releaseDate := start.AddDate(0, 0, occurrence*step)
		if releaseDate.After(dateOnly(until)) {
			break
		}
		if s.RecurrenceUntil != nil && releaseDate.After(dateOnly(*s.RecurrenceUntil)) {
			break
		}
		release := releaseDate
		if parsed, err := time.Parse("15:04", s.ReleaseTime); err == nil {
			release = time.Date(releaseDate.Year(), releaseDate.Month(), releaseDate.Day(), parsed.Hour(), parsed.Minute(), 0, 0, releaseDate.Location())
		}
		dates = append(dates, release)
	}
	return dates
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func occurrenceDeadline(release time.Time, weekday time.Weekday, clock string) time.Time {
	offset := (int(weekday) - int(release.Weekday()) + 7) % 7
	deadline := release.AddDate(0, 0, offset)
	hour, minute := 18, 0
	if parsed, err := time.Parse("15:04", clock); err == nil {
		hour, minute = parsed.Hour(), parsed.Minute()
	}
	return time.Date(deadline.Year(), deadline.Month(), deadline.Day(), hour, minute, 0, 0, release.Location())
}

func occurrenceName(seriesName string, number int) string {
	return fmt.Sprintf("%s %d", seriesName, number)
}
