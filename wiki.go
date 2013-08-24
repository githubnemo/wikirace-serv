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
	"strings"
	"time"
)

type Wiki struct {
	Name, URL, RandomPage, BodySelector string
}

var wikis map[string]Wiki

// Result type to pass over chan for concurrentHead()
type headResult struct {
	res *http.Response
	err error
}

func init() {
	// TODO: Make this somehow better perhaps? Less timeout? Think about this.
	t := http.DefaultTransport.(*http.Transport)

	t.ResponseHeaderTimeout = 10 * time.Second

	var err error

	wikis, err = readSupportedWikis()

	if err != nil {
		panic(err)
	}
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

func buildWikiPageLink(url, page string) string {
	return url + "/wiki/" + page
}

func ServeWikiPage(url, page string, w http.ResponseWriter) {
	doc, err := goquery.NewDocument(buildWikiPageLink(url, page))

	if err != nil {
		panic(err)
	}

	// Links are not clickable as they don't link to a page.
	removeLinksFromImages(doc, url)

	content, err := rewriteWikiUrls(doc, url)

	if err != nil {
		panic(err)
	}

	w.Write([]byte(content))
}

// Fetch two random pages from wikipedia and get the corresponding
// page titles which will then represent the start and the goal of the game.
func DetermineStartAndGoal(url string) (string, string, error) {
	wiki := getWikiInformationByUrl(url)

	wpRandomUrl := buildWikiPageLink(wiki.URL, wiki.RandomPage)

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

// Copied from goquery.Selection.Html().
//
// The original Html method missed to include the
// selected element and only included the children.
func htmlContent(sel *goquery.Selection) (ret string, e error) {
	// Since there is no .innerHtml, the HTML content must be re-created from
	// the nodes usint html.Render().
	var buf bytes.Buffer

	if len(sel.Nodes) > 0 {
		for c := sel.Nodes[0]; c != nil; c = c.NextSibling {
			e = html.Render(&buf, c)
			if e != nil {
				return
			}
		}
		ret = buf.String()
	}

	return
}

func removeLinksFromImages(doc *goquery.Document, wikiUrl string) {
	bodySelector := getWikiInformationByUrl(wikiUrl).BodySelector

	imageRemover := func(i int, e *goquery.Selection) {
		imageNode := e.Nodes[0]
		anchorNode := imageNode.Parent
		anchorParent := anchorNode.Parent

		anchorParent.RemoveChild(anchorNode)
		anchorNode.RemoveChild(imageNode)
		anchorParent.AppendChild(imageNode)
	}

	doc.Find(bodySelector + " a > img").Each(imageRemover)
}


func rewriteWikiUrls(doc *goquery.Document, wikiUrl string) (string, error) {
	hrefRewriter := func(i int, e *goquery.Selection) {
		link, ok := e.Attr("href")

		if !ok || !strings.HasPrefix(link, "/wiki/") || strings.Contains(link, ":") {
			return
		}

		page := trimPageName(link)

		setAttributeValue(e.Nodes[0], "href", serviceVisitUrl(page))
	}

	bodySelector := getWikiInformationByUrl(wikiUrl).BodySelector

	doc.Find(bodySelector + " a").Each(hrefRewriter)
	doc.Find(bodySelector + " area").Each(hrefRewriter)

	content, err := htmlContent(doc.Find(bodySelector))

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

func GetFirstWikiParagraph(url, pageTitle string) (string, error) {
	doc, err := goquery.NewDocument(buildWikiPageLink(url, pageTitle))

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

func readSupportedWikis() (map[string]Wiki, error) {
	var wikis map[string]Wiki

	file, err := ioutil.ReadFile("config/supported_wikis")

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(file, &wikis)

	if err != nil {
		return nil, err
	}

	// This looks broken and is really a workaround for go #3117
	// which says that `wikis[url].URL = url` won't work.
	for url, _ := range wikis {
		w := wikis[url]
		w.URL = url
		wikis[url] = w
	}

	return wikis, nil
}

func getWikiInformationByUrl(url string) *Wiki {
	w := wikis[url]

	return &w
}
