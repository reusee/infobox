package main

import (
	"net/http"
	"net/url"
	"strings"
)

type Jar struct {
	Store map[string]map[string]*http.Cookie
}

func NewJar() *Jar {
	return &Jar{
		Store: make(map[string]map[string]*http.Cookie),
	}
}

func (j *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
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
	for domain, cookies := range j.Store {
		if domain == u.Host || strings.Contains(u.Host, domain) {
			for _, cookie := range cookies {
				ret = append(ret, cookie)
			}
		}
	}
	return
}

func (j *Jar) SetFromString(domain, str string) {
	pairs := strings.Split(str, ";")
	m, ok := j.Store[domain]
	if !ok {
		m = make(map[string]*http.Cookie)
	}
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		var value string
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		}
		m[strings.TrimSpace(parts[0])] = &http.Cookie{
			Name:  name,
			Value: value,
		}
	}
	j.Store[domain] = m
}
