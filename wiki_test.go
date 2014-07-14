package main

import (
	"testing"
)

func TestResolveCorrectTitle(t *testing.T) {
	cases := []struct{URL, Title string}{
		{"http://de.wikipedia.org/wiki/Fureidis", "Furaidis"},
		{"http://de.wikipedia.org/wiki/CPU", "Prozessor"},
	}

	for _, e := range cases {
		if title, err := fetchWikiPageTitle(e.URL); err != nil {
			t.Fatal("Error while fetching page title:", err)
		} else if title != e.Title {
			t.Fatal("Mismatch:", title, "!=", e.Title)
		}
	}
}
