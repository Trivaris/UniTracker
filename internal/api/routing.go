package api

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*.html templates/partials/*.html
var templateFS embed.FS

type SubjectSummary struct {
	SubjectName string
	Achieved    float64
	Possible    float64
	Percent     float64
}

type DashboardData struct {
	Subjects          []*Subject
	SelectedSubjectID SubjectID
	SelectedStatus    string
	Tasks             []TaskView
	Occurrences       []AssignmentOccurrence
	Summary           SubjectSummary
}

type Routing struct {
	store  Store
	engine *gin.Engine
}

func NewRouting(store Store) *Routing {
	engine := gin.Default()
	engine.SetHTMLTemplate(loadTemplates())
	return &Routing{store: store, engine: engine}
}

func loadTemplates() *template.Template {
	funcMap := template.FuncMap{
		"add": func(left, right int) int {
			return left + right
		},
		"formatDate": func(value *time.Time) string {
			if value == nil {
				return "—"
			}
			return value.Format(time.DateOnly)
		},
		"stateLabel": func(value TaskState) string {
			switch value {
			case TaskStateInProgress:
				return "In Progress"
			case TaskStateFinished:
				return "Finished"
			default:
				return "Not Started"
			}
		},
		"stateClass": func(value TaskState) string {
			switch value {
			case TaskStateInProgress:
				return "bg-amber-100 text-amber-800"
			case TaskStateFinished:
				return "bg-emerald-100 text-emerald-800"
			default:
				return "bg-slate-200 text-slate-700"
			}
		},
		"formatPercent": func(achieved, possible float64) string {
			if possible <= 0 {
				return "0%"
			}
			return fmt.Sprintf("%.0f%%", (achieved/possible)*100)
		},
		"pointsPercent": func(task *Task) float64 {
			if task == nil || task.PossiblePoints == nil || *task.PossiblePoints <= 0 {
				return 0
			}
			if task.AchievedPoints == nil {
				return 0
			}
			return (*task.AchievedPoints / *task.PossiblePoints) * 100
		},
		"isOverdue": func(value *time.Time) bool {
			if value == nil {
				return false
			}
			return value.Before(time.Now())
		},
	}

	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFS(templateFS,
		"templates/*.html",
		"templates/partials/*.html",
	)
	if err != nil {
		panic(err)
	}
	return tmpl
}

func (r *Routing) Route() error {
	r.engine.GET("/", r.handleDashboard)
	r.engine.GET("/subject", r.handleListSubjects)
	r.engine.GET("/task", r.handleListTasks)
	r.engine.GET("/subject/:id", r.handleGetSubject)
	r.engine.GET("/task/:id", r.handleGetTask)
	r.engine.POST("/subject", r.handleCreateSubject)
	r.engine.POST("/task", r.handleCreateTask)
	r.engine.POST("/ui/task", r.handleCreateTaskUI)
	r.engine.POST("/ui/series", r.handleCreateSeriesUI)
	r.engine.PATCH("/ui/task/:id/state", r.handleUpdateTaskStateUI)
	r.engine.PATCH("/ui/task/:id/points", r.handleUpdateTaskPointsUI)
	r.engine.PATCH("/ui/exercise/:id/points", r.handleUpdateExercisePointsUI)
	r.engine.DELETE("/ui/task/:id", r.handleDeleteTaskUI)

	r.engine.GET("/api/subjects", r.handleListSubjects)
	r.engine.GET("/api/tasks", r.handleListTasks)
	r.engine.GET("/api/subject/:id", r.handleGetSubject)
	r.engine.GET("/api/task/:id", r.handleGetTask)
	r.engine.POST("/api/subject", r.handleCreateSubject)
	r.engine.POST("/api/task", r.handleCreateTask)

	return r.engine.Run()
}

func (r *Routing) handleDashboard(ctx *gin.Context) {
	subjectID := SubjectID(strings.TrimSpace(ctx.Query("subject_id")))
	selectedStatus := normalizeStatusFilter(ctx.Query("status"))
	subjects, err := r.store.GetAllSubjects(ctx.Request.Context())
	if err != nil && !errors.Is(err, ErrSubjectNotFound) {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	occurrences, err := r.store.GetAssignmentOccurrences(ctx.Request.Context(), subjectID)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	tasks, err := r.store.GetAllTasks(ctx.Request.Context())
	if err != nil && !errors.Is(err, ErrTaskNotFound) {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	filteredTasks := make([]TaskView, 0, len(tasks))
	filteredSummaryTasks := make([]*Task, 0, len(tasks))
	summary := SubjectSummary{SubjectName: "All subjects"}
	for _, task := range tasks {
		if task.SeriesID != "" {
			continue
		}
		if subjectID != "" && task.SubjectID != subjectID {
			continue
		}
		if selectedStatus != "" && string(task.State) != selectedStatus {
			continue
		}
		subject, _ := r.store.GetSubject(ctx.Request.Context(), task.SubjectID)
		view := TaskView{Task: task}
		if subject != nil {
			view.SubjectName = subject.Name
		}
		filteredTasks = append(filteredTasks, view)
		filteredSummaryTasks = append(filteredSummaryTasks, task)
	}
	if subjectID != "" {
		summary = computeSummary(subjectID, filteredSummaryTasks)
	} else {
		summary = computeSummary("", filteredSummaryTasks)
	}
	filteredOccurrences := make([]AssignmentOccurrence, 0, len(occurrences))
	for _, occurrence := range occurrences {
		if selectedStatus != "" && string(occurrence.State) != selectedStatus {
			continue
		}
		filteredOccurrences = append(filteredOccurrences, occurrence)
		for _, exercise := range occurrence.Exercises {
			if exercise.PossiblePoints != nil && *exercise.PossiblePoints > 0 {
				summary.Possible += *exercise.PossiblePoints
			}
			if exercise.AchievedPoints != nil {
				summary.Achieved += *exercise.AchievedPoints
			}
		}
	}
	if summary.Possible > 0 {
		summary.Percent = (summary.Achieved / summary.Possible) * 100
	}

	ctx.HTML(http.StatusOK, "layout.html", DashboardData{
		Subjects:          subjects,
		SelectedSubjectID: subjectID,
		SelectedStatus:    selectedStatus,
		Tasks:             filteredTasks,
		Occurrences:       filteredOccurrences,
		Summary:           summary,
	})
}

func normalizeStatusFilter(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case string(TaskStateNotStarted), string(TaskStateInProgress), string(TaskStateFinished):
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return ""
	}
}

func (r *Routing) handleListSubjects(ctx *gin.Context) {
	subjects, err := r.store.GetAllSubjects(ctx.Request.Context())
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	if ctx.GetHeader("HX-Request") == "true" {
		for _, subject := range subjects {
			ctx.HTML(http.StatusOK, "subjectItem", subject)
			return
		}
		ctx.Status(http.StatusOK)
		return
	}
	ctx.IndentedJSON(http.StatusOK, subjects)
}

func (r *Routing) handleListTasks(ctx *gin.Context) {
	tasks, err := r.store.GetAllTasks(ctx.Request.Context())
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	views := make([]TaskView, 0, len(tasks))
	for _, task := range tasks {
		subject, _ := r.store.GetSubject(ctx.Request.Context(), task.SubjectID)
		view := TaskView{Task: task}
		if subject != nil {
			view.SubjectName = subject.Name
		}
		views = append(views, view)
	}
	ctx.IndentedJSON(http.StatusOK, views)
}

func (r *Routing) handleGetSubject(ctx *gin.Context) {
	id := SubjectID(ctx.Param("id"))
	subject, err := r.store.GetSubject(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSubjectNotFound) {
			ctx.Status(http.StatusNotFound)
			return
		}
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.IndentedJSON(http.StatusOK, subject)
}

func (r *Routing) handleGetTask(ctx *gin.Context) {
	id := TaskID(ctx.Param("id"))
	task, err := r.store.GetTask(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			ctx.Status(http.StatusNotFound)
			return
		}
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.IndentedJSON(http.StatusOK, task)
}

func (r *Routing) handleCreateSubject(ctx *gin.Context) {
	name := strings.TrimSpace(ctx.PostForm("name"))
	if name == "" {
		ctx.String(http.StatusBadRequest, "name is required")
		return
	}
	subject, err := r.store.StoreSubject(ctx.Request.Context(), name)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	if ctx.GetHeader("HX-Request") == "true" {
		ctx.HTML(http.StatusOK, "subjectItem", subject)
		return
	}
	if ctx.GetHeader("Accept") == "application/json" || strings.Contains(ctx.GetHeader("Accept"), "application/json") {
		ctx.IndentedJSON(http.StatusCreated, subject)
		return
	}
	ctx.Redirect(http.StatusSeeOther, "/")
}

func (r *Routing) handleCreateTask(ctx *gin.Context) {
	task, err := parseTaskForm(ctx)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	created, err := r.store.StoreTask(ctx.Request.Context(), task)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	if ctx.GetHeader("HX-Request") == "true" || strings.Contains(ctx.GetHeader("Accept"), "text/html") {
		subject, _ := r.store.GetSubject(ctx.Request.Context(), created.SubjectID)
		view := TaskView{Task: created}
		if subject != nil {
			view.SubjectName = subject.Name
		}
		ctx.HTML(http.StatusCreated, "taskRow", view)
		return
	}
	ctx.IndentedJSON(http.StatusCreated, created)
}

func (r *Routing) handleCreateTaskUI(ctx *gin.Context) {
	if ctx.GetHeader("HX-Request") != "true" {
		ctx.Redirect(http.StatusSeeOther, "/")
		return
	}
	task, err := parseTaskForm(ctx)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	created, err := r.store.StoreTask(ctx.Request.Context(), task)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	subject, _ := r.store.GetSubject(ctx.Request.Context(), created.SubjectID)
	view := TaskView{Task: created}
	if subject != nil {
		view.SubjectName = subject.Name
	}
	ctx.HTML(http.StatusCreated, "taskRow", view)
}

func (r *Routing) handleCreateSeriesUI(ctx *gin.Context) {
	series, err := parseAssignmentSeriesForm(ctx)
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	if _, err = r.store.StoreAssignmentSeries(ctx.Request.Context(), series); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	if ctx.GetHeader("HX-Request") == "true" {
		ctx.Header("HX-Refresh", "true")
		ctx.Status(http.StatusCreated)
		return
	}
	ctx.Redirect(http.StatusSeeOther, "/")
}

func (r *Routing) handleUpdateTaskStateUI(ctx *gin.Context) {
	id := TaskID(ctx.Param("id"))
	state := NormalizeTaskState(ctx.PostForm("state"))
	if err := r.store.UpdateTaskState(ctx.Request.Context(), id, state); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	task, err := r.store.GetTask(ctx.Request.Context(), id)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	if task.SeriesID != "" {
		occurrences, err := r.store.GetAssignmentOccurrences(ctx.Request.Context(), task.SubjectID)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		for _, occurrence := range occurrences {
			if occurrence.ID == task.ID {
				ctx.HTML(http.StatusOK, "assignmentOccurrence", occurrence)
				return
			}
		}
	}
	subject, _ := r.store.GetSubject(ctx.Request.Context(), task.SubjectID)
	view := TaskView{Task: task}
	if subject != nil {
		view.SubjectName = subject.Name
	}
	ctx.HTML(http.StatusOK, "taskRow", view)
}

func (r *Routing) handleUpdateTaskPointsUI(ctx *gin.Context) {
	id := TaskID(ctx.Param("id"))
	achieved := parseOptionalFloat(ctx.PostForm("achieved_points"))
	possible := parseOptionalFloat(ctx.PostForm("possible_points"))
	if err := r.store.UpdateTaskPoints(ctx.Request.Context(), id, achieved, possible); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	task, err := r.store.GetTask(ctx.Request.Context(), id)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	subject, _ := r.store.GetSubject(ctx.Request.Context(), task.SubjectID)
	view := TaskView{Task: task}
	if subject != nil {
		view.SubjectName = subject.Name
	}
	ctx.HTML(http.StatusOK, "taskRow", view)
}

func (r *Routing) handleUpdateExercisePointsUI(ctx *gin.Context) {
	id := ExerciseID(ctx.Param("id"))
	if err := r.store.UpdateExercisePoints(ctx.Request.Context(), id, parseOptionalFloat(ctx.PostForm("achieved_points")), parseOptionalFloat(ctx.PostForm("possible_points"))); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Header("HX-Refresh", "true")
	ctx.Status(http.StatusOK)
}

func (r *Routing) handleDeleteTaskUI(ctx *gin.Context) {
	id := TaskID(ctx.Param("id"))
	if err := r.store.DeleteTask(ctx.Request.Context(), id); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusOK)
}

func parseTaskForm(ctx *gin.Context) (Task, error) {
	name := strings.TrimSpace(ctx.PostForm("name"))
	if name == "" {
		return Task{}, errors.New("name is required")
	}
	subjectID := SubjectID(strings.TrimSpace(ctx.PostForm("subject_id")))
	if subjectID == "" {
		subjectID = SubjectID(strings.TrimSpace(ctx.PostForm("subject")))
	}
	if subjectID == "" {
		return Task{}, errors.New("subject is required")
	}

	task := Task{
		Name:               name,
		SubjectID:          subjectID,
		State:              NormalizeTaskState(ctx.PostForm("state")),
		IsRecurring:        ctx.PostForm("is_recurring") == "on" || ctx.PostForm("is_recurring") == "true",
		RecurrenceInterval: strings.TrimSpace(ctx.PostForm("recurrence_interval")),
		SheetURL:           strings.TrimSpace(ctx.PostForm("sheet_url")),
		SolutionURL:        strings.TrimSpace(ctx.PostForm("solution_url")),
	}
	if startDate := parseFormDate(ctx.PostForm("start_date")); startDate != nil {
		task.StartDate = startDate
	}
	if deadline := parseFormDate(ctx.PostForm("deadline")); deadline != nil {
		task.Deadline = deadline
	}
	if recurrenceUntil := parseFormDate(ctx.PostForm("recurrence_until")); recurrenceUntil != nil {
		task.RecurrenceUntil = recurrenceUntil
	}
	if task.IsRecurring && task.RecurrenceInterval == "" {
		task.RecurrenceInterval = "weekly"
	}
	if task.RecurrenceInterval == "" {
		task.IsRecurring = false
	}
	if value := parseOptionalFloat(ctx.PostForm("possible_points")); value != nil {
		task.PossiblePoints = value
	}
	if value := parseOptionalFloat(ctx.PostForm("achieved_points")); value != nil {
		task.AchievedPoints = value
	}
	task.Normalize()
	return task, nil
}

func parseAssignmentSeriesForm(ctx *gin.Context) (AssignmentSeries, error) {
	name := strings.TrimSpace(ctx.PostForm("name"))
	if name == "" {
		return AssignmentSeries{}, errors.New("name is required")
	}
	subjectID := SubjectID(strings.TrimSpace(ctx.PostForm("subject_id")))
	if subjectID == "" {
		return AssignmentSeries{}, errors.New("subject is required")
	}
	series := AssignmentSeries{
		Name:               name,
		SubjectID:          subjectID,
		Kind:               AssignmentKind(strings.TrimSpace(ctx.PostForm("kind"))),
		Requirement:        AssignmentRequirement(strings.TrimSpace(ctx.PostForm("requirement"))),
		RecurrenceInterval: strings.TrimSpace(ctx.PostForm("recurrence_interval")),
		ReleaseTime:        strings.TrimSpace(ctx.PostForm("release_time")),
		DeadlineTime:       strings.TrimSpace(ctx.PostForm("deadline_time")),
		SheetURL:           strings.TrimSpace(ctx.PostForm("sheet_url")),
		SolutionURL:        strings.TrimSpace(ctx.PostForm("solution_url")),
	}
	series.DeadlineWeekday = time.Weekday(parseIntDefault(ctx.PostForm("deadline_weekday"), int(time.Friday)))
	series.StartDate = parseFormDate(ctx.PostForm("start_date"))
	series.RecurrenceUntil = parseFormDate(ctx.PostForm("recurrence_until"))
	if series.StartDate == nil {
		return AssignmentSeries{}, errors.New("release date is required")
	}
	exerciseCount := parseIntDefault(ctx.PostForm("exercise_count"), 1)
	if exerciseCount < 1 {
		exerciseCount = 1
	}
	for number := 1; number <= exerciseCount; number++ {
		series.Exercises = append(series.Exercises, Exercise{
			Number: number,
			Name:   fmt.Sprintf("Exercise %d", number),
			State:  TaskStateNotStarted,
		})
	}
	series.Normalize()
	return series, nil
}

func parseIntDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseOptionalFloat(value string) *float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseFormDate(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := time.Parse(time.DateOnly, trimmed)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, trimmed)
		if err != nil {
			return nil
		}
	}
	return &parsed
}

func computeSummary(subjectID SubjectID, tasks []*Task) SubjectSummary {
	summary := SubjectSummary{SubjectName: "All subjects"}
	for _, task := range tasks {
		if subjectID != "" && task.SubjectID != subjectID {
			continue
		}
		if task.PossiblePoints != nil && *task.PossiblePoints > 0 {
			summary.Possible += *task.PossiblePoints
		}
		if task.AchievedPoints != nil {
			summary.Achieved += *task.AchievedPoints
		}
	}
	if summary.Possible > 0 {
		summary.Percent = (summary.Achieved / summary.Possible) * 100
	}
	if subjectID != "" {
		summary.SubjectName = string(subjectID)
	}
	return summary
}
