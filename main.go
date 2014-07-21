package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Err(format string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(format, args...))
}

type Entry interface {
	GetKey() string
	ToRssItem() RssItem
}

type Collector interface {
	Collect() ([]Entry, error)
}

var (
	SilenceMode bool
)

func main() {
	// parse args
	for _, arg := range os.Args[1:] {
		if arg == "silence" {
			SilenceMode = true
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

	collect := func() {
		// collect
		for _, f := range []func() (Collector, error){
			func() (Collector, error) { return NewBilibiliCollector(client) },
			func() (Collector, error) { return NewDoubanCollector(db.TokenCache("douban")) },
		} {
			collector, err := f()
			if err != nil {
				continue //TODO log
			}
			entries, err := collector.Collect()
			if err != nil {
				continue //TODO log
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
			time.Sleep(time.Minute * 3)
		}
	}()

	p("start rss server.\n")
	http.HandleFunc("/rss", db.RssHandler)
	err = http.ListenAndServe("127.0.0.1:38888", nil)
	if err != nil {
		log.Fatal(err)
	}

}
