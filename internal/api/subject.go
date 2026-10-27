package api

type SubjectID string

type Subject struct {
	ID SubjectID `json:"id"`
	Name string `json:"name"`
}