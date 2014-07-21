package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"code.google.com/p/goauth2/oauth"
)

func init() {
	gob.Register(new(DoubanEntry))
}

type DoubanCollector struct {
	client *http.Client
}

func NewDoubanCollector(tokenCache oauth.Cache) (Collector, error) {
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
		if SilenceMode { // do not auth in silence mode
			return Err("douban auth error")
		}
		url := config.AuthCodeURL("")
		p("%s\n", url)
		var code string
		p("enter douban auth code: ")
		fmt.Scanf("%s", &code)
		token, err = transport.Exchange(code)
		if err != nil {
			return Err("exchange douban token %v", err)
		}
		return nil
	}
	if err != nil {
		err = auth()
		if err != nil {
			return nil, err
		}
	}
	transport.Token = token
	c := &DoubanCollector{
		client: transport.Client(),
	}

	// ping
validate:
	resp, err := c.client.Get("https://api.douban.com/v2/user/~me")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var ret struct {
		Msg  string
		Code int
	}
	err = json.Unmarshal(content, &ret)
	if ret.Code != 0 { // token error
		err = auth()
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
		return nil, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//buf := new(bytes.Buffer)
	//json.Indent(buf, content, "", "    ")
	//p("%s\n", buf.Bytes())

	var result []*DoubanEntry
	err = json.Unmarshal(content, &result)
	if err != nil {
		return nil, err
	}
	for _, entry := range result {
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
		} `json:"media"`
		Href string
		Type string
	}
	Id   int
	User struct {
		Uid    string
		Avatar string `json:"large_avatar"`
		Name   string `json:"screen_name"`
	}
	Reshared *DoubanEntry `json:"reshared_status"` //TODO not work
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

func (d *DoubanEntry) collectParts() []string {
	parts := []string{
		d.User.Name,
		d.Title,
	}
	for _, attach := range d.Attachments {
		if attach.Title != "" {
			parts = append(parts, attach.Title)
		}
	}
	if d.Text != "" {
		parts = append(parts, d.Text)
	}
	if d.Reshared != nil {
		parts = append(parts, d.Reshared.collectParts()...)
	}
	return parts
}
