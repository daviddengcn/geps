package gp

import (
	"bytes"
	"fmt"
	"github.com/daviddengcn/go-villa"
	"log"
	"strconv"
	"strings"
	"unicode"
)

const (
	ct_LOCAL = iota
	ct_GLOBAL
	ct_EVAL
	ct_HTML
	ct_IGNORE
)

type GspPart interface {
	goSource() string
}

type HtmlGspPart string

func (p HtmlGspPart) goSource() string {
	return fmt.Sprintf("__print__(%q)\n", p)
}

type CodeGspPart string

func (p CodeGspPart) goSource() string {
	return string(p) + "\n"
}

type EvalGspPart string

func (p EvalGspPart) goSource() string {
	return fmt.Sprintf("__print__(%s)\n", p)
}

type GspParts struct {
	local                       []GspPart
	imports, included, required villa.StrSet
}

func newGspParts() *GspParts {
	return &GspParts{}
}

func (ps *GspParts) addHtml(src string) {
	if len(src) == 0 {
		return
	} // if
	ps.local = append(ps.local, HtmlGspPart(src))
}

func sepGlobal(src string) (cmd, remain string) {
	var i int
	var r rune

	for i, r = range src {
		if !unicode.IsLetter(r) {
			break
		}
	}

	return src[:i], src[i:]
}

func (p *Parser) include(ps *GspParts, path villa.Path) error {
	src, err := p.Load(path)

	if err != nil {
		return err
	}

	p.parse(src, ps)

	return nil
}

func (p *Parser) addCode(ps *GspParts, src string, codeType int) {
	switch codeType {
	case ct_LOCAL:
		ps.local = append(ps.local, CodeGspPart(src))

	case ct_EVAL:
		ps.local = append(ps.local, EvalGspPart(strings.TrimSpace(src)))
		
	case ct_IGNORE:
		// Do nothing

	case ct_GLOBAL:
		//            ps.global = append(ps.global, CodeGspPart(src))
		cmd, src := sepGlobal(src)
		switch cmd {
		case "import":
			imports := strings.Split(src, ",")
			for _, imp := range imports {
				impstr, err := strconv.Unquote(strings.TrimSpace(imp))
				if err == nil {
					ps.imports.Put(impstr)
				} else {
					log.Printf("import %s error: %v", imp, err)
				}
			}

		case "include":
			imp := strings.TrimSpace(src)
			inc, err := strconv.Unquote(imp)
			if err == nil {
				if !ps.included.In(inc) {
					ps.included.Put(inc)
					err = p.include(ps, villa.Path(inc))
					if err != nil {
						log.Printf("include %s failed: %v", inc, err)
					}
					ps.included.Delete(inc)
				}
			} else {
				log.Printf("include %s error: %v", imp, err)
			}

		case "require":
			imp := strings.TrimSpace(src)
			inc, err := strconv.Unquote(imp)
			if err == nil {
				if !ps.required.In(inc) {
					ps.required.Put(inc)
					err = p.include(ps, villa.Path(inc))
					if err != nil {
						log.Printf("require %s failed: %v", inc, err)
					}
				}
			} else {
				log.Printf("require %s error: %v", imp, err)
			}
		}
	}
}

func (ps GspParts) GoSource() (src string) {
	src = "package main\n"

	for imp := range ps.imports {
		src += "import " + strconv.Quote(imp) + "\n"
	}

	src += "func __process__() {\n"

	for _, p := range ps.local {
		src += "    " + p.goSource()
	}

	src += "}\n"

	return src
}

func (p *Parser) parse(src string, parts *GspParts) (err error) {
	/*
			Status Transform
			
			             .-- R(eady) <---.
			         (<)/                 |
			           V                  |
			           C0                 |(>)
			        (%)|   ___(%)____     |
			           V  /          V    |
			tp=LOCAL   C1 ---> C2 ---> C3-'
			           | \        (%)
			           |  \(=)
			tp=EVAL    |   `-> C2 ...
			           |\(!)
			tp=GLOBAL  | `---> C2 ...
			            \(#)
			tp=IGNORED   `---> C2 ...
	*/
	const (
		R = iota
		C0
		C1
		C2
		C3
	)
	status, tp := R, ct_GLOBAL
	var source bytes.Buffer
	for _, r := range src {
		switch status {
		case R:
			switch r {
			case '<':
				status = C0

			default:
				source.WriteRune(r)
			}
		case C0:
			switch r {
			case '%':
				status, tp = C1, ct_LOCAL
				parts.addHtml(source.String())
				source.Reset()

			default:
				status = R
				source.WriteRune('<') // the < causing R->C0
				source.WriteRune(r)
			}

		case C1:
			switch r {
			case '=':
				status, tp = C2, ct_EVAL

			case '!':
				status, tp = C2, ct_GLOBAL

			case '#':
				status, tp = C2, ct_IGNORE
				
			case '%':
				status = C3

			default:
				status = C2
				source.WriteRune(r)
			}

		case C2:
			switch r {
			case '%':
				status = C3

			default:
				source.WriteRune(r)
			}

		case C3:
			switch r {
			case '>':
				status = R
				p.addCode(parts, source.String(), tp)
				source.Reset()

			default:
				status = C2
				source.WriteRune('%') // the % causing C2->C3
				source.WriteRune(r)
			}
		} // switch status
	} // for r

	switch status {
	case R:
		if source.Len() > 0 {
			parts.addHtml(source.String())
		}
	default:
		log.Println("Unclosed tag")
	}
	return nil
}

type LoadFunc func(path villa.Path) (string, error)

type Parser struct {
	Load LoadFunc
}

func NewParser(load LoadFunc) *Parser {
	return &Parser{Load: load}
}

func (p *Parser) Parse(src string) (parts *GspParts, err error) {
	parts = newGspParts()
	err = p.parse(src, parts)
	if err != nil {
		return nil, err
	}

	return parts, nil
}
