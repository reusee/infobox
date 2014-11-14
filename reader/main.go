package main

import (
	"encoding/gob"
	"fmt"
	"log"

	infobox "../"
	"github.com/reusee/gobchest"
)

func init() {
	gob.Register([]*infobox.Item{})
}

var (
	pt = fmt.Printf
)

func main() {
	client, err := gobchest.NewClient("localhost:2800")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	v, err := client.Get("infobox.entries")
	if err != nil {
		log.Fatal(err)
	}
	entries := v.([]*infobox.Item)
	pt("%d entries\n", len(entries))
}
