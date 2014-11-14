package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/reusee/gobchest"

	"code.google.com/p/go.net/html"
)

var p = fmt.Printf
var f = fmt.Fprintf
var s = fmt.Sprintf

func tidyHtml(input []byte) ([]byte, error) {
	// tidy
	nodes, err := html.ParseFragment(bytes.NewReader(input), nil)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	for _, node := range nodes {
		err = html.Render(buf, node)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func find(obj interface{}, predict func(interface{}) bool) interface{} {
	if predict(obj) {
		return obj
	}
	switch reflect.TypeOf(obj).Kind() {
	case reflect.Array, reflect.Slice:
		value := reflect.ValueOf(obj)
		l := value.Len()
		for i := 0; i < l; i++ {
			elem := value.Index(i)
			if result := find(elem.Interface(), predict); result != nil {
				return result
			}
		}
	case reflect.Map:
		value := reflect.ValueOf(obj)
		for _, key := range value.MapKeys() {
			elem := value.MapIndex(key)
			if result := find(elem.Interface(), predict); result != nil {
				return result
			}
		}
	case reflect.Struct:
		value := reflect.ValueOf(obj)
		n := value.NumField()
		for i := 0; i < n; i++ {
			field := value.Field(i)
			if result := find(field.Interface(), predict); result != nil {
				return result
			}
		}
	}
	return nil
}

type KvStore interface {
	KvGet(string) interface{}
	KvSet(string, interface{})
}

type kvclient struct {
	*gobchest.Client
}

func (k *kvclient) KvGet(key string) interface{} {
	v, err := k.Get("infobox." + key)
	if err != nil {
		return nil
	}
	return v
}

func (k *kvclient) KvSet(key string, v interface{}) {
	err := k.Set("infobox."+key, v)
	if err != nil {
		log.Fatalf("KvSet: %v", err)
	}
}

func NewKvStore(client *gobchest.Client) KvStore {
	return &kvclient{client}
}

func Err(format string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(format, args...))
}

type ErrorHost struct {
	prefix string
}

func NewErrorHost(prefix string) *ErrorHost {
	return &ErrorHost{
		prefix: prefix,
	}
}

func (e *ErrorHost) Err(format string, args ...interface{}) error {
	return errors.New(s("%s %s", e.prefix, s(format, args...)))
}
