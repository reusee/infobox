package main

import (
	"bytes"

	"code.google.com/p/go.net/html"
)

func tidyHtml(input []byte) ([]byte, error) {
	// tidy
	nodes, err := html.ParseFragment(bytes.NewReader(input), nil)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	for _, node := range nodes {
		err = html.Render(buf, node)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
