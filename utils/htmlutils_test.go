package utils

import(
	"testing"
)

func TestHTMLEscapeString(t *testing.T) {
	cases := []struct {in, out string} {
		{" ", "&nbsp;"},
		{"\n", "<br/>"},
	}
	
	for _, c := range cases {
		act := HTMLEscapeString(c.in)
		if act != c.out {
			t.Errorf("HTMLEscapeString(%s): expected %s, but got %s", c.in, c.out, act)
		}
	}
}