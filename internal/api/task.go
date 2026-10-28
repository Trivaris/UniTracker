package api

type TaskID string

type Task struct {
	ID TaskID
	Name string
	SubjectID SubjectID
}

//func (t Task) MarshalJSON() ([]byte, error) {
//	var subjectID SubjectID
//	if t.Subject != nil {
//		subjectID = t.Subject.ID
//	}
//	
//	return json.Marshal(struct {
//		ID TaskID `json:"id"`
//		Name string `json:"name"`
//		SubjectID SubjectID `json:"subject"`
//	}{
//		ID: t.ID,
//		Name: t.Name,
//		SubjectID: subjectID,
//	})
//}