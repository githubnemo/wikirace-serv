package main

import (
	"fmt"
	"log"
	"net/url"
	"net/http"
	"html/template"
	"runtime/debug"
)

// Initialized in main()
var (
	session *GameSessionStore
	gameStore *Store
	templates *template.Template
)


func serviceVisitUrl(wpHost, page string) string {
	// TODO: page = encrypt(page)
	page = url.QueryEscape(page)
	wpHost = url.QueryEscape(wpHost)
	return "/visit?page=" + page + "&host=" + wpHost
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
		templates.ExecuteTemplate(w, "join.html", gameId)
		return
	}

	// Check if game really exists
	// TODO

	// Check if player name is not taken
	// TODO

	session, err := session.GetGameSession(r)
	session.Init(playerName, gameId)
	session.Save(r, w)

	game, err := session.GetGame()

	if err != nil {
		panic(err)
	}

	game.AddPlayer(playerName)

	// FIXME: racy
	game.Save()

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

	summary, err := getFirstWikiParagraph("http://de.wikipedia.org/wiki/" + game.Goal)

	if err != nil {
		summary = err.Error()
	}

	wikiUrl := "/visit?page=" + game.Start + "&host=de.wikipedia.org"

	templates.ExecuteTemplate(w, "game.html", struct{
		Game *Game
		Summary string
		WikiURL string
	}{game, summary, wikiUrl})
}

// Serves initial page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	templates.ExecuteTemplate(w, "index.html", nil)
}

func parseTemplates() (err error) {
	templates, err = template.ParseGlob("templates/*.html")

	return err
}

func reloadHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	err := parseTemplates()

	if err != nil {
		panic(err)
	}

	fmt.Fprintf(w, "Reload OK.")
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

	err := parseTemplates()

	if err != nil {
		log.Fatal(err)
	}

	gameStore = NewStore("./games")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/reload", reloadHandler)
	http.HandleFunc("/visit", visitHandler)
	http.HandleFunc("/start", startHandler)
	http.HandleFunc("/game", gameHandler)
	http.HandleFunc("/join", joinHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
