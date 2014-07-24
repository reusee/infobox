package main

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var defaultClient *http.Client

func init() {
	defaultClient = &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, time.Second*30)
			},
			ResponseHeaderTimeout: time.Second * 30,
		},
		Timeout: time.Second * 30,
	}
}

func GetBytesWithCookie(url string, cookieStr string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", cookieStr)
	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func GetBytes(url string) ([]byte, error) {
	resp, err := defaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func Get(url string) (*http.Response, error) {
	return defaultClient.Get(url)
}

func GetWithCookie(url string, cookieStr string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", cookieStr)
	return defaultClient.Do(req)
}

func GetCookieStr(url string) (string, error) {
	resp, err := Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return RespGetCookieStr(resp), nil
}

func PostForm(url string, values url.Values) (*http.Response, error) {
	return defaultClient.PostForm(url, values)
}

func PostFormWithCookie(url string, values url.Values, cookieStr string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieStr)
	return defaultClient.Do(req)
}

func RespGetCookieStr(resp *http.Response) string {
	buf := new(bytes.Buffer)
	for _, entry := range resp.Header["Set-Cookie"] {
		parts := strings.SplitN(entry, ";", 2)
		buf.WriteString(parts[0])
		buf.WriteString("; ")
	}
	return string(buf.Bytes())
}
