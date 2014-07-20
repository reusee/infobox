package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var p = fmt.Printf

func Err(format string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(format, args...))
}

type Entry interface {
	Key() string
}

type Collector interface {
	Collect() ([]Entry, error)
}

func main() {
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

	// collect
	for _, collector := range []Collector{
		NewBilibiliCollector(client), // bilibili
	} {
		entries, err := collector.Collect()
		if err != nil {
			log.Fatal(err)
		}
		db.AddEntries(entries)
	}

	// save
	if err := db.Save(); err != nil {
		log.Fatal(err)
	}

	p("total %d entries\n", len(db.Entries))
}
