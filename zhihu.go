package main

import (
	"fmt"
	"net/url"
	"regexp"
)

type ZhihuCollector struct {
	client *Client
}

func NewZhihuCollector(client *Client) (Collector, error) {
	return &ZhihuCollector{
		client: client,
	}, nil
}

func (z *ZhihuCollector) Login() error {
	content, err := z.client.GetBytes("http://www.zhihu.com/#signin", nil)
	if err != nil {
		return err
	}
	res := regexp.MustCompile(`<input type="hidden" name="_xsrf" value="([^"]+)"/>`).FindSubmatch(content)
	if len(res) == 0 {
		return Err("zhihu: cannot get xsrf value")
	}
	xsrf := string(res[1])

	var user, pass string
	p("input zhihu username: ")
	fmt.Scanf("%s", &user)
	p("input zhihu password: ")
	fmt.Scanf("%s", &pass)

	resp, err := z.client.PostForm("http://www.zhihu.com/login", url.Values{
		"_xsrf":      {xsrf},
		"email":      {user},
		"password":   {pass},
		"rememberme": {"y"},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (z *ZhihuCollector) Collect() (ret []Entry, err error) {
	// get content
	content, err := z.client.GetBytes("http://www.zhihu.com/", nil)
	if err != nil {
		return nil, err
	}
	content, err = tidyHtml(content)
	if err != nil {
		return nil, err
	}

	// parse

	return
}
