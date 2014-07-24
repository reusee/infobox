package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Entry interface {
	GetKey() string
	ToRssItem() RssItem
	ToHtml() string
}

type Collector interface {
	Collect() ([]Entry, error)
}

var (
	InteractiveMode bool
)

func main() {
	// parse args
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "inter") {
			InteractiveMode = true
		} else {
			log.Fatalf("unknown command line argument %s", arg)
		}
	}

	// init database
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dbDir := filepath.Join(user.HomeDir, ".infobox")
	_, err = os.Stat(dbDir)
	if err != nil {
		err = os.Mkdir(dbDir, 0777)
		if err != nil {
			log.Fatal("create db dir %v", err)
		}
	}
	db, err := NewDatabase(dbDir)
	if err != nil {
		log.Fatal(err)
	}

	// init http client
	client := NewClient(db.Jar)
	_ = client

	// collect
	var collectors []Collector
	for _, f := range []func() (Collector, error){
		func() (Collector, error) { return NewDoubanCollector(db.TokenCache("douban")) },
		func() (Collector, error) { return NewZhihuCollector(client, db) },
		func() (Collector, error) { return NewBilibiliCollector(client) },
	} {
		collector, err := f()
		if err != nil {
			log.Fatal(err)
		}
		collectors = append(collectors, collector)
	}
	collect := func() {
		for _, f := range []func() (Collector, error){
			func() (Collector, error) { return NewDoubanCollector(db.TokenCache("douban")) },
			func() (Collector, error) { return NewZhihuCollector(client, db) },
			func() (Collector, error) { return NewBilibiliCollector(client) },
		} {
			collector, err := f()
			if err != nil {
				log.Fatal(err)
			}
			entries, err := collector.Collect()
			if err != nil {
				// insert error report entry
				db.AddEntries([]Entry{
					NewErrorEntry(err),
				})
				p("%v\n", err)
				continue
			}
			db.AddEntries(entries)
		}
		// save
		if err := db.Save(); err != nil {
			log.Fatal(err)
		}
	}

	go func() {
		for {
			collect()
			time.Sleep(time.Minute * 2)
		}
	}()

	//go NewReader(db)

	// rss server
	p("start rss server.\n")
	http.HandleFunc("/rss", db.RssHandler)
	err = http.ListenAndServe("127.0.0.1:38888", nil)
	if err != nil {
		log.Fatal(err)
	}

}
