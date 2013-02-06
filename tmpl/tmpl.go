package main

import(
	"fmt"
	"html"
	"html/template"
	"strings"
)

func __print__(s interface{}) {
    fmt.Print(s)
}

/* <html>$text</html> */
func Html(text interface{}) string {
	return strings.Replace(html.EscapeString(fmt.Sprint(text)), "\n", "<br>", -1)
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

func main() {
	fmt.Println("Content-Type: text/html; charset=UTF-8")
	fmt.Println()
    __process__()
}