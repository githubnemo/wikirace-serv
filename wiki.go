package main

import (
	"fmt"
	"log"
	"net/url"
	"net/http"
	"strings"
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


func serveWikiPage(host, page string, w http.ResponseWriter) {
	content, err := rewriteWikiUrls("http://" + host + "/wiki/" + page)

	if err != nil {
		panic(err)
	}

	w.Write([]byte(content))
}


// Fetch two random pages from wikipedia and get the corresponding
// page titles which will then represent the start and the goal of the game.
func determineStartAndGoal() (string, string, error) {
	const wpRandomUrl = "http://de.wikipedia.org/wiki/Spezial:Zuf%C3%A4llige_Seite"

	sresp, err := http.Head(wpRandomUrl)

	if err != nil {
		return "", "", err
	}

	gresp, err := http.Head(wpRandomUrl)

	if err != nil {
		return "", "", err
	}

	return trimPageName(sresp.Request.URL.Path),
		   trimPageName(gresp.Request.URL.Path),
		   nil
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

		page := trimPageName(link)

		setAttributeValue(e.Nodes[0], "href", serviceVisitUrl(wpUrl.Host, page))
	})

	content, err := doc.Find("#bodyContent").Html()

	if err != nil {
		return "", err
	}

	header, err := doc.Find("head").Html()

	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`<html><head>%s</head><body>%s</body></html>`,
		header, content), nil
}


func getFirstWikiParagraph(wikiUrl string) (string, error) {
	doc, err := goquery.NewDocument(wikiUrl)

	log.Println(wikiUrl)

	if err != nil {
		return "", err
	}

	selections := doc.Find("#mw-content-text p")

	if selections.Length() == 0 {
		return "", fmt.Errorf("No selections found.")
	}

	return selections.First().Text(), nil
}


func trimPageName(path string) string {
	return path[len("/wiki/"):]
}