package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type Client struct {
	*http.Client
}

func NewClient(jar http.CookieJar) *Client {
	return &Client{
		Client: &http.Client{
			Transport: &http.Transport{
				Dial: func(network, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, time.Second*30)
				},
				ResponseHeaderTimeout: time.Second * 30,
			},
			Jar:     jar,
			Timeout: time.Second * 30,
		},
	}
}

func (c *Client) GetBytes(url string, extraHeader map[string][]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if extraHeader != nil {
	}
	for key, value := range extraHeader {
		req.Header[key] = value
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
