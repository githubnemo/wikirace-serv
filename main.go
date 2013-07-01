package main

import (
	"fmt"
	"log"
	"strings"
	"net/url"
	"net/http"
	"github.com/PuerkitoBio/goquery"
	"code.google.com/p/go.net/html"
	"github.com/gorilla/sessions"
)

var encryptionKey = "lirumlarum"
var store = sessions.NewCookieStore([]byte(encryptionKey))

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

func panicOnError(err error) {
	panic(err)
}

func serveWikiPage(host, page string, w http.ResponseWriter) {
	content, err := rewriteWikiUrls("http://" + host + "/wiki/" + page)

	if err != nil {
		panic(err)
	}

	fmt.Fprintf(w, content)
}

// Accepts visits and serves new wiki page
func visitHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	host := values.Get("host")
	page := values.Get("page")

    session, _ := store.Get(r, "game")

	if i, ok := session.Values["visits"].(int); !ok {
		session.Values["visits"] = 1
	} else {
		session.Values["visits"] = i + 1
	}

	session.Values["opponent"] = "someName"

    session.Save(r, w)

	serveWikiPage(host, page, w)
}

func visitsHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

    session, _ := store.Get(r, "game")

	fmt.Fprintf(w, "visits: %d, opponent: %s\n", session.Values["visits"],
		session.Values["opponent"])
}


// Serves initial page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	wikiUrl := "/visit?page=jQuery&host=de.wikipedia.org"
	doc := `
<html>
	<iframe src="%s"></iframe>
	%q
</html>
`
	fmt.Fprintf(w, doc, wikiUrl, html.EscapeString(r.URL.Path))
}

// TODO: send new visit and visit of opponent to website

func main() {
//	serveWikiUrl("http://de.wikipedia.org/wiki/jQuery")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/visit", visitHandler)
	http.HandleFunc("/visits", visitsHandler)
	http.HandleFunc("/serve", visitHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
