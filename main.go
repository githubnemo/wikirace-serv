package main

import (
	"crypto/cipher"
	"crypto/des"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"runtime/debug"
)

// Initialized in main()
var (
	session    *GameSessionStore
	gameStore  *GameStore
	templates  *template.Template
	pageCipher cipher.Block
)

// pad the input bytes and return the amount of padded bytes
func pad(in []byte, sz int) (padded []byte, bytes int) {
	padded = in

	if len(in)%sz != 0 {
		newLen := int(float64(sz) * math.Ceil(float64(len(in))/float64(sz)))
		padded = make([]byte, newLen)

		bytes = newLen - len(in)
		copy(padded, in)
	}

	return padded, bytes
}

func encryptPage(page string) string {
	dst, padding := pad([]byte(page), pageCipher.BlockSize())

	pageCipher.Encrypt(dst, dst)

	return fmt.Sprintf("%d:%s", padding, base64.URLEncoding.EncodeToString(dst))
}

func decryptPage(input string) string {
	var padding int
	var b64page string

	_, err := fmt.Sscanf(input, "%d:%s", &padding, &b64page)

	if err != nil {
		panic(err)
	}

	dst, err := base64.URLEncoding.DecodeString(b64page)

	if err != nil {
		panic(err)
	}

	pageCipher.Decrypt(dst, dst)

	return string(dst[:len(dst)-padding])
}

func serviceVisitUrl(wpHost, page string) string {
	if len(page) == 0 {
		panic("Empty page. This is quite likely a bug.")
	}

	page = encryptPage(page)
	wpHost = url.QueryEscape(wpHost)
	return "/visit?page=" + page + "&host=" + wpHost
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		w.WriteHeader(401)

		fmt.Fprintf(w, "Oh...:(\n\n")

		if e, ok := err.(error); ok {
			w.Write([]byte(e.Error()))
			w.Write([]byte{'\n', '\n'})
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

	page = decryptPage(page)
	page, err = url.QueryUnescape(page)

	page, err = url.QueryUnescape(page)

	if err != nil {
		panic(err)
	}

	session, err := session.GetGameSession(r)

	if err != nil || !session.IsInitialized() {
		panic(fmt.Errorf("Invalid session, sorry :/ (Error: %v)", err))
	}

	game, err := session.GetGame()

	if err != nil {
		panic(err)
	}

	session.Visited(page)
	session.Save(r, w)

	// FIXME: this could be racy
	if page == game.Goal {
		// He reached the goal

		// This one is the winner (for now)
		if len(game.WinnerPath) == 0 || len(game.WinnerPath) > len(session.Visits()) {
			game.Winner = session.PlayerName()
			game.WinnerPath = session.Visits()
			err := gameStore.Save(game)

			if err != nil {
				panic(err)
			}

			// TODO: detect actual game win (not only temporary win)

			game.Broadcast(GameMessage(NewFinishMessage(session)))
		}

		templates.ExecuteTemplate(w, "win.html", struct {
			Game            *Game
			Session         *GameSession
			IsWinner        bool
			WinningPageLink string
		}{
			game,
			session,
			game.Winner == session.PlayerName(),
			buildWikiPageLink(host, page),
		})

		return
	}

	msg := NewVisitMessage(session, page)
	game.Broadcast(msg)

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
	http.Redirect(w, r, "/game?id="+game.Hash(), 301)
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	gameId := values.Get("id")
	playerName := values.Get("name")

	if len(gameId) == 0 {
		// Just some lost sould trying to get a game starting, redirect
		// him to the start page.
		http.Redirect(w, r, "/", 301)
		return
	}

	if len(playerName) == 0 {
		templates.ExecuteTemplate(w, "join.html", gameId)
		return
	}

	// Check if game really exists
	if !gameStore.Contains(gameId) {
		panic("No such game")
	}

	game, err := gameStore.GetGameByHash(gameId)

	if err != nil {
		panic(err)
	}

	// Check if player name is not taken
	if game.HasPlayer(playerName) {
		panic("Player name already taken.")
	}

	// Don't allow join when there's already a winner
	if len(game.Winner) > 0 {
		panic("Game is locked as it has already a winner.")
	}

	session, err := session.GetGameSession(r)
	session.Init(playerName, gameId)
	session.Save(r, w)

	game.AddPlayer(playerName)

	// FIXME: racy
	gameStore.Save(game)

	http.Redirect(w, r, "/game?id="+gameId, 301)

	msg := NewJoinMessage(session.PlayerName())
	game.Broadcast(msg)
}

// Serve game content
func gameHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	session, err := session.GetGameSession(r)

	if err != nil {
		panic(err)
	}

	// TODO: session valid but game inexistant -> invalidate session

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	gameId := values.Get("id")

	// session valid but is another game -> overwrite game with new one
	if session.IsInitialized() && len(gameId) > 0 {
		if game, _ := session.GetGame(); game.Hash() != gameId {
			// TODO: warn about losing game
			session.Invalidate()
		}
	}

	// Handle a new player
	if !session.IsInitialized() {
		http.Redirect(w, r, "/join?"+r.URL.RawQuery, 301)
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

	wikiUrl := serviceVisitUrl("de.wikipedia.org", session.LastVisited())

	templates.ExecuteTemplate(w, "game.html", struct {
		Game    *Game
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

// TODO: Player leave messages

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

	pageCipher, err = des.NewCipher([]byte("lirumlar"))

	if err != nil {
		log.Fatal(err)
	}

	gameStore = NewGameStore(NewStore("./games"))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/reload", reloadHandler)
	http.HandleFunc("/visit", visitHandler)
	http.HandleFunc("/start", startHandler)
	http.HandleFunc("/game", gameHandler)
	http.HandleFunc("/join", joinHandler)

	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("assets/js"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("assets/css"))))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
