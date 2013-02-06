package main

import(
	"testing"
	"fmt"
)

func TestEscapeFuncs(t *testing.T) {
	src := "\n+=?\"'&:/ ~!#<>|"
	htmlSrc := Html(src)
	fmt.Println("Html:", htmlSrc)
	htmlValue := Value(src)
	fmt.Println("Value:", htmlValue)
	htmlQuery := Query(src)
	fmt.Println("Query:", htmlQuery)
	htmlJS := JS(src)
	fmt.Println("JS:", htmlJS)
}