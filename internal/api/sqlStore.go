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
			name, subject_id, start_date, deadline, is_recurring, recurrence_interval,
			recurrence_until, state, sheet_url, solution_url, possible_points, achieved_points
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		task.Name,
		task.SubjectID,
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

	err := s.db.QueryRowContext(ctx, `SELECT id, name, subject_id, start_date, deadline, is_recurring, recurrence_interval, recurrence_until, state, sheet_url, solution_url, possible_points, achieved_points FROM task WHERE id = ?`, id).Scan(
		&taskID, &name, &subjectID, &startDate, &deadline, &isRecurring, &recurrenceInt, &recurrenceUntil, &state, &sheetURL, &solutionURL, &possible, &achieved,
	)
	if err != nil {
		return nil, err
	}

	task := &Task{
		ID:                 TaskID(strconv.FormatInt(taskID, 10)),
		Name:               name,
		SubjectID:          SubjectID(subjectID),
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
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, subject_id, start_date, deadline, is_recurring, recurrence_interval, recurrence_until, state, sheet_url, solution_url, possible_points, achieved_points FROM task ORDER BY deadline IS NULL, deadline ASC, id ASC`)
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
		if err = rows.Scan(&taskID, &name, &subjectID, &startDate, &deadline, &isRecurring, &recurrenceInt, &recurrenceUntil, &state, &sheetURL, &solutionURL, &possible, &achieved); err != nil {
			return nil, err
		}

		task := &Task{
			ID:                 TaskID(strconv.FormatInt(taskID, 10)),
			Name:               name,
			SubjectID:          SubjectID(subjectID),
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
