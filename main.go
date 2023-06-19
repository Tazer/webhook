package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
)

func main() {

	reqLogs := NewRequestLogs()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(c *gin.Context) {

		id := ulid.Make()

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"url":     "/log/" + id.String(),
			"urllogs": "/log/" + id.String() + "/logs",
		})
	})
	r.Any("/log/:id", func(c *gin.Context) {

		reqLog := RequestLog{}

		defer c.Request.Body.Close()
		respbody, err := ioutil.ReadAll(c.Request.Body)
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
	r.GET("/log/:id/logs", func(c *gin.Context) {

		logs := reqLogs.Get(c.Param("id"))

		c.JSON(http.StatusOK, logs)
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

type RequestLogs struct {
	mu   sync.Mutex
	logs map[string][]RequestLog
}

func NewRequestLogs() *RequestLogs {
	return &RequestLogs{
		mu:   sync.Mutex{},
		logs: map[string][]RequestLog{},
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
