package main

import (
	"bytes"
	"encoding/gob"
	"encoding/xml"
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
)

func init() {
	gob.Register(new(BilibiliEntry))
}

type BilibiliCollector struct {
	client *Client
}

func NewBilibiliCollector(client *Client) (*BilibiliCollector, error) {
	return &BilibiliCollector{
		client: client,
	}, nil
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
		sem := make(chan bool, 10)
		wg := new(sync.WaitGroup)
		wg.Add(maxPage)
		lock := new(sync.Mutex)
		errors := make([]error, 0, maxPage)
		for page := 1; page <= maxPage; page++ {
			go func(page int) {
				defer wg.Done()
				sem <- true
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
						return nil, Err("bilibili auth error")
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
	data, err := b.client.GetBytes(url, nil)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(data, []byte(`document.write("请先登录！");`)) {
		return nil, bilibiliLoginError
	}
	data, err = tidyHtml(data)
	if err != nil {
		return nil, err
	}

	// parse
	structure := struct {
		Body struct {
			Lis []struct {
				Divs []struct {
					Class string `xml:"class,attr"`
					A     []struct {
						Href string `xml:"href,attr"`
						Card string `xml:"card,attr"`
						Img  struct {
							Src string `xml:"src,attr"`
						} `xml:"img"`
						Text string `xml:",chardata"`
					} `xml:"a"`
					Text string `xml:",chardata"`
				} `xml:"div"`
				Tags []struct {
					A struct {
						Href string `xml:"href,attr"`
						Text string `xml:",chardata"`
					} `xml:"a"`
				} `xml:"ul>li"`
			} `xml:"li"`
		} `xml:"body"`
	}{}
	err = xml.Unmarshal(data, &structure)
	if err != nil {
		return nil, err
	}

	// collect
loop_lis:
	for _, li := range structure.Body.Lis {
		var link, title, image, desc string
		var id int
		image = li.Divs[0].A[0].Img.Src
		msgType := strings.TrimSpace(li.Divs[1].Text)
		switch msgType {
		case "上传了新视频":
			link = li.Divs[0].A[0].Href
			title = li.Divs[1].A[1].Text
			desc = strings.TrimSpace(li.Divs[2].Text)
		case "专题 　添加了新的视频", "专题 　添加了新的新番":
			link = "http://www.bilibili.com" + li.Divs[1].A[1].Href
			title = li.Divs[1].A[1].Text
			desc = strings.TrimSpace(li.Divs[2].Text)
		case "专题 　添加了新的专题":
			continue loop_lis
		default:
			return nil, Err("unknown message type %s", msgType)
		}
		id, err = strconv.Atoi(regexp.MustCompile(`av([0-9]+)`).FindStringSubmatch(link)[1])
		if err != nil {
			return nil, Err("link without av id %s", link)
		}
		ret = append(ret, &BilibiliEntry{
			Id:          id,
			Link:        link,
			Title:       title,
			Image:       image,
			Description: desc,
		})
	}

	return
}

func (b *BilibiliCollector) CollectNewest(urlPattern string, page int) (ret []Entry, err error) {
	// get content
	url := s(urlPattern, page)
	data, err := b.client.GetBytes(url, nil)
	if err != nil {
		return nil, err
	}
	data, err = tidyHtml(data)
	if err != nil {
		return nil, err
	}

	// parse
	type Ul struct {
		Lis []struct {
			As []struct {
				Href string `xml:"href,attr"`
				Img  struct {
					Src string `xml:"src,attr"`
				} `xml:"img"`
				Text string `xml:",chardata"`
			} `xml:"a"`
		} `xml:"li"`
		Class string `xml:"class,attr"`
	}
	type Div struct {
		Divs []Div `xml:"div"`
		Ul   []Ul  `xml:"ul"`
	}
	structure := struct {
		Body struct {
			Divs []Div `xml:"div"`
		} `xml:"body"`
	}{}
	err = xml.Unmarshal(data, &structure)
	if err != nil {
		return nil, err
	}
	var ul Ul
	var ok bool
	if ul, ok = find(structure, func(i interface{}) bool {
		if ul, ok := i.(Ul); ok && ul.Class == "vd_list" {
			return true
		}
		return false
	}).(Ul); !ok {
		return nil, Err("no ul found")
	}

	// collect
	for _, li := range ul.Lis {
		link := "http://www.bilibili.com" + li.As[0].Href
		id, err := strconv.Atoi(regexp.MustCompile(`av([0-9]+)`).FindStringSubmatch(link)[1])
		if err != nil {
			return nil, Err("link without av id %s", link)
		}
		title := li.As[1].Text
		image := li.As[0].Img.Src
		ret = append(ret, &BilibiliEntry{
			Id:    id,
			Link:  link,
			Title: title,
			Image: image,
		})
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
	// get captcha
	resp, err := b.client.Get("https://secure.bilibili.com/captcha?r=0.43428630707785487")
	if err != nil {
		return err
	}
	f, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		return err
	}
	io.Copy(f, resp.Body)
	f.Close()
	resp.Body.Close()
	exec.Command("qiv", "-fm", f.Name()).Run()
	var code string
	p("input captcha code: ")
	fmt.Scanf("%s", &code)
	var user, password string
	p("input username: ")
	fmt.Scanf("%s", &user)
	p("input password: ")
	fmt.Scanf("%s", &password)

	// post login form
	resp, err = b.client.PostForm("https://secure.bilibili.com/login", url.Values{
		"act":      {"login"},
		"gourl":    {"http://www.bilibili.com/account/dynamic"},
		"keeptime": {"2592000"},
		"userid":   {user},
		"pwd":      {password},
		"vdcode":   {code},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
