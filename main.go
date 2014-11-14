package main

import (
	"log"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/reusee/gobchest"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Entry interface {
	GetKey() string
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

	// init gobchest client
	client, err := gobchest.NewClient("localhost:2800")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// collect
	collect := func() {
		for _, f := range []func() (Collector, error){
			func() (Collector, error) { return NewV2exCollector() },
			//func() (Collector, error) { return NewZhihuCollector(db) },
			func() (Collector, error) { return NewBilibiliCollector(db) },
			func() (Collector, error) {
				return NewDoubanCollector(NewOAuthTokenCache(client, "infobox.douban.oauthtoken"))
			},
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

	for {
		collect()
		time.Sleep(time.Minute * 5)
	}
}
