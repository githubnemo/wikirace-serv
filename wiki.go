package main

import (
	"fmt"
	"log"
	"bytes"
	"net/url"
	"net/http"
	"html/template"
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

func buildWikiPageLink(host, page string) string {
	return "http://" + host + "/wiki/" + page
}

func serveWikiPage(host, page string, w http.ResponseWriter) {
	content, err := rewriteWikiUrls(buildWikiPageLink(host, page))

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

	hrefRewriter := func(i int, e *goquery.Selection) {
		link, ok := e.Attr("href")

		if !ok || !strings.HasPrefix(link, "/wiki/") || strings.Contains(link, ":") {
			return
		}

		page := trimPageName(link)

		setAttributeValue(e.Nodes[0], "href", serviceVisitUrl(wpUrl.Host, page))
	}

	doc.Find("#bodyContent a").Each(hrefRewriter)
	doc.Find("#bodyContent area").Each(hrefRewriter)

	content, err := doc.Find("#bodyContent").Html()

	if err != nil {
		return "", err
	}

	header, err := doc.Find("head").Html()

	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer([]byte{})

	templates.ExecuteTemplate(buf, "wiki.html", struct{
		Header template.HTML
		Content template.HTML
	}{template.HTML(header), template.HTML(content)})

	return buf.String(), nil
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
