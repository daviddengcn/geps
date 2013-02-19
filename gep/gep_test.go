package gep

import (
	"errors"
	"fmt"
	"github.com/daviddengcn/go-villa"
	"strings"
	"testing"
)

type simple map[villa.Path]string

func (s simple) Load(path villa.Path) (src string, err error) {
	src, ok := s[path]
	if !ok {
		return "", errors.New(("Not found: " + path).S())
	}

	return src, nil
}

func (s simple) Error(message string) {
	fmt.Println("Error:", message)
}

func (s simple) GenRawPart(src string) interface{} {
	return src
}

func (s simple) GenCodePart(src string) interface{} {
	return "[CODE]" + strings.TrimSpace(src) + "[/CODE]"
}

func (s simple) GenEvalPart(src string) interface{} {
	return "[EVAL]" + strings.TrimSpace(src) + "[/EVAL]"
}

func TestParser(t *testing.T) {
	f := simple{
		"header": `== This is the header == requiring "funcs" in header <%!require "funcs" %><%= play() %> The following include will be ignored: <%= "<" %>%!include "sub1"%>`,

		"footer": `requiring "funcs" in footer <%!require "funcs" %><%= play() %> === footer ===`,

		"funcs": `<% play := func() string { return "playing\n" } %>`}

	src := `<%!include "header"%> abc<%!import "fmt", "github.com/daviddengcn/go-villa"%>" <%!require "header"%>`

	res, err := Parse(f, src)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	fmt.Println("Parts:", res.Parts)
	expectedParts := []interface{}{
		`== This is the header == requiring "funcs" in header `,
		`[CODE]play := func() string { return "playing\n" }[/CODE]`,
		`[EVAL]play()[/EVAL]`,
		` The following include will be ignored: `,
		`[EVAL]"<"[/EVAL]`,
		`%!include "sub1"%>`,
		` abc`,
		`" `}
	if !res.Parts.Equals(expectedParts) {
		t.Errorf("Expected:\n%v\nbut got\n%v", expectedParts, res.Parts)
	}

	fmt.Println("Imports:", res.Imports)
	expectedImports := villa.NewStrSet("github.com/daviddengcn/go-villa", "fmt")
	if !res.Imports.Equals(expectedImports) {
		t.Errorf("Expected imports: %v, but got %v", expectedImports, res.Imports)
	}

	fmt.Println("Depends:", res.Depends)
	expectedDepends := villa.NewStrSet("header", "funcs")
	if !res.Depends.Equals(expectedDepends) {
		t.Errorf("Expected depends: %v, but got %v", expectedDepends, res.Depends)
	}

	fmt.Println("Includeonly:", res.IncludeOnly)
	if res.IncludeOnly {
		t.Errorf("Expected iuncludeonly: false")
	}
}

func TestParser_bugstatus(t *testing.T) {
	f := simple{}
	src := "<% \"%v\" %>"
	parts, err := Parse(f, src)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if len(parts.Parts) != 1 {
		t.Errorf("Expected 1 part")
	}

	if parts.Parts[0].(string) != `[CODE]"%v"[/CODE]` {
		t.Errorf("Expected %s but got %s", `[CODE]"%v"[/CODE]`, parts.Parts[0])
	}
}

func TestParser_ignore(t *testing.T) {
	f := simple{}
	src := "<%# \"%v\" %>"
	parts, err := Parse(f, src)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if len(parts.Parts) != 0 {
		t.Errorf("Expected 0 part, but got %d lines", len(parts.Parts))
	}
}

func TestParser_includeonly(t *testing.T) {
	f := simple{"file": "<%!includeonly%>"}

	src := "<%!includeonly%>"
	parts, err := Parse(f, src)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if parts.IncludeOnly != true {
		t.Errorf("IncludeOnly is expected to be true")
	}

	src = `<%!include "file"%>`
	parts, err = Parse(f, src)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if parts.IncludeOnly != false {
		t.Errorf("IncludeOnly is expected to be false")
	}
}
