package main

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type Jar struct {
	Store map[string]map[string]*http.Cookie
	lock  sync.Mutex
}

func NewJar() *Jar {
	return &Jar{
		Store: make(map[string]map[string]*http.Cookie),
	}
}

func (j *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.lock.Lock()
	defer j.lock.Unlock()
	for _, cookie := range cookies {
		domain := u.Host
		if cookie.Domain != "" {
			domain = cookie.Domain
		}
		m, ok := j.Store[domain]
		if !ok {
			m = make(map[string]*http.Cookie)
			j.Store[domain] = m
		}
		m[cookie.Name] = cookie
	}
}

func (j *Jar) Cookies(u *url.URL) (ret []*http.Cookie) {
	j.lock.Lock()
	defer j.lock.Unlock()
	for domain, cookies := range j.Store {
		if domain == u.Host || (domain != "" && strings.Contains(u.Host, domain)) {
			for _, cookie := range cookies {
				ret = append(ret, cookie)
			}
		}
	}
	return
}

func (j *Jar) init() {
	if j.Store == nil {
		j.Store = make(map[string]map[string]*http.Cookie)
	}
}
