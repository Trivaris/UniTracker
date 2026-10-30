package api

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func (s *SQLiteStore) StoreAssignmentSeries(ctx context.Context, series AssignmentSeries) (*AssignmentSeries, error) {
	series.Normalize()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO assignment_series (
			name, subject_id, kind, requirement, start_date, recurrence_interval,
			recurrence_until, release_time, deadline_weekday, deadline_time, sheet_url, solution_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, series.Name, series.SubjectID, series.Kind, series.Requirement, datetimeString(series.StartDate),
		series.RecurrenceInterval, datetimeString(series.RecurrenceUntil), series.ReleaseTime,
		series.DeadlineWeekday, series.DeadlineTime, series.SheetURL, series.SolutionURL)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	series.ID = AssignmentSeriesID(strconv.FormatInt(id, 10))
	for number, exercise := range series.Exercises {
		_, err = s.db.ExecContext(ctx, `INSERT INTO exercise_template (series_id, number, name, link_url, possible_points) VALUES (?, ?, ?, ?, ?)`,
			series.ID, number+1, exercise.Name, exercise.LinkURL, floatValue(exercise.PossiblePoints))
		if err != nil {
			return nil, err
		}
	}
	return &series, nil
}

func (s *SQLiteStore) GetAssignmentSeries(ctx context.Context, id AssignmentSeriesID) (*AssignmentSeries, error) {
	series, err := s.scanAssignmentSeries(ctx, `SELECT id, name, subject_id, kind, requirement, start_date, recurrence_interval, recurrence_until, release_time, deadline_weekday, deadline_time, sheet_url, solution_url FROM assignment_series WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	series.Exercises, err = s.getExerciseTemplates(ctx, id)
	return series, err
}

func (s *SQLiteStore) GetAssignmentOccurrences(ctx context.Context, subjectID SubjectID) ([]AssignmentOccurrence, error) {
	query := `SELECT id, name, subject_id, kind, requirement, start_date, recurrence_interval, recurrence_until, release_time, deadline_weekday, deadline_time, sheet_url, solution_url FROM assignment_series`
	args := []any{}
	if subjectID != "" {
		query += " WHERE subject_id = ?"
		args = append(args, subjectID)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	var seriesList []*AssignmentSeries
	for rows.Next() {
		series, err := scanSeriesRow(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		seriesList = append(seriesList, series)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}

	var occurrences []AssignmentOccurrence
	for _, series := range seriesList {
		series.Normalize()
		for number, release := range series.OccurrenceDates(time.Now()) {
			if err := s.ensureOccurrence(ctx, series, number+1, release); err != nil {
				return nil, err
			}
			task, err := s.GetTask(ctx, TaskID(fmt.Sprint(s.occurrenceID(ctx, series.ID, number+1))))
			if err != nil {
				return nil, err
			}
			subject, err := s.GetSubject(ctx, task.SubjectID)
			if err != nil {
				return nil, err
			}
			exercises, err := s.getExercises(ctx, task.ID)
			if err != nil {
				return nil, err
			}
			occurrences = append(occurrences, AssignmentOccurrence{
				TaskView: TaskView{Task: task, SubjectName: subject.Name},
				SeriesID: series.ID, OccurrenceNumber: number + 1,
				Kind: series.Kind, Requirement: series.Requirement, Exercises: exercises,
			})
		}
	}
	return occurrences, nil
}

func (s *SQLiteStore) UpdateExercisePoints(ctx context.Context, id ExerciseID, achieved *float64, possible *float64) error {
	parts, args := []string{}, []any{}
	if achieved != nil {
		parts = append(parts, "achieved_points = ?")
		args = append(args, *achieved)
	}
	if possible != nil {
		parts = append(parts, "possible_points = ?")
		args = append(args, *possible)
	}
	if len(parts) == 0 {
		return nil
	}
	args = append(args, id)
	_, err := s.db.ExecContext(ctx, "UPDATE exercise SET "+strings.Join(parts, ", ")+" WHERE id = ?", args...)
	return err
}

func NewSQLStore(ctx context.Context, connStr string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err = db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}

	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	if err = initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) StoreTask(ctx context.Context, task Task) (*Task, error) {
	task.Normalize()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO task (
			name, subject_id, series_id, occurrence_number, start_date, deadline, is_recurring, recurrence_interval,
			recurrence_until, state, sheet_url, solution_url, possible_points, achieved_points
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		task.Name,
		task.SubjectID,
		nullableInt(task.SeriesID),
		nullableInt(task.OccurrenceNumber),

		datetimeString(task.StartDate),
		datetimeString(task.Deadline),
		boolToInt(task.IsRecurring),
		task.RecurrenceInterval,
		datetimeString(task.RecurrenceUntil),
		string(task.State),
		task.SheetURL,
		task.SolutionURL,
		floatValue(task.PossiblePoints),
		floatValue(task.AchievedPoints),
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	task.ID = TaskID(strconv.FormatInt(id, 10))
	return &task, nil
}

func (s *SQLiteStore) StoreSubject(ctx context.Context, name string) (*Subject, error) {
	result, err := s.db.ExecContext(ctx, "INSERT INTO subject (name) VALUES (?)", name)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Subject{ID: SubjectID(strconv.FormatInt(id, 10)), Name: name}, nil
}

func (s *SQLiteStore) GetTask(ctx context.Context, id TaskID) (*Task, error) {
	var (
		taskID          int64
		name            string
		subjectID       string
		seriesID        sql.NullString
		occurrenceNo    sql.NullInt64
		startDate       sql.NullString
		deadline        sql.NullString
		isRecurring     int
		recurrenceInt   sql.NullString
		recurrenceUntil sql.NullString
		state           sql.NullString
		sheetURL        sql.NullString
		solutionURL     sql.NullString
		possible        sql.NullFloat64
		achieved        sql.NullFloat64
	)

	err := s.db.QueryRowContext(ctx, `SELECT id, name, subject_id, series_id, occurrence_number, start_date, deadline, is_recurring, recurrence_interval, recurrence_until, state, sheet_url, solution_url, possible_points, achieved_points FROM task WHERE id = ?`, id).Scan(
		&taskID, &name, &subjectID, &seriesID, &occurrenceNo, &startDate, &deadline, &isRecurring, &recurrenceInt, &recurrenceUntil, &state, &sheetURL, &solutionURL, &possible, &achieved,
	)
	if err != nil {
		return nil, err
	}

	task := &Task{
		ID:                 TaskID(strconv.FormatInt(taskID, 10)),
		Name:               name,
		SubjectID:          SubjectID(subjectID),
		SeriesID:           AssignmentSeriesID(seriesID.String),
		OccurrenceNumber:   int(occurrenceNo.Int64),
		IsRecurring:        isRecurring == 1,
		RecurrenceInterval: recurrenceInt.String,
		State:              TaskState(state.String),
		SheetURL:           sheetURL.String,
		SolutionURL:        solutionURL.String,
	}
	task.StartDate, err = parseNullableDate(startDate)
	if err != nil {
		return nil, fmt.Errorf("parse start date: %w", err)
	}
	task.Deadline, err = parseNullableDate(deadline)
	if err != nil {
		return nil, fmt.Errorf("parse deadline: %w", err)
	}
	task.RecurrenceUntil, err = parseNullableDate(recurrenceUntil)
	if err != nil {
		return nil, fmt.Errorf("parse recurrence until: %w", err)
	}
	if possible.Valid {
		value := possible.Float64
		task.PossiblePoints = &value
	}
	if achieved.Valid {
		value := achieved.Float64
		task.AchievedPoints = &value
	}
	task.Normalize()
	return task, nil
}

func (s *SQLiteStore) GetSubject(ctx context.Context, id SubjectID) (*Subject, error) {
	var (
		subjectID int64
		name      string
	)
	err := s.db.QueryRowContext(ctx, "SELECT id, name FROM subject WHERE id = ?", id).Scan(&subjectID, &name)
	if err != nil {
		return nil, err
	}
	return &Subject{ID: SubjectID(strconv.FormatInt(subjectID, 10)), Name: name}, nil
}

func (s *SQLiteStore) GetAllSubjects(ctx context.Context) ([]*Subject, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name FROM subject ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []*Subject
	for rows.Next() {
		var subjectID int64
		var name string
		if err = rows.Scan(&subjectID, &name); err != nil {
			return nil, err
		}
		subjects = append(subjects, &Subject{ID: SubjectID(strconv.FormatInt(subjectID, 10)), Name: name})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return subjects, nil
}

func (s *SQLiteStore) GetAllTasks(ctx context.Context) ([]*Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, subject_id, series_id, occurrence_number, start_date, deadline, is_recurring, recurrence_interval, recurrence_until, state, sheet_url, solution_url, possible_points, achieved_points FROM task ORDER BY deadline IS NULL, deadline ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var (
			taskID          int64
			name            string
			subjectID       string
			seriesID        sql.NullString
			occurrenceNo    sql.NullInt64
			startDate       sql.NullString
			deadline        sql.NullString
			isRecurring     int
			recurrenceInt   sql.NullString
			recurrenceUntil sql.NullString
			state           sql.NullString
			sheetURL        sql.NullString
			solutionURL     sql.NullString
			possible        sql.NullFloat64
			achieved        sql.NullFloat64
		)
		if err = rows.Scan(&taskID, &name, &subjectID, &seriesID, &occurrenceNo, &startDate, &deadline, &isRecurring, &recurrenceInt, &recurrenceUntil, &state, &sheetURL, &solutionURL, &possible, &achieved); err != nil {
			return nil, err
		}

		task := &Task{
			ID:                 TaskID(strconv.FormatInt(taskID, 10)),
			Name:               name,
			SubjectID:          SubjectID(subjectID),
			SeriesID:           AssignmentSeriesID(seriesID.String),
			OccurrenceNumber:   int(occurrenceNo.Int64),
			IsRecurring:        isRecurring == 1,
			RecurrenceInterval: recurrenceInt.String,
			State:              TaskState(state.String),
			SheetURL:           sheetURL.String,
			SolutionURL:        solutionURL.String,
		}
		task.StartDate, err = parseNullableDate(startDate)
		if err != nil {
			return nil, fmt.Errorf("parse start date: %w", err)
		}
		task.Deadline, err = parseNullableDate(deadline)
		if err != nil {
			return nil, fmt.Errorf("parse deadline: %w", err)
		}
		task.RecurrenceUntil, err = parseNullableDate(recurrenceUntil)
		if err != nil {
			return nil, fmt.Errorf("parse recurrence until: %w", err)
		}
		if possible.Valid {
			value := possible.Float64
			task.PossiblePoints = &value
		}
		if achieved.Valid {
			value := achieved.Float64
			task.AchievedPoints = &value
		}
		task.Normalize()
		tasks = append(tasks, task)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *SQLiteStore) DeleteSubject(ctx context.Context, id SubjectID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM task WHERE subject_id = ?", id)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM subject WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) DeleteTask(ctx context.Context, id TaskID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM task WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) UpdateTaskState(ctx context.Context, id TaskID, state TaskState) error {
	taskState := NormalizeTaskState(string(state))
	_, err := s.db.ExecContext(ctx, "UPDATE task SET state = ? WHERE id = ?", string(taskState), id)
	return err
}

func (s *SQLiteStore) UpdateTaskPoints(ctx context.Context, id TaskID, achieved *float64, possible *float64) error {
	if achieved == nil && possible == nil {
		return nil
	}

	parts := []string{}
	args := []any{}
	if achieved != nil {
		parts = append(parts, "achieved_points = ?")
		args = append(args, *achieved)
	}
	if possible != nil {
		parts = append(parts, "possible_points = ?")
		args = append(args, *possible)
	}
	args = append(args, id)

	query := "UPDATE task SET " + strings.Join(parts, ", ") + " WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS subject (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS task (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		subject_id INTEGER NOT NULL,
		series_id INTEGER,
		occurrence_number INTEGER,
		start_date TEXT,
		deadline TEXT,
		is_recurring INTEGER NOT NULL DEFAULT 0,
		recurrence_interval TEXT,
		recurrence_until TEXT,
		state TEXT NOT NULL DEFAULT 'not_started',
		sheet_url TEXT,
		solution_url TEXT,
		possible_points REAL,
		achieved_points REAL,
		FOREIGN KEY(subject_id) REFERENCES subject(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS assignment_series (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		subject_id INTEGER NOT NULL,
		kind TEXT NOT NULL DEFAULT 'sheet',
		requirement TEXT NOT NULL DEFAULT 'required',
		start_date TEXT,
		recurrence_interval TEXT NOT NULL DEFAULT 'weekly',
		recurrence_until TEXT,
		release_time TEXT NOT NULL DEFAULT '08:00',
		deadline_weekday INTEGER NOT NULL DEFAULT 5,
		deadline_time TEXT NOT NULL DEFAULT '18:00',
		sheet_url TEXT,
		solution_url TEXT,
		FOREIGN KEY(subject_id) REFERENCES subject(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS exercise_template (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		series_id INTEGER NOT NULL,
		number INTEGER NOT NULL,
		name TEXT NOT NULL,
		link_url TEXT,
		possible_points REAL,
		FOREIGN KEY(series_id) REFERENCES assignment_series(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS exercise (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		occurrence_id INTEGER NOT NULL,
		number INTEGER NOT NULL,
		name TEXT NOT NULL,
		link_url TEXT,
		possible_points REAL,
		achieved_points REAL,
		state TEXT NOT NULL DEFAULT 'not_started',
		FOREIGN KEY(occurrence_id) REFERENCES task(id) ON DELETE CASCADE
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	if err := addColumnIfMissing(db, "task", "subject_id", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "start_date", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "deadline", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "is_recurring", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "recurrence_interval", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "recurrence_until", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "state", "TEXT NOT NULL DEFAULT 'not_started'"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "sheet_url", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "solution_url", "TEXT"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "possible_points", "REAL"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "achieved_points", "REAL"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "series_id", "INTEGER"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "task", "occurrence_number", "INTEGER"); err != nil {
		return err
	}

	return nil
}

func addColumnIfMissing(db *sql.DB, tableName, columnName, columnType string) error {
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE " + tableName + " ADD COLUMN " + columnName + " " + columnType)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func floatValue(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt(value any) any {
	switch typed := value.(type) {
	case AssignmentSeriesID:
		if typed == "" {
			return nil
		}
		return typed
	case int:
		if typed == 0 {
			return nil
		}
		return typed
	default:
		return nil
	}
}

func datetimeString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.DateOnly)
}

func parseNullableDate(value sql.NullString) (*time.Time, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.DateOnly, value.String)
	if err == nil {
		return &parsed, nil
	}
	parsed, err = time.Parse(time.RFC3339, value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (s *SQLiteStore) Close() {
	if err := s.db.Close(); err != nil {
		fmt.Println(err)
	}
}

func (s *SQLiteStore) scanAssignmentSeries(ctx context.Context, query string, args ...any) (*AssignmentSeries, error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	return scanSeriesRow(row)
}

type scanner interface{ Scan(dest ...any) error }

func scanSeriesRow(row scanner) (*AssignmentSeries, error) {
	var (
		id, name, subjectID, kind, requirement, startDate, interval, until, releaseTime, deadlineTime, sheetURL, solutionURL sql.NullString
		weekday                                                                                                              sql.NullInt64
	)
	if err := row.Scan(&id, &name, &subjectID, &kind, &requirement, &startDate, &interval, &until, &releaseTime, &weekday, &deadlineTime, &sheetURL, &solutionURL); err != nil {
		return nil, err
	}
	series := &AssignmentSeries{ID: AssignmentSeriesID(id.String), Name: name.String, SubjectID: SubjectID(subjectID.String), Kind: AssignmentKind(kind.String), Requirement: AssignmentRequirement(requirement.String), RecurrenceInterval: interval.String, ReleaseTime: releaseTime.String, DeadlineWeekday: time.Weekday(weekday.Int64), DeadlineTime: deadlineTime.String, SheetURL: sheetURL.String, SolutionURL: solutionURL.String}
	var err error
	series.StartDate, err = parseNullableDate(startDate)
	if err != nil {
		return nil, err
	}
	series.RecurrenceUntil, err = parseNullableDate(until)
	if err != nil {
		return nil, err
	}
	series.Normalize()
	return series, nil
}

func (s *SQLiteStore) getExerciseTemplates(ctx context.Context, seriesID AssignmentSeriesID) ([]Exercise, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT number, name, link_url, possible_points FROM exercise_template WHERE series_id = ? ORDER BY number", seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var exercises []Exercise
	for rows.Next() {
		var exercise Exercise
		var link sql.NullString
		var possible sql.NullFloat64
		if err := rows.Scan(&exercise.Number, &exercise.Name, &link, &possible); err != nil {
			return nil, err
		}
		exercise.LinkURL = link.String
		if possible.Valid {
			value := possible.Float64
			exercise.PossiblePoints = &value
		}
		exercises = append(exercises, exercise)
	}
	return exercises, rows.Err()
}

func (s *SQLiteStore) ensureOccurrence(ctx context.Context, series *AssignmentSeries, number int, release time.Time) error {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task WHERE series_id = ? AND occurrence_number = ?", series.ID, number).Scan(&exists)
	if err != nil || exists > 0 {
		return err
	}
	deadline := occurrenceDeadline(release, series.DeadlineWeekday, series.DeadlineTime)
	result, err := s.db.ExecContext(ctx, `INSERT INTO task (name, subject_id, series_id, occurrence_number, start_date, deadline, state, sheet_url, solution_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		occurrenceName(series.Name, number), series.SubjectID, series.ID, number, release.Format(time.RFC3339), deadline.Format(time.RFC3339), TaskStateNotStarted, series.SheetURL, series.SolutionURL)
	if err != nil {
		return err
	}
	occurrenceID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	templates, err := s.getExerciseTemplates(ctx, series.ID)
	if err != nil {
		return err
	}
	for _, exercise := range templates {
		_, err = s.db.ExecContext(ctx, `INSERT INTO exercise (occurrence_id, number, name, link_url, possible_points, state) VALUES (?, ?, ?, ?, ?, ?)`, occurrenceID, exercise.Number, exercise.Name, exercise.LinkURL, floatValue(exercise.PossiblePoints), TaskStateNotStarted)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) occurrenceID(ctx context.Context, seriesID AssignmentSeriesID, number int) int64 {
	var id int64
	if err := s.db.QueryRowContext(ctx, "SELECT id FROM task WHERE series_id = ? AND occurrence_number = ?", seriesID, number).Scan(&id); err != nil {
		return 0
	}
	return id
}

func (s *SQLiteStore) getExercises(ctx context.Context, occurrenceID TaskID) ([]Exercise, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, number, name, link_url, possible_points, achieved_points, state FROM exercise WHERE occurrence_id = ? ORDER BY number", occurrenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var exercises []Exercise
	for rows.Next() {
		var exercise Exercise
		var id, link, state sql.NullString
		var possible, achieved sql.NullFloat64
		if err := rows.Scan(&id, &exercise.Number, &exercise.Name, &link, &possible, &achieved, &state); err != nil {
			return nil, err
		}
		exercise.ID, exercise.OccurrenceID, exercise.LinkURL, exercise.State = ExerciseID(id.String), occurrenceID, link.String, NormalizeTaskState(state.String)
		if possible.Valid {
			value := possible.Float64
			exercise.PossiblePoints = &value
		}
		if achieved.Valid {
			value := achieved.Float64
			exercise.AchievedPoints = &value
		}
		exercises = append(exercises, exercise)
	}
	return exercises, rows.Err()
}
