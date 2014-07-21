package main

import (
	"encoding/xml"
	"net/http"
)

type RssItem struct {
	Title  string `xml:"title"`
	Link   string `xml:"link"`
	Desc   string `xml:"description"`
	Author string `xml:"author"`
	Guid   string `xml:"guid"`
	Pub    string `xml:"pubDate"`
}

func (d *Database) RssHandler(w http.ResponseWriter, req *http.Request) {
	structure := struct {
		XMLName xml.Name `xml:"rss"`
		Version string   `xml:"version,attr"`
		Channel struct {
			Title string    `xml:"title"`
			Link  string    `xml:"link"`
			Desc  string    `xml:"description"`
			Items []RssItem `xml:"item"`
		} `xml:"channel"`
	}{}

	structure.Version = "2.0"
	structure.Channel.Title = "Infobox"
	structure.Channel.Link = "http://foo"

	for i := len(d.Entries) - 1; i >= 0; i-- {
		entry := d.Entries[i]
		item := entry.Entry.ToRssItem()
		item.Pub = s("%v", entry.AddTime)
		structure.Channel.Items = append(structure.Channel.Items, item)
		if len(structure.Channel.Items) > 10000 {
			break
		}
	}

	out, err := xml.MarshalIndent(structure, "", "  ")
	if err != nil {
		return
	}
	w.Write(out)
}
