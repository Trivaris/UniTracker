package api

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type MySQLStore struct {
	db *sql.DB
}

func NewSQLStore(ctx context.Context, connStr string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5 * time.Second)
	defer cancel()

	err = db.PingContext(pingCtx)
	if err != nil {
		return nil, err
	}

	// Recommended Settings
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)

	return &MySQLStore{
		db: db,
	}, nil
}

func (s *MySQLStore) StoreTask(ctx context.Context, name string, subjectid SubjectID) (*Task, error) {
	result, err := s.db.ExecContext(ctx, "INSERT INTO task VALUES (?, ?)", name, subjectid)
	if err != nil {
		return nil, err
	}
	
	taskid, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Task{
		ID: TaskID(taskid),
		Name: name,
		SubjectID: subjectid,
	}, nil
}

func (s *MySQLStore) StoreSubject(ctx context.Context, name string) (*Subject, error) {
	result, err := s.db.ExecContext(ctx, "INSERT INTO subject VALUES (?)", name)
	if err != nil {
		return nil, err
	}
	
	subjectid, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Subject{
		ID: SubjectID(subjectid),
		Name: name,
	}, nil
}

func (s *MySQLStore) GetTask(ctx context.Context, id TaskID) (*Task, error) {
	var name *string
	var subjectid *SubjectID
	err := s.db.QueryRowContext(ctx, "SELECT name, subject FROM task WHERE id=?", id).Scan(&name, &subjectid)
	
	if err != nil {
		return nil, err
	}
	
	return &Task{
		Name: *name,
		SubjectID: *subjectid,
	}, nil
}

func (s *MySQLStore) GetSubject(ctx context.Context, id SubjectID) (*Subject, error) {
	var name *string
	err := s.db.QueryRowContext(ctx, "SELECT name FROM subject WHERE id=?", id).Scan(&name)
	
	if err != nil {
		return nil, err
	}
	
	return &Subject{
		Name: *name,
	}, nil
}

func (s *MySQLStore) GetAllSubjects(ctx context.Context) ([]*Subject, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name FROM subject")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []*Subject
	for rows.Next() {
		var id SubjectID
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			return nil, err
		}
		subjects = append(subjects, &Subject{
			ID: id,
			Name: name,
		})
	}

	return subjects, nil
}

func (s *MySQLStore) GetAllTasks(ctx context.Context) ([]*Task, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, subject FROM task")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var id TaskID
		var name string
		var subjectid SubjectID
		err := rows.Scan(&id, &name, &subjectid)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, &Task{
			ID: id,
			Name: name,
			SubjectID: subjectid,
		})
	}

	return tasks, nil
}

func (s *MySQLStore) Close() {
	err := s.db.Close()
	if err != nil {
		fmt.Println(err)
	}
}