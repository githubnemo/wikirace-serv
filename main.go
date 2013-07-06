package main

import (
	"fmt"
	"log"
	"strings"
	"net/url"
	"net/http"
	"runtime/debug"
	"github.com/PuerkitoBio/goquery"
	"code.google.com/p/go.net/html"
)

// Initialized in main()
var (
	session *GameSessionStore
	gameStore *Store
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

func trimPageName(path string) string {
	return path[len("/wiki/"):]
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

func errorHandler(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		w.WriteHeader(401)

		fmt.Fprintf(w,"Oh...:(\n\n")

		if e,ok := err.(error); ok {
			w.Write([]byte(e.Error()))
			w.Write([]byte{'\n','\n'})
			w.Write(debug.Stack())
		} else {
			fmt.Fprintf(w, "%s\n\n%s\n", err, debug.Stack())
		}

		log.Println(
			"panic catched:", err,
			"\nRequest data:", r,
			"\nStack:", string(debug.Stack()))
	}
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

// Accepts visits and serves new wiki page
func visitHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	host := values.Get("host")
	page := values.Get("page")

    session, err := session.GetGameSession(r)

	if err != nil || !session.IsInitialized() {
		panic(fmt.Errorf("Invalid session, sorry :/ (Error: %v)", err))
	}

	game, err := session.GetGame()

	if err != nil {
		panic(err)
	}

	// FIXME: this could be racy
	if len(game.Winner) == 0 && page == game.Goal {
		// We have a winner.
		game.Winner = session.PlayerName()

		fmt.Fprintf(w, "u win \\o/")
		return
	}

	session.Visited(page)
    session.Save(r, w)

	serveWikiPage(host, page, w)
	fmt.Fprintf(w, "Session dump: %#v\n", session.Values)
	fmt.Fprintf(w, "Game dump: %#v\n", game)
}

// start game session
// params:
// - your name
//
// sets randomly
// - start page
// - end page
func startHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	playerName := values.Get("playerName")

	// FIXME: overwrites running game

	game := NewGame(playerName)

	start, goal, err := determineStartAndGoal()

	if err != nil {
		panic(err)
	}

	game.Start = start
	game.Goal = goal

	err = gameStore.PutMarshal(game.Hash(), game)

	if err != nil {
		panic(err)
	}

	session, err := session.GetGameSession(r)

	if err != nil {
		panic(err)
	}

	// TODO: kill previous game with hash `session.Values["hash"]`

	session.Init(playerName, game.Hash())
	session.Save(r, w)

	// Everything went well, tell him he shall go to the game session.
	// The URL to the game shall be shareable.
	http.Redirect(w, r, "/game?id=" + game.Hash(), 300)
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	gameId := values.Get("id")
	playerName := values.Get("name")

	if len(playerName) == 0 {
		doc := `
<html>
	<form method="get">
		<label for="name">Enter your name</label>
		<input type="hidden" name="id" value="%s">
		<input type="text" name="name">
		<input type="submit">
	</form>
</html>
`
		fmt.Fprintf(w, doc, gameId)
		return
	}

	// Check if game really exists
	// TODO

	// Check if player name is not taken
	// TODO

	session, err := session.GetGameSession(r)
	session.Init(playerName, gameId)
	session.Save(r, w)

	http.Redirect(w, r, "/game?id=" + gameId, 301)
}

// Serve game content
func gameHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	session, err := session.GetGameSession(r)

	if err != nil {
		panic(err)
	}

	if !session.IsInitialized() {
		http.Redirect(w, r, "/join?" + r.URL.RawQuery, 300)
		return
	}

	game, err := session.GetGame()

	if err != nil {
		panic(err)
	}

	wikiUrl := "/visit?page=" + game.Start + "&host=de.wikipedia.org"
	doc := `
<html>
	Start: %s<br>
	Goal: %s<br>
	<iframe name="gameFrame" width="50%%" height="50%%" src="%s"></iframe>
</html>
`
	fmt.Fprintf(w, doc, game.Start, game.Goal, wikiUrl)
}

// Serves initial page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	doc := `
<html>
	Start new game:
	<form method="get" action="/start">
		<label for="playerName">Player name:</label>
		<input type="text" name="playerName">
		<input type="submit">
	</form>
	%q
</html>
`
	fmt.Fprintf(w, doc, html.EscapeString(r.URL.Path))
}

// TODO: send new visit and visit of opponent to website

// Game initialization:
//
// foo goes to: /start?player=foo
// game creates link: /game?id=<uniqGameHash(foo)>
//
// Participation:
//
// bar goes to: /game?id=<uniqGameHash(foo)>
// server knows: this is not foo or a playing player, ask for name
// bar enters name: bar
// server registers player bar in game

func main() {
	session = NewGameSessionStore()

	gameStore = NewStore("./games")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/visit", visitHandler)
	http.HandleFunc("/start", startHandler)
	http.HandleFunc("/game", gameHandler)
	http.HandleFunc("/join", joinHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
