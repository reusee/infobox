package main

import "math/rand"

type ErrorEntry struct {
	Id      int64
	Message string
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

func NewErrorEntry(e error) *ErrorEntry {
	return &ErrorEntry{
		Id:      rand.Int63(),
		Message: s("%v", e),
	}
}
