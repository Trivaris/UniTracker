package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Routing struct {
	store *Store
	engine *gin.Engine
}

func NewRouting(store *Store) *Routing {
	return &Routing{
		store: store,
		engine: gin.Default(),
	}
}

func (r *Routing) Route() {
	r.engine.GET("/subject", func(ctx *gin.Context) {
		subjects := r.store.GetAllSubjects()
		ctx.IndentedJSON(http.StatusAccepted, subjects)
	})

	r.engine.GET("/task", func(ctx *gin.Context) {
		tasks := r.store.GetAllTasks()
		ctx.IndentedJSON(http.StatusAccepted, tasks)
	})
	
	r.engine.GET("/subject/:id", func(ctx *gin.Context) {
		id := SubjectID(ctx.Param("id"))
		subject := r.store.GetSubject(id)
		if subject == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("subject with id %s not found", id))
			return
		}
		ctx.IndentedJSON(http.StatusAccepted, subject)
	})

	r.engine.GET("/task/:id", func(ctx *gin.Context) {
		id := TaskID(ctx.Param("id"))
		task := r.store.GetTask(id)
		if task == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("task with id %s not found", id))
			return
		}
		ctx.IndentedJSON(http.StatusAccepted, task)
	})

	r.engine.POST("/subject", func(ctx *gin.Context) {
		name := ctx.PostForm("name")
		subject, err := r.store.StoreSubject(name)		
		if err != nil {
			ctx.String(http.StatusAccepted, err.Error())
		}
		ctx.IndentedJSON(http.StatusAccepted, subject)
	})

	r.engine.POST("/task", func(ctx *gin.Context) {
		name := ctx.PostForm("name")
		subjectID := SubjectID(ctx.PostForm("subject"))
		task, err := r.store.StoreTask(name, subjectID)		
		if err != nil {
			ctx.String(http.StatusAccepted, err.Error())
		}
		ctx.IndentedJSON(http.StatusAccepted, task)
	})

	r.engine.Run()
}