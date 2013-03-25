package utils

import (
	"bytes"
	"io"
	"strings"
)

var (
	htmlQuot    = []byte("&#34;") // shorter than "&quot;"
	htmlApos    = []byte("&#39;") // shorter than "&apos;" and apos was not in HTML until HTML5
	htmlAmp     = []byte("&amp;")
	htmlLt      = []byte("&lt;")
	htmlGt      = []byte("&gt;")
	htmlSpace   = []byte("&nbsp;")
	htmlNewLine = []byte("<br/>")
)

// HTMLEscape writes to w the escaped HTML equivalent of the plain text data b.
func HTMLEscape(w io.Writer, b []byte) {
	last := 0
	lastb := byte(0)
	for i, c := range b {
		var html []byte
		switch c {
		case '"':
			html = htmlQuot
		case '\'':
			html = htmlApos
		case '&':
			html = htmlAmp
		case '<':
			html = htmlLt
		case '>':
			html = htmlGt
		case ' ':
			if lastb > ' ' && lastb != '>' {
				lastb = ' '
				continue
			}
			html = htmlSpace
		case '\n':
			html = htmlNewLine
		default:
			lastb = c
			continue
		}
		w.Write(b[last:i])
		w.Write(html)
		last = i + 1
		lastb = html[len(html)-1]
	}
	w.Write(b[last:])
}

func HTMLEscapeString(s string) string {
	// Avoid allocation if we can.
	if strings.IndexAny(s, "'\"&<> \n") < 0 {
		return s
	}
	var b bytes.Buffer
	HTMLEscape(&b, []byte(s))
	return b.String()
}
