package utils

import (
	"testing"
)

func TestHTMLEscapeString(t *testing.T) {
	cases := []struct{ in, out string }{
		{" ", "&nbsp;"},
		{"  ", "&nbsp; "},
		{"   ", "&nbsp; &nbsp;"},
		{"    ", "&nbsp; &nbsp; "},
		{"\n a", "<br/>&nbsp;a"},
		{"a b", "a b"},
		{"\n", "<br/>"},
	}

	for _, c := range cases {
		act := HTMLEscapeString(c.in)
		if act != c.out {
			t.Errorf("HTMLEscapeString(%q): expected %q, but got %q", c.in, c.out, act)
		}
	}
}
