package main

import (
	"log"

	pyqt "github.com/reusee/go-pyqt5"
)

func NewReader(db *Database) {
	qt, err := pyqt.New(`
from PyQt5.QtCore import Qt

from PyQt5.QtWidgets import QWidget
class Win(QWidget):
	def __init__(self, **args):
		super().__init__(**args)
	def keyPressEvent(self, ev):
		Emit("key", ev.key())
win = Win(styleSheet = "background-color: black;")

from PyQt5.QtWidgets import QHBoxLayout
layout = QHBoxLayout()
win.setLayout(layout)
layout.addStretch()

from PyQt5.QtWebKitWidgets import QWebView
view = QWebView()
layout.addWidget(view)
Connect('set-html', lambda s: view.setHtml(s))

layout.addStretch()

win.showMaximized()
	`)
	if err != nil {
		log.Fatal(err)
	}

	keys := make(chan rune)
	qt.Connect("key", func(key float64) {
		select {
		case keys <- rune(key):
		default:
		}
	})

	css := `<style>
* {
	color: white;
	background-color: black;
	font-family: Terminus;
	font-size: 24px;
}
a {
	color: #0099CC;
}
h2 {
	font-size: 36px;
	color: #0099CC;
}
</style>`

	i := len(db.Entries) - 1
	for i >= 0 {
		item := db.Entries[i]
		html := item.Entry.ToHtml()
		qt.Emit("set-html", css+html)
		key := <-keys
		switch key {
		case 'J':
			i -= 1
		case 'K':
			if i < len(db.Entries)-1 {
				i += 1
			}
		}
	}
}
