package main

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code.google.com/p/goauth2/oauth"
)

type Item struct {
	Entry   Entry
	AddTime time.Time
	Read    bool
}

type Database struct {
	Entries     []*Item
	Set         map[string]struct{}
	Jar         *Jar
	dbPath      string
	OAuthTokens map[string]*oauth.Token
	portLock    net.Listener
	Kv          map[string]interface{}
	kvLock      sync.Mutex
}

func NewDatabase(dbDir string) (*Database, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:53892")
	if err != nil {
		return nil, Err("database lock fail")
	} else {
		p("db locked.\n")
	}
	dbPath := filepath.Join(dbDir, "db")
	f, err := os.Open(dbPath)
	if err != nil { // no file or error, create new database
		database := &Database{
			Set:         make(map[string]struct{}),
			Jar:         NewJar(),
			dbPath:      dbPath,
			OAuthTokens: make(map[string]*oauth.Token),
			Kv:          make(map[string]interface{}),
		}
		p("new database created.\n")
		return database, nil
	}
	defer f.Close()
	var database Database
	err = gob.NewDecoder(f).Decode(&database)
	if err != nil {
		return nil, Err("gob decode %v", err)
	}
	database.dbPath = dbPath
	if database.OAuthTokens == nil {
		database.OAuthTokens = make(map[string]*oauth.Token)
	}
	if database.Kv == nil {
		database.Kv = make(map[string]interface{})
	}
	database.Jar.init()
	database.portLock = ln
	p("database loaded.\n")
	return &database, nil
}

func (d *Database) Save() error {
	if len(d.dbPath) == 0 {
		panic("no dbPath")
	}
	// save database
	tmpPath := d.dbPath + fmt.Sprintf(".%d", rand.Uint32())
	tmpF, err := os.Create(tmpPath)
	if err != nil {
		return Err("temp db %v", err)
	}
	err = gob.NewEncoder(tmpF).Encode(d)
	if err != nil {
		return Err("encode db %v", err)
	}
	tmpF.Close()
	err = os.Rename(tmpPath, d.dbPath)
	if err != nil {
		return Err("rename temp db %v", err)
	}
	return nil
}

func (d *Database) AddEntries(entries []Entry) {
	for _, entry := range entries {
		key := entry.GetKey()
		if _, ok := d.Set[key]; !ok {
			d.Entries = append(d.Entries, &Item{
				Entry:   entry,
				AddTime: time.Now(),
			})
			d.Set[key] = struct{}{}
		}
	}
}

func (d *Database) KvGet(key string) interface{} {
	d.kvLock.Lock()
	defer d.kvLock.Unlock()
	return d.Kv[key]
}

func (d *Database) KvSet(key string, value interface{}) {
	d.kvLock.Lock()
	defer d.kvLock.Unlock()
	d.Kv[key] = value
}
