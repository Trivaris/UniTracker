package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Routing struct {
	store Store
	engine *gin.Engine
}

func NewRouting(store Store) *Routing {
	return &Routing{
		store: store,
		engine: gin.Default(),
	}
}

func (r *Routing) Route() error {
	r.engine.GET("/subject", func(ctx *gin.Context) {
		subjects, err := r.store.GetAllSubjects(ctx)
		switch err {
		case nil: {
			ctx.IndentedJSON(http.StatusOK, subjects)
		}
		case ErrSubjectNotFound: {
			ctx.Status(http.StatusNotFound)
		}
		default: {
			ctx.String(http.StatusInternalServerError, err.Error())
		}
		}
	})

	r.engine.GET("/task", func(ctx *gin.Context) {
		tasks, err := r.store.GetAllTasks(ctx)
		switch err {
		case nil: {
			ctx.IndentedJSON(http.StatusOK, tasks)
		}
		case ErrTaskNotFound: {
			ctx.Status(http.StatusNotFound)
		}
		default: {
			ctx.String(http.StatusInternalServerError, err.Error())
		}
		}
	})
	
	r.engine.GET("/subject/:id", func(ctx *gin.Context) {
		id := SubjectID(ctx.Param("id"))
		subject, err := r.store.GetSubject(ctx.Request.Context(), id)
		switch err {
		case nil: {
			ctx.IndentedJSON(http.StatusOK, subject)
		}
		case ErrSubjectNotFound: {
			ctx.Status(http.StatusNotFound)
		}
		default: {
			ctx.String(http.StatusInternalServerError, err.Error())
		}
		}
	})

	r.engine.GET("/task/:id", func(ctx *gin.Context) {
		id := TaskID(ctx.Param("id"))
		task, err := r.store.GetTask(ctx.Request.Context(), id)
		switch err {
		case nil: {
			ctx.IndentedJSON(http.StatusOK, task)
		}
		case ErrTaskNotFound: {
			ctx.Status(http.StatusNotFound)
		}
		default: {
			ctx.String(http.StatusInternalServerError, err.Error())
		}
		}
	})

	r.engine.POST("/subject", func(ctx *gin.Context) {
		name := ctx.PostForm("name")
		subject, err := r.store.StoreSubject(ctx.Request.Context(), name)		
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		ctx.IndentedJSON(http.StatusCreated, subject)
	})

	r.engine.POST("/task", func(ctx *gin.Context) {
		name := ctx.PostForm("name")
		subjectID := SubjectID(ctx.PostForm("subject"))
		task, err := r.store.StoreTask(ctx.Request.Context(), name, subjectID)		
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return 
		}
		ctx.IndentedJSON(http.StatusCreated, task)
	})

	return r.engine.Run()
}