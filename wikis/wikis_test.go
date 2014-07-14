package wikis

import (
	"testing"
)

func TestResolveCorrectTitle(t *testing.T) {
	cases := []struct{URL, Title string}{
		{"http://de.wikipedia.org/wiki/Fureidis", "Furaidis"},
		{"http://de.wikipedia.org/wiki/CPU", "Prozessor"},
	}

	wiki := &Wiki{}

	for _, e := range cases {
		if title, err := wiki.PageTitle(e.URL); err != nil {
			t.Fatal("Error while fetching page title:", err)
		} else if title != e.Title {
			t.Fatal("Mismatch:", title, "!=", e.Title)
		}
	}
}
