package main

import(
	"fmt"
	"html/template"
	"strings"
	"net/http"
	"net/http/cgi"
	"github.com/russross/blackfriday"
)

func __print__(response http.ResponseWriter, s interface{}) {
    response.Write([]byte(fmt.Sprint(s)))
}

/* <html>$text</html> */
func Html(text interface{}) string {
	return strings.Replace(template.HTMLEscapeString(fmt.Sprint(text)), "\n", "<br>", -1)
}


/* <input attr='$text'> */
func Value(text interface{}) string {
	return template.HTMLEscapeString(fmt.Sprint(text))
}

/* http://xxx.xxx/?xxx=$text */
func Query(text interface{}) string {
	return template.URLQueryEscaper(fmt.Sprint(text))
}

/* <script> s='$text' </script> */
func JS(text interface{}) string {
	return template.JSEscaper(fmt.Sprint(text))
}

// Markdown converts a markdown markup text into HTML
func Markdown(text interface{}) string {
	return string(blackfriday.MarkdownCommon([]byte(fmt.Sprint(text))))
}


func main() {
	cgi.Serve(http.HandlerFunc(__process__))
}
