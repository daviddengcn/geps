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

	fmt.Println(parts.GoSource())

	//	str, err := strconv.Unquote("\"abc\\\"\\\"abc\"")
	//	fmt.Println("strconv", "|" + str + "|", err)
}

func TestParser2(t *testing.T) {
	p := NewParser(nil)
	src := "<% \"%v\" %>"
	parts, err := p.Parse(src)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if len(parts.local) != 1 {
		t.Errorf("Expected 1 local line")
	}

	if parts.local[0].(CodeGspPart) != " \"%v\" " {
		t.Errorf("Expected %s but got %s", " \"%v\" ", parts.local[0])
	}
}
