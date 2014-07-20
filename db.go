package main

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

type Database struct {
	Entries map[string]Entry
	Jar     *Jar
	dbPath  string
}

func NewDatabase(dbDir string) (*Database, error) {
	dbPath := filepath.Join(dbDir, "db")
	f, err := os.Open(dbPath)
	if err != nil { // no file or error, create new database
		database := &Database{
			Entries: make(map[string]Entry),
			Jar:     NewJar(),
			dbPath:  dbPath,
		}
		p("new database created.\n")
		return database, nil
	}
	var database Database
	err = gob.NewDecoder(f).Decode(&database)
	if err != nil {
		return nil, Err("gob decode %v", err)
	}
	database.dbPath = dbPath
	f.Close()
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
		key := entry.Key()
		if _, ok := d.Entries[key]; !ok {
			d.Entries[key] = entry
		}
	}
}
