package main

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
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

	n.Attr = append(n.Attr, html.Attribute{
		Key: attrName,
		Val: value,
	})

	return nil
}

func buildWikiPageLink(url, page string) string {
	return url + "/wiki/" + page
}

func ServeWikiPage(url, page string, w http.ResponseWriter) {
	doc, err := goquery.NewDocument(buildWikiPageLink(url, page))
	addCSSOverride(doc)

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


// Result type to pass over chan for concurrentHead()
type pageLookupResponse struct {
	pageTitle string
	res       *http.Response
	err       error
}

// Resolve the page title from a wiki-page-url.
func fetchWikiPageTitle(url string) (string, error) {
	doc, err := goquery.NewDocument(url)

	if err != nil {
		return "", err
	}

	return doc.Find("#firstHeading").Text(), nil
}

// Fetch two random pages from wikipedia and get the corresponding
// page titles which will then represent the start and the goal of the game.
func DetermineStartAndGoal(url string) (string, string, error) {
	wiki := getWikiInformationByUrl(url)

	wpRandomUrl := buildWikiPageLink(wiki.URL, wiki.RandomPage)

	type result struct{ title string; err error }

	c := make(chan result)

	go func() {
		title, err := fetchWikiPageTitle(wpRandomUrl)
		c <- result{title, err}
	}()

	go func() {
		title, err := fetchWikiPageTitle(wpRandomUrl)
		c <- result{title, err}
	}()

	sres := <-c
	gres := <-c

	if sres.err != nil {
		return "", "", sres.err
	}

	if gres.err != nil {
		return "", "", gres.err
	}

	return sres.title, gres.title, nil
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

func isUnsupportedLink(link string) bool {
	return !strings.HasPrefix(link, "/wiki/") || strings.Contains(link, ":")
}

func addCSSOverride(doc *goquery.Document) {
	s := "<link rel='stylesheet' type='text/css' href='css/wiki_overrides.css'>"

	// see: http://stackoverflow.com/questions/15081119/any-way-to-use-html-parse-without-it-adding-nodes-to-make-a-well-formed-tree
	newCssNode, err := html.ParseFragment(strings.NewReader(s), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})

	if err != nil {
		panic(err)
	}

	originalNode := doc.Find("head").Get(0)
	originalNode.AppendChild(newCssNode[0])
}

func rewriteWikiUrls(doc *goquery.Document, wikiUrl string) (string, error) {
	hrefRewriter := func(i int, e *goquery.Selection) {
		link, ok := e.Attr("href")

		if !ok {
			return
		}

		if strings.HasPrefix(link, "#") {
			// Do not rewrite fragment links on the same page as they are
			// useful to the user.
			return
		}

		// Disable unsupported links so that the user does not accidently
		// clicks on these.
		if isUnsupportedLink(link) {
			e.Nodes[0].Attr = append(e.Nodes[0].Attr, html.Attribute{
				Key: "style",
				Val: "color: gray;",
			})
			setAttributeValue(e.Nodes[0], "href", "#"+link)
			setAttributeValue(e.Nodes[0], "onClick", "javascript: alert('This link is not supported by wikiracer, thus it was disabled. If you feel this is an error, contact us. The original target was: "+link+"');")
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
	for url := range wikis {
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
