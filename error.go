package main

import (
	"encoding/gob"
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
