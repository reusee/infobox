package main

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
)

type Jar struct {
	lock  sync.Mutex
	Cache map[string]JarCacheEntry
	jar   *cookiejar.Jar
}

type JarCacheEntry struct {
	URL    URL
	Cookie *http.Cookie
}

type URL struct {
	Scheme string
	Host   string
	Path   string
}

func NewJar() *Jar {
	jar, _ := cookiejar.New(nil)
	return &Jar{
		Cache: make(map[string]JarCacheEntry),
		jar:   jar,
	}
}

func (j *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.lock.Lock()
	defer j.lock.Unlock()
	for _, cookie := range cookies {
		j.Cache[s("%s %s", u.Host, cookie.Name)] = JarCacheEntry{
			URL: URL{
				Scheme: u.Scheme,
				Host:   u.Host,
				Path:   u.Path,
			},
			Cookie: cookie,
		}
	}
	j.jar.SetCookies(u, cookies)
}

func (j *Jar) Cookies(u *url.URL) (ret []*http.Cookie) {
	return j.jar.Cookies(u)
}

func (j *Jar) rebuild() {
	if j.Cache == nil {
		j.Cache = make(map[string]JarCacheEntry)
	}
	if j.jar == nil {
		jar, _ := cookiejar.New(nil)
		j.jar = jar
	}
	for _, entry := range j.Cache {
		j.jar.SetCookies(&url.URL{
			Scheme: entry.URL.Scheme,
			Host:   entry.URL.Host,
			Path:   entry.URL.Path,
		}, []*http.Cookie{
			entry.Cookie,
		})
	}
}
