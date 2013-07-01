package main

import (
	"fmt"
	"log"
	"strings"
	"net/url"
	"net/http"
	"github.com/PuerkitoBio/goquery"
	"code.google.com/p/go.net/html"
)

func setAttributeValue(n *html.Node, attrName, value string) error {
	for i, a := range n.Attr {
		if a.Key == attrName {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			n.Attr = append(n.Attr, html.Attribute{"", "href", value})
			return nil
		}
	}

	return fmt.Errorf("Didn't find attribute %s.\n", attrName)
}

func serviceVisitUrl(wpHost, page string) string {
	// TODO: page = encrypt(page)
	return "/visit?page=" + page + "&host=" + wpHost
}

func rewriteWikiUrls(wikiUrl string) (string, error) {
	doc, err := goquery.NewDocument(wikiUrl)

	if err != nil {
		return "", err
	}

	wpUrl, err := url.Parse(wikiUrl)

	if err != nil {
		return "", err
	}

	doc.Find("#bodyContent a").Each(func(i int, e *goquery.Selection) {
		link, ok := e.Attr("href")

		if !ok || !strings.HasPrefix(link, "/wiki/") || strings.Contains(link, ":") {
			return
		}

		page := link[len("/wiki/"):]

		setAttributeValue(e.Nodes[0], "href", serviceVisitUrl(wpUrl.Host, page))
	})

	return doc.Find("#bodyContent").Html()
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
	}
}

// Accepts visits and serves new wiki page
func visitHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	content, err := rewriteWikiUrls("http://de.wikipedia.org/wiki/" + values.Get("page"))

	if err != nil {
		panic(err)
	}

	fmt.Fprintf(w, content)
}

// Serves initial page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	wikiUrl := "/visit?page=jQuery"
	doc := `
<html>
	<iframe src="%s"></iframe>
	%q
</html>
`
	fmt.Fprintf(w, doc, wikiUrl, html.EscapeString(r.URL.Path))
}

func main() {
//	serveWikiUrl("http://de.wikipedia.org/wiki/jQuery")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/visit", visitHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
