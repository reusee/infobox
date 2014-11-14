package main

import (
	"encoding/gob"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os/user"
	"path/filepath"

	infobox "../"
	"github.com/reusee/gobchest"
)

func init() {
	go http.ListenAndServe("localhost:2801", nil)

	gob.Register(new(infobox.Item))
	gob.Register(new([]*infobox.Item))
}

func main() {
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	filePath := filepath.Join(user.HomeDir, ".infobox.chest")
	server, err := gobchest.NewServer("localhost:2800", filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	<-make(chan bool)
}
