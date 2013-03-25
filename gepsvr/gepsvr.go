package main

import (
	"fmt"
	"github.com/daviddengcn/geps/utils"
	"github.com/russross/blackfriday"
	"html/template"
	"log"
	"net/http"
	"os"
)

// map from path to HandlerFunc
var processors map[string]http.HandlerFunc = map[string]http.HandlerFunc{}

// registerPath registers a path-HandlerFunc pair in processors
func registerPath(path string, f http.HandlerFunc) {
	log.Println("Register path:", path)
	processors[path] = f
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if proc, ok := processors[path]; ok {
		proc(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func main() {
	host := ":8081"
	if len(os.Args) > 1 {
		host = os.Args[1]
	}
	http.HandleFunc("/", handler)
	log.Println("Back server listening at", host)
	http.ListenAndServe(host, nil)
}

// for gep files
func __print__(response http.ResponseWriter, s interface{}) {
	response.Write([]byte(fmt.Sprint(s)))
}

/* <html>$text</html> */
func Html(text interface{}) string {
	return utils.HTMLEscapeString(fmt.Sprint(text))
}

/* <input attr='$text'> <pre>$text</pre> <textarea>$text</textarea>*/
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
