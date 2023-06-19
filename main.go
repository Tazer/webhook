package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
)

var (
	//go:embed templates/*
	templates embed.FS
)

func main() {

	reqLogs := NewRequestLogs()

	r := gin.Default()
	templ := template.Must(template.New("").ParseFS(templates, "templates/*.tmpl"))
	r.SetHTMLTemplate(templ)
	r.GET("/", func(c *gin.Context) {

		id := ulid.Make()

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"id":      id.String(),
			"url":     "/log/" + id.String(),
			"urllogs": "/log/" + id.String() + "/logs",
		})
	})

	r.POST("/log/:id/password", func(c *gin.Context) {
		//setPass := SetPassword{}
		//err := c.MustBindWith(&setPass, binding.JSON)
		//if err != nil {
		//	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		//	return
		//}

		err := reqLogs.AddPassword(c.Param("id"), c.PostForm("password"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "accepted",
		})
	})

	r.Any("/log/:id", func(c *gin.Context) {

		reqLog := RequestLog{}

		defer c.Request.Body.Close()
		respbody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		reqLog.Body = string(respbody)
		reqLog.Headers = c.Request.Header
		reqLog.URL = c.Request.URL
		reqLog.Method = c.Request.Method

		reqLogs.Add(c.Param("id"), reqLog)
		c.JSON(http.StatusOK, gin.H{
			"status": "accepted",
		})
	})
	r.Any("/log/:id/fail/:status", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{
			"status": "accepted",
		})
	})
	r.POST("/log/:id/logs", func(c *gin.Context) {
		if !reqLogs.CheckPassword(c.Param("id"), c.PostForm("password")) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
			return
		}

		renderLogs(c, reqLogs)
	})
	r.GET("/log/:id/logs", func(c *gin.Context) {

		if reqLogs.HasPassword(c.Param("id")) {
			c.HTML(http.StatusOK, "password.tmpl", gin.H{
				"id": c.Param("id"),
			})
			return
		}

		renderLogs(c, reqLogs)
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func renderLogs(c *gin.Context, reqLogs *RequestLogs) {
	logs := reqLogs.Get(c.Param("id"))

	logsJson, err := json.Marshal(logs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.HTML(http.StatusOK, "logs.tmpl", gin.H{"logs": logs, "logs_json": logsJson})
}

type RequestLogs struct {
	mu        sync.Mutex
	logs      map[string][]RequestLog
	passwords map[string]string
}

func NewRequestLogs() *RequestLogs {
	return &RequestLogs{
		mu:        sync.Mutex{},
		logs:      map[string][]RequestLog{},
		passwords: map[string]string{},
	}
}

func (r *RequestLogs) Add(id string, reqLog RequestLog) {
	r.mu.Lock()
	defer r.mu.Unlock()
	//if r.logs[id] == nil {
	//	r.logs[id] = []RequestLog{}
	//}
	r.logs[id] = append(r.logs[id], reqLog)
}

func (r *RequestLogs) AddPassword(id string, password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.passwords[id]; ok {
		return fmt.Errorf("password already set")
	}

	r.passwords[id] = password
	return nil
}

func (r *RequestLogs) HasPassword(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.passwords[id]
	return ok
}

func (r *RequestLogs) CheckPassword(id string, password string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.passwords[id]; !ok {
		return true
	}

	return r.passwords[id] == password
}

func (r *RequestLogs) Get(id string) []RequestLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.logs[id]
}

type RequestLog struct {
	Headers http.Header
	Body    string
	URL     *url.URL
	Method  string
}

type SetPassword struct {
	Password string `json:"password"`
}
