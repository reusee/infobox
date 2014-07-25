package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	xsrf   string
	kv     KvStore
	cookie string
}

func NewZhihuCollector(kv KvStore) (Collector, error) {
	var xsrf string
	if res := kv.KvGet("zhihu-xsrf"); res != nil {
		xsrf = res.(string)
	}
	z := &ZhihuCollector{
		kv:   kv,
		xsrf: xsrf,
	}
	var cookie string
	if res := kv.KvGet("zhihu-cookie"); res != nil {
		cookie = res.(string)
	}
	if cookie == "" {
		err := z.Login()
		if err != nil {
			return nil, err
		}
	} else {
		z.cookie = cookie
	}
	return z, nil
}

func (z *ZhihuCollector) Login() error {
	resp, err := Get("http://www.zhihu.com/#signin")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	res := regexp.MustCompile(`<input type="hidden" name="_xsrf" value="([^"]+)"/>`).FindSubmatch(content)
	if len(res) == 0 {
		return Err("zhihu: cannot get xsrf value")
	}
	xsrf := string(res[1])
	cookie := RespGetCookieStr(resp)
	//p("=> %s\n", cookie)

	var user, pass string
	if res := z.kv.KvGet("zhihu-username"); res != nil {
		user = res.(string)
	}
	if res := z.kv.KvGet("zhihu-password"); res != nil {
		pass = res.(string)
	}
	if user == "" || pass == "" {
		p("input zhihu username: ")
		fmt.Scanf("%s", &user)
		p("input zhihu password: ")
		fmt.Scanf("%s", &pass)
		z.kv.KvSet("zhihu-username", user)
		z.kv.KvSet("zhihu-password", pass)
	}

	resp, err = PostFormWithCookie("http://www.zhihu.com/login", url.Values{
		"_xsrf":      {xsrf},
		"email":      {user},
		"password":   {pass},
		"rememberme": {"y"},
	}, cookie)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	//io.Copy(os.Stdout, resp.Body)
	cookie = RespGetCookieStr(resp)
	//p("-> %s\n", cookie)
	z.xsrf = xsrf
	z.kv.KvSet("zhihu-xsrf", xsrf)
	z.kv.KvSet("zhihu-cookie", cookie)
	z.cookie = cookie

	return nil
}

func (z *ZhihuCollector) Collect() (ret []Entry, err error) {
	if InteractiveMode {
		z.Login()
	}
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
	resp, err := PostFormWithCookie("http://www.zhihu.com/node/HomeFeedListV2", url.Values{
		"params": {s(`{"offset": 21, "start": "%s"}`, start)},
		"method": {"next"},
		"_xsrf":  {z.xsrf},
	}, z.cookie)
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
