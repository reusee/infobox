package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"code.google.com/p/goauth2/oauth"
)

var ignoreUsers = []string{
	"rubysola",
	"naomi-wang",
	"foxy",
	"mia1612",
}

func init() {
	gob.Register(new(DoubanEntry))
}

type DoubanCollector struct {
	client *http.Client
	*ErrorHost
}

func NewDoubanCollector(tokenCache oauth.Cache) (Collector, error) {
	c := &DoubanCollector{
		ErrorHost: NewErrorHost("Douban"),
	}

	// oauth config
	config := &oauth.Config{
		ClientId:     "07f3402dcfdf369d17d9c0896f9da3d7",
		ClientSecret: "53f98ee44918973f",
		AuthURL:      "https://www.douban.com/service/auth2/auth",
		TokenURL:     "https://www.douban.com/service/auth2/token",
		RedirectURL:  "foobar",
		TokenCache:   tokenCache,
	}
	transport := &oauth.Transport{Config: config}

	// get token
	token, err := config.TokenCache.Token()
	auth := func() error {
		url := config.AuthCodeURL("")
		p("%s\n", url)
		var code string
		p("enter douban auth code: ")
		fmt.Scanf("%s", &code)
		token, err = transport.Exchange(code)
		if err != nil {
			return c.Err("exchange token %v", err)
		}
		return nil
	}
	if err != nil {
		if InteractiveMode {
			err = auth()
		} else {
			err = Err("douban auth error")
		}
		if err != nil {
			return nil, err
		}
	}
	transport.Token = token
	c.client = transport.Client()

	// ping
validate:
	uri := "https://api.douban.com/v2/user/~me"
	resp, err := c.client.Get(uri)
	if err != nil {
		return nil, c.Err("get %s %v", uri, err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, c.Err("read body %s %v", uri, err)
	}
	var ret struct {
		Msg  string
		Code int
	}
	err = json.Unmarshal(content, &ret)
	if ret.Code != 0 { // token error
		if InteractiveMode {
			err = auth()
		} else {
			err = Err("douban auth error")
		}
		if err != nil {
			return nil, err
		}
		goto validate
	}

	return c, nil
}

func (d *DoubanCollector) Collect() (ret []Entry, err error) {
	max := 10
	wg := new(sync.WaitGroup)
	wg.Add(max)
	lock := new(sync.Mutex)
	errors := make([]error, 0, max)
	for i := 0; i < max; i++ {
		go func(i int) {
			r, err := d.CollectTimeline(i)
			lock.Lock()
			errors = append(errors, err)
			ret = append(ret, r...)
			lock.Unlock()
			wg.Done()
		}(i)
	}
	wg.Wait()
	for _, e := range errors {
		if e != nil {
			err = e
		}
	}
	p("collect %d entries from douban.\n", len(ret))
	return
}

func (d *DoubanCollector) CollectTimeline(i int) (ret []Entry, err error) {
	perPage := 200
	url := fmt.Sprintf("https://api.douban.com/shuo/v2/statuses/home_timeline?count=%d&start=%d",
		perPage, i*perPage)
	resp, err := d.client.Get(url)
	if err != nil {
		return nil, d.Err("get %s %v", url, err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, d.Err("read body %s %v", url, err)
	}

	//buf := new(bytes.Buffer)
	//json.Indent(buf, content, "", "    ")
	//p("%s\n====================\n", buf.Bytes())

	var result []*DoubanEntry
	err = json.Unmarshal(content, &result)
	if err != nil {
		// try to unmarshal as error message
		var msg struct {
			Msg     string
			Code    int
			Request string
		}
		err = json.Unmarshal(content, &msg)
		if err != nil {
			return nil, d.Err("unmarshal %v %s", err, content)
		} else {
			return nil, d.Err("api error %s %s", msg.Msg, msg.Request)
		}
	}
loop:
	for _, entry := range result {
		for _, name := range ignoreUsers {
			if entry.User.Uid == name {
				continue loop
			}
		}
		ret = append(ret, entry)
	}

	return
}

type DoubanEntry struct {
	Title       string
	Text        string
	CreatedAt   string `json:"created_at"`
	Attachments []struct {
		Description string
		Title       string
		Media       []struct {
			Src      string
			Original string `json:"original_src"`
			Href     string
			Type     string
		}
		Href string
		Type string
	}
	Id   int
	User struct {
		Uid    string
		Avatar string `json:"large_avatar"`
		Name   string `json:"screen_name"`
	}
	Reshared *DoubanEntry `json:"reshared_status"`
}

func (d DoubanEntry) GetKey() string {
	return s("douban %d", d.Id)
}

func (d *DoubanEntry) ToRssItem() RssItem {
	parts := d.collectParts()
	return RssItem{
		Title:  strings.Join(parts, " - "),
		Link:   s("http://www.douban.com/people/%s/status/%d/", d.User.Uid, d.Id),
		Desc:   strings.Join(parts, " - "),
		Author: "Douban",
	}
}

var doubanHtmllTemplate = template.Must(template.New("douban").Parse(`
<h2>Douban</h2>
<p>{{.User.Name}}</p>
<p>{{.Title}}</p>
{{range $index, $elem := .Attachments}}
<div class="attachment">
	<p>{{$elem.Title}}</p>
	<p>{{$elem.Description}}</p>
	{{range $i, $e := $elem.Media}}
		{{if eq $e.Type "image"}}
		<p><img src="{{$e.Src}}" /></p>
		{{end}}
	{{end}}
</div>
{{end}}
<p>{{.Text}}</p>
`))

func (d *DoubanEntry) ToHtml() string {
	buf := new(bytes.Buffer)
	err := doubanHtmllTemplate.Execute(buf, d)
	if err != nil {
		return s("render error %v", err)
	}
	if d.Reshared != nil {
		err := doubanHtmllTemplate.Execute(buf, d.Reshared)
		if err != nil {
			return s("render error %v", err)
		}
	}
	return string(buf.Bytes())
}

func (d *DoubanEntry) collectParts() []string {
	var parts []string
	var hasImage bool
	parts = append(parts, d.User.Name)
	parts = append(parts, d.Title)
	for _, attach := range d.Attachments {
		if attach.Title != "" {
			parts = append(parts, attach.Title)
		}
		if attach.Type == "image" {
			hasImage = true
		}
	}
	if d.Text != "" {
		parts = append(parts, d.Text)
	}
	if d.Reshared != nil {
		parts = append(parts, d.Reshared.collectParts()...)
	}
	if hasImage {
		parts = append([]string{"[图]"}, parts...)
	}
	return parts
}
