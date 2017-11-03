package indexer

import (
	"testing"
)

var strs = [][]string{
	{`moo`, `moo`, `^moo$`},
	{"*", `^.*$`},
	{"{moo,goo}", `^(moo|goo)$`},
	{"(moo|goo)", `^(moo|goo)$`},
	{"{m*,goo}", `^(m.*|goo)$`},
}

func Test_Utils__NeedRegex(t *testing.T) {
	for _, s := range strs {
		if s[0] == s[1] && needRegex(s[0]) {
			t.Fatalf("Should not have a regex: %s", s[0])
		} else if s[0] != s[1] && !needRegex(s[0]) {
			t.Fatalf("Should have a regex: %s: %s: %s", s[0], s[1], fullRegString(s[0]))
		}
	}
}

func Test_Utils__RegifyKey(t *testing.T) {
	for _, s := range strs {
		rVal := s[1]
		if len(s) == 3 {
			rVal = s[2]
		}
		if fullRegString(s[0]) != rVal {
			t.Fatalf("Should have a regex: %s: %s: gotten: %s", s[0], rVal, fullRegString(s[0]))
		}
		_, err := regifyKey(s[0])
		if err != nil {
			t.Fatalf("Regex failed: %s: %v", s[0], err)
		}
	}
}

func Test_Utils__SplitMetric(t *testing.T) {

	loStrings := []string{
		"moo.goo.org,loo.poo.org",
		"{moo,goo}.goo.org,loo.{joo,hoo}.org",
		"(moo|goo).goo.org,loo.(moo|goo).org",
	}

	for _, s := range loStrings {
		if len(splitMetricsPath(s)) != 2 {
			t.Fatalf("Split failed: %s: %v", s, splitMetricsPath(s))
		}
	}
	l1Strings := []string{
		"moo.goo.org",
		"{moo,goo}.goo.org",
		"(moo|goo).goo.org",
	}

	for _, s := range l1Strings {
		spl := splitMetricsPath(s)
		if len(spl) != 1 && spl[0] != s {
			t.Fatalf("Split failed: %s: %v", s, splitMetricsPath(s))
		}
	}
}
