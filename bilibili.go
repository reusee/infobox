package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/reusee/hns"
)

func init() {
	gob.Register(new(BilibiliEntry))
}

type BilibiliCollector struct {
	kv     KvStore
	cookie string
	*ErrorHost
}

func NewBilibiliCollector(kv KvStore) (*BilibiliCollector, error) {
	b := &BilibiliCollector{
		kv:        kv,
		ErrorHost: NewErrorHost("Bilibili"),
	}
	var cookie string
	if res := b.kv.KvGet("bilibili-cookie"); res != nil {
		cookie = res.(string)
	}
	if cookie == "" {
		err := b.Login()
		if err != nil {
			return nil, err
		}
	} else {
		b.cookie = cookie
	}
	return b, nil
}

var bilibiliLoginError = errors.New("bilibili login error")

func (b *BilibiliCollector) Collect() (ret []Entry, err error) {
	for _, fun := range []func(int) ([]Entry, error){
		b.CollectTimeline,
		func(page int) ([]Entry, error) {
			return b.CollectNewest("http://www.bilibili.com/video/bangumi-two-%d.html", page)
		},
		func(page int) ([]Entry, error) {
			return b.CollectNewest("http://www.bilibili.com/video/douga-else-information-%d.html", page)
		},
	} {
		maxPage := 10
		sem := make(chan bool, 2)
		wg := new(sync.WaitGroup)
		wg.Add(maxPage)
		lock := new(sync.Mutex)
		errors := make([]error, 0, maxPage)
		for page := 1; page <= maxPage; page++ {
			sem <- true
			go func(page int) {
				defer wg.Done()
				entries, err := fun(page)
				lock.Lock()
				ret = append(ret, entries...)
				errors = append(errors, err)
				lock.Unlock()
				<-sem
			}(page)
		}
		wg.Wait()
		for _, e := range errors {
			if e != nil {
				if e == bilibiliLoginError {
					if InteractiveMode {
						err = b.Login()
						if err != nil {
							return nil, err
						}
						return b.Collect()
					} else {
						return nil, b.Err("need login")
					}
				}
				err = e
			}
		}
	}

	p("collected %d entries from bilibili.\n", len(ret))
	return
}

func (b *BilibiliCollector) CollectTimeline(page int) (ret []Entry, err error) {
	// get content
	url := fmt.Sprintf("http://www.bilibili.com/account/dynamic/dyn-%d", page)
	data, err := GetBytesWithCookie(url, b.cookie)
	if err != nil {
		return nil, b.Err("get timeline %s %v", url, err)
	}
	if bytes.Contains(data, []byte(`document.write("请先登录！");`)) {
		return nil, bilibiliLoginError
	}

	root, err := hns.ParseBytes(data)
	if err != nil {
		return nil, err
	}

	var image, msgType, link, title, desc string
	var id int
	var walkErr error
	root.Walk(hns.Css("li", hns.Multi(
		hns.Css("img.preview", hns.Do(func(n *hns.Node) {
			image = n.Attr["src"]
		})),
		hns.Css("div.t", hns.Do(func(n *hns.Node) {
			msgType = n.Text
		})),
		hns.Css("a.vt", hns.Do(func(n *hns.Node) {
			title = n.Text
			link = n.Attr["href"]
		})),
		hns.Css("div.content", hns.Do(func(n *hns.Node) {
			desc = strings.TrimSpace(n.Text)
		})),
		hns.Do(func(node *hns.Node) {
			id, err = strconv.Atoi(regexp.MustCompile(`av([0-9]+)`).FindStringSubmatch(link)[1])
			if err != nil {
				walkErr = b.Err("link without av id %s at %s", link, url)
				return
			}
			ret = append(ret, &BilibiliEntry{
				Id:          id,
				Link:        link,
				Title:       title,
				Image:       image,
				Description: desc,
			})
		}),
	)))
	if walkErr != nil {
		return nil, walkErr
	}

	return
}

func (b *BilibiliCollector) CollectNewest(urlPattern string, page int) (ret []Entry, err error) {
	// get content
	url := s(urlPattern, page)
	resp, err := GetWithCookie(url, b.cookie)
	if err != nil {
		return nil, b.Err("get newest page %s %v", url, err)
	}
	defer resp.Body.Close()
	root, err := hns.Parse(resp.Body)
	if err != nil {
		return nil, b.Err("parse html %v", err)
	}

	var link, title, image string
	var id int
	var walkErr error
	root.Walk(hns.Css("ul.vd_list li", hns.Multi(
		hns.Css("a.title", hns.Do(func(n *hns.Node) {
			link = "http://www.bilibili.com" + n.Attr["href"]
			title = n.Text
		})),
		hns.Css("a.preview img", hns.Do(func(n *hns.Node) {
			image = n.Attr["src"]
		})),
		hns.Do(func(node *hns.Node) {
			id, err = strconv.Atoi(regexp.MustCompile(`av([0-9]+)`).FindStringSubmatch(link)[1])
			if err != nil {
				walkErr = b.Err("link without av id %s at %s", link, url)
				return
			}
			ret = append(ret, &BilibiliEntry{
				Id:    id,
				Link:  link,
				Title: title,
				Image: image,
			})
		}),
	)))
	if walkErr != nil {
		return nil, walkErr
	}

	return
}

type BilibiliEntry struct {
	Id          int
	Link        string
	Title       string
	Image       string
	Description string
}

func (e *BilibiliEntry) GetKey() string {
	return fmt.Sprintf("bilibili %d", e.Id)
}

func (e *BilibiliEntry) ToRssItem() RssItem {
	return RssItem{
		Title:  e.Title,
		Link:   e.Link,
		Desc:   e.Description,
		Author: "Bilibili",
	}
}

var bilibiliHtmlTemplate = template.Must(template.New("bilibili").Parse(`
<h2>Bilibili</h2>
<p>{{.Title}}</p>
<p>{{.Description}}</p>
`))

func (e *BilibiliEntry) ToHtml() string {
	buf := new(bytes.Buffer)
	err := bilibiliHtmlTemplate.Execute(buf, e)
	if err != nil {
		return s("render error %v", err)
	}
	return string(buf.Bytes())
}

func (b *BilibiliCollector) Login() error {
	// get cookie
	uri := "https://secure.bilibili.com/login"
	cookie, err := GetCookieStr(uri)
	if err != nil {
		return b.Err("get %s %v", uri, err)
	}

	// get captcha
	uri = "https://secure.bilibili.com/captcha?r=0.43428630707785487"
	resp, err := GetWithCookie(uri, cookie)
	if err != nil {
		return b.Err("get %s %v", uri, err)
	}
	f, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		return b.Err("create temp file %v", err)
	}
	io.Copy(f, resp.Body)
	f.Close()
	resp.Body.Close()
	exec.Command("qiv", "-fm", f.Name()).Run()
	var code string
	p("input captcha code: ")
	fmt.Scanf("%s", &code)

	// get username and password
	var user, password string
	if res := b.kv.KvGet("bilibili-username"); res != nil {
		user = res.(string)
	}
	if res := b.kv.KvGet("bilibili-password"); res != nil {
		password = res.(string)
	}
	if user == "" || password == "" {
		p("input username: ")
		fmt.Scanf("%s", &user)
		p("input password: ")
		fmt.Scanf("%s", &password)
		b.kv.KvSet("bilibili-username", user)
		b.kv.KvSet("bilibili-password", password)
	}

	// post login form
	uri = "https://secure.bilibili.com/login"
	resp, err = PostFormWithCookie(uri, url.Values{
		"act":      {"login"},
		"gourl":    {"http://www.bilibili.com/account/dynamic"},
		"keeptime": {"2592000"},
		"userid":   {user},
		"pwd":      {password},
		"vdcode":   {code},
	}, cookie)
	if err != nil {
		return b.Err("post %s %v", uri, err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return b.Err("read body %s %v", uri, err)
	}
	if !bytes.Contains(content, []byte(`document.write("成功登录，现在转向指定页面...");`)) {
		return b.Err("login fail")
	}
	cookie = RespGetCookieStr(resp)
	p("bilibili cookie => %s\n", cookie)
	b.kv.KvSet("bilibili-cookie", cookie)
	b.cookie = cookie

	return nil
}
