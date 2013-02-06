package gp

import (
	"errors"
	"fmt"
	"github.com/daviddengcn/go-villa"
	"testing"
)

func TestParser(t *testing.T) {
	p := NewParser(func(path villa.Path) (string, error) {
		switch path {
		case "header":
			return `
			== This is the header ==
			requiring "funcs" in header
			<%!require "funcs" %><%= play() %>
			The following include will be ignored: <%= "<" %>%!include "sub1"%>
			`, nil
			
		case "footer":
			return `
			requiring "funcs" in footer
			<%!require "funcs" %><%= play() %>
			=== footer ===
			`, nil
			
		case "funcs":
			return `
	<% play := func() string {
        return "playing\n"
    } %>
			`, nil
			
		}
		return "", errors.New(("Not found: " + path).S())
	})

	src := `
<%!include "header"%>
abc<%!import "fmt", "github.com/daviddengcn/go-villa"%>"
<%!require "header"%>
	`

	parts, err := p.Parse(src)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	fmt.Println(parts.goSource())

	//	str, err := strconv.Unquote("\"abc\\\"\\\"abc\"")
	//	fmt.Println("strconv", "|" + str + "|", err)
}
