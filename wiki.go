package main

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Wiki struct {
	Name, URL, RandomPage string
}

var wikis []Wiki

// Result type to pass over chan for concurrentHead()
type headResult struct {
	res *http.Response
	err error
}

func init() {
	// TODO: Make this somehow better perhaps? Less timeout? Think about this.
	t := http.DefaultTransport.(*http.Transport)

	t.ResponseHeaderTimeout = 10 * time.Second
}

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

func concurrentHead(url string) chan *headResult {
	c := make(chan *headResult)

	go func() {
		res, err := http.Head(url)

		c <- &headResult{res, err}
	}()

	return c
}

func buildWikiPageLink(host, page string) string {
	return host + "/wiki/" + page
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
func determineStartAndGoal(host string) (string, string, error) {
	wiki := getWikiInformationByUrl(host)
	wpRandomUrl := wiki.URL + "/wiki/" + wiki.RandomPage

	fmt.Println(wpRandomUrl)

	c1 := concurrentHead(wpRandomUrl)
	c2 := concurrentHead(wpRandomUrl)

	sres := <-c1

	if sres.err != nil {
		return "", "", sres.err
	}

	gres := <-c2

	if gres.err != nil {
		return "", "", gres.err
	}

	return trimPageName(sres.res.Request.URL.Path),
		trimPageName(gres.res.Request.URL.Path),
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

	err = templates.ExecuteTemplate(buf, "wiki.html", struct {
		Header  template.HTML
		Content template.HTML
	}{template.HTML(header), template.HTML(content)})

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getFirstWikiParagraph(host, pageTitle string) (string, error) {
	doc, err := goquery.NewDocument(buildWikiPageLink(host, pageTitle))

	if err != nil {
		return "", err
	}

	selections := doc.Find("#mw-content-text > p")

	if selections.Length() == 0 {
		return "", fmt.Errorf("No selections found.")
	}

	return selections.First().Text(), nil
}

func trimPageName(path string) string {
	return path[len("/wiki/"):]
}

func fillSupportedWikis() {
	file, err := ioutil.ReadFile("config/supported_wikis")
	if err == nil {
		json.Unmarshal(file, &wikis)
	}
}

func getWikiInformationByUrl(url string) *Wiki {
	for _, tmpWiki := range wikis {
		if tmpWiki.URL == url {
			return &tmpWiki
		}
	}
	return nil
}
