package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/PuerkitoBio/goquery"

	"net/url"
	"regexp"
)

func init() {
	gob.Register(new(ZhihuEntry))
}

type ZhihuCollector struct {
	client *Client
	xsrf   string
	kv     KvStore
}

func NewZhihuCollector(client *Client, kv KvStore) (Collector, error) {
	var xsrf string
	if res := kv.KvGet("zhihu-xsrf"); res != nil {
		xsrf = res.(string)
	}
	return &ZhihuCollector{
		client: client,
		kv:     kv,
		xsrf:   xsrf,
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
	z.xsrf = xsrf
	z.kv.KvSet("zhihu-xsrf", xsrf)

	return nil
}

func (z *ZhihuCollector) Collect() (ret []Entry, err error) {
	if z.xsrf == "" {
		p("zhihu: no xsrf.\n")
		if InteractiveMode {
			z.Login()
		} else {
			return nil, Err("zhihu auth error")
		}
	}

	var start string
	n := 10
	// get content
get:
	resp, err := z.client.PostForm("http://www.zhihu.com/node/HomeFeedListV2", url.Values{
		"params": {s(`{"offset": 21, "start": "%s"}`, start)},
		"method": {"next"},
		"_xsrf":  {z.xsrf},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// parse
	structure := struct {
		R   int
		Msg []string
	}{}
	err = json.NewDecoder(resp.Body).Decode(&structure)
	if err != nil {
		return nil, err
	}
	if structure.R != 0 {
		return nil, Err("return non-zero")
	}

	// collect
	for _, msg := range structure.Msg {
		html, err := tidyHtml([]byte(msg))
		if err != nil {
			return nil, err
		}
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
		if err != nil {
			return nil, err
		}
		titleA := doc.Find("div.content h2 a")
		title := titleA.Text()
		if title == "" {
			return nil, Err("no title")
		}
		body := doc.Find("div.content div.entry-body")
		content := body.Text()
		_ = content
		id, ok := doc.Find("div.feed-item").Attr("id")
		if !ok {
			return nil, Err("no id")
		}
		start = strings.Replace(id, "feed-", "", -1)
		link, ok := doc.Find("div.content h2 a").Attr("href")
		if !ok {
			return nil, Err("no link")
		}
		if !strings.HasPrefix(link, "http") {
			link = "http://www.zhihu.com" + link
		}
		ret = append(ret, &ZhihuEntry{
			Id:      id,
			Title:   title,
			Content: content,
			Link:    link,
		})
	}
	n -= 1
	if n > 0 {
		goto get
	}

	p("collect %d entries from zhihu.\n", len(ret))
	return
}

type ZhihuEntry struct {
	Id      string
	Title   string
	Content string
	Link    string
}

func (z *ZhihuEntry) GetKey() string {
	return s("zhihu %s", z.Id)
}

func (z *ZhihuEntry) ToRssItem() RssItem {
	return RssItem{
		Title:  z.Title,
		Link:   z.Link,
		Desc:   z.Content,
		Author: "Zhihu",
	}
}

var zhihuHtmlTemplate = template.Must(template.New("zhihu").Parse(`
<h2>Zhihu</h2>
<p>{{.Title}}</p>
<div>{{.Content}}</div>
`))

func (z *ZhihuEntry) ToHtml() string {
	buf := new(bytes.Buffer)
	err := zhihuHtmlTemplate.Execute(buf, z)
	if err != nil {
		return s("render error %v", err)
	}
	return string(buf.Bytes())
}
