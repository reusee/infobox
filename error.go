package main

import (
	"bytes"
	"encoding/gob"
	"html/template"
	"math/rand"
)

func init() {
	gob.Register(new(ErrorEntry))
}

type ErrorEntry struct {
	Id      int64
	Message string
}

func NewErrorEntry(e error) *ErrorEntry {
	return &ErrorEntry{
		Id:      rand.Int63(),
		Message: s("%v", e),
	}
}

func (e *ErrorEntry) GetKey() string {
	return s("error %d", e.Id)
}

func (e *ErrorEntry) ToRssItem() RssItem {
	return RssItem{
		Title:  e.Message,
		Desc:   e.Message,
		Author: "Error",
	}
}

var errorHtmlTemplate = template.Must(template.New("error").Parse(`
<h2>Error</h2>
<p>{{.Message}}</p>
`))

func (e *ErrorEntry) ToHtml() string {
	buf := new(bytes.Buffer)
	err := errorHtmlTemplate.Execute(buf, e)
	if err != nil {
		return s("render error %v", err)
	}
	return string(buf.Bytes())
}
