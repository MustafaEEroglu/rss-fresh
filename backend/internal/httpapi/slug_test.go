package httpapi

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"AI News":               "ai-news",
		"  Hello World  ":       "hello-world",
		"Tech & Engineering":    "tech-engineering",
		"---weird---":           "weird",
		"Türkçe Başlık":         "t-rk-e-ba-l-k", // non-ASCII collapses; OK for now
		"":                      "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
