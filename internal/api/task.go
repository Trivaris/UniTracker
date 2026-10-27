package api

type TaskID string

type Task struct {
	ID TaskID `json:"id"`
	Name string `json:"name"`
	SubjectId SubjectID `json:"subject"`
}
