package main

import (
	"encoding/gob"
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

type Item struct {
	Entry   Entry
	AddTime time.Time
	Read    bool
}

func init() {
	gob.Register(new(Item))
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

	addEntry := func(entry Entry) {
		key := entry.GetKey()
		if !client.SetExists("infobox.entry-keys", key) {
			err := client.ListAppend("infobox.entries", &Item{
				Entry:   entry,
				AddTime: time.Now(),
			})
			if err != nil {
				log.Fatal(err)
			}
			client.SetAdd("infobox.entry-keys", key)
		}
	}

	// collect
	collect := func() {
		for _, f := range []func() (Collector, error){
			func() (Collector, error) { return NewV2exCollector() },
			//func() (Collector, error) { return NewZhihuCollector(NewKvStore(client)) },
			func() (Collector, error) { return NewBilibiliCollector(NewKvStore(client)) },
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
				addEntry(NewErrorEntry(err))
				continue
			}
			db.AddEntries(entries)
			for _, e := range entries {
				addEntry(e)
			}
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
