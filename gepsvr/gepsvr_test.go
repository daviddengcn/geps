package main

import (
	"testing"
)

func TestEscapeFuncs(t *testing.T) {
	src := "\n+=?\"'&:/  ~!#<>|"
	cases := []struct {
		f     func(interface{}) string
		fname string
		out   string
	}{
		{Html, "Html", `<br/>+=?&#34;&#39;&amp;:/ &nbsp;~!#&lt;&gt;|`},
		{Value, "Value", "\n+=?&#34;&#39;&amp;:/  ~!#&lt;&gt;|"},
		{Query, "Query", "%0A%2B%3D%3F%22%27%26%3A%2F++~%21%23%3C%3E%7C"},
		{Markdown, "Markdown", "<p>+=?&ldquo;&rsquo;&amp;:/  ~!#&lt;&gt;|</p>\n"},
	}

	for _, c := range cases {
		act := c.f(src)
		if act != c.out {
			t.Errorf("%s(%q): expected %q, but got %q!", c.fname, src, c.out, act)
		}
	}
}

func TestAssureExistence(t *testing.T) {
	if false {
		registerPath("", nil)
		__print__(nil, nil)
	}
}
