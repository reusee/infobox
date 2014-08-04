package main

import (
	"encoding/gob"
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"github.com/reusee/hns"
)

func init() {
	gob.Register(new(V2exEntry))
}

type V2exEntry struct {
	Id    int
	Title string
}

type V2exCollector struct {
	*ErrorHost
}

func NewV2exCollector() (*V2exCollector, error) {
	v := &V2exCollector{
		ErrorHost: NewErrorHost("V2ex"),
	}
	return v, nil
}

func (v *V2exCollector) Collect() (ret []Entry, err error) {
	nodes := []string{
		"share",
	}
	maxPage := 10
	var uris []string
	for _, node := range nodes {
		for page := 1; page <= maxPage; page++ {
			uris = append(uris, fmt.Sprintf("http://v2ex.com/go/%s?p=%d", node, page))
		}
	}

	sem := make(chan bool, 8)
	wg := new(sync.WaitGroup)
	wg.Add(len(uris))
	lock := new(sync.Mutex)
	errors := make([]error, 0, len(uris))
	for i, uri := range uris {
		go func(i int, uri string) {
			defer wg.Done()
			sem <- true
			entries, err := v.CollectPage(uri)
			lock.Lock()
			ret = append(ret, entries...)
			errors = append(errors, err)
			lock.Unlock()
			<-sem
		}(i, uri)
	}
	wg.Wait()

	for _, e := range errors {
		if e != nil {
			return nil, e
		}
	}

	fmt.Printf("collect %d entries from V2ex\n", len(ret))
	return
}

var v2exPidPattern = regexp.MustCompile(`/t/([0-9]+)`)

func (v *V2exCollector) CollectPage(uri string) (ret []Entry, err error) {
	resp, err := Get(uri)
	if err != nil {
		return nil, v.Err("get %s error: %v", uri, err)
	}
	defer resp.Body.Close()
	root, err := hns.Parse(resp.Body)
	if err != nil {
		return nil, v.Err("parse html %s: %v", uri, err)
	}

	var walkError error
	root.Walk(hns.Css("div.cell span.item_title a", hns.Do(func(n *hns.Node) {
		id, err := strconv.Atoi(v2exPidPattern.FindStringSubmatch(n.Attr["href"])[1])
		if err != nil {
			walkError = v.Err("no post id: %s", uri)
			return
		}
		ret = append(ret, &V2exEntry{
			Id:    id,
			Title: n.Text,
		})
	})))
	if walkError != nil {
		return nil, walkError
	}

	return
}

func (v *V2exEntry) GetKey() string {
	return fmt.Sprintf("v2ex %d", v.Id)
}

func (v *V2exEntry) ToHtml() string {
	return "" //TODO
}

func (v *V2exEntry) ToRssItem() RssItem {
	return RssItem{
		Title:  v.Title,
		Link:   fmt.Sprintf("http://v2ex.com/t/%d", v.Id),
		Author: "V2ex",
	}
}
