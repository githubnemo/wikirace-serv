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

func serviceVisitUrl(page string) string {
	if len(page) == 0 {
		panic("Empty page. This is quite likely a bug.")
	}

	page = encryptPage(page)

	return "/visit?page=" + page
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

	page := values.Get("page")

	page = decryptPage(page)
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

	player, err := PlayerFromSession(session)

	if err != nil {
		panic(err)
	}

	player.Visited(page)

	// He reached the goal
	if page == game.Goal {

		isWinner, isTemporaryWinner := game.EvaluateWinner(player)

		switch {
		case isWinner:
			game.Broadcast(GameMessage(NewGameOverMessage(session)))
		case isTemporaryWinner:
			game.Broadcast(GameMessage(NewFinishMessage(session)))
		}

		templates.ExecuteTemplate(w, "win.html", struct {
			Game            *Game
			Player          *Player
			IsWinner        bool
			WinningPageLink string
		}{
			game,
			player,
			game.Winner == player.Name,
			buildWikiPageLink(game.WikiUrl, page),
		})

		return
	}

	game.Broadcast(NewVisitMessage(session, page, player))

	ServeWikiPage(game.WikiUrl, page, w)

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
	wikiUrl := values.Get("wikiLanguage")

	// FIXME: overwrites running game
	game := NewGame(playerName, wikiUrl)

	// TODO: make this selectable

	host := game.WikiUrl

	start, goal, err := DetermineStartAndGoal(host)

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
}

// Serve game content
func gameHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)

	session, err := session.GetGameSession(r)

	if err != nil {
		panic(err)
	}

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	gameId := values.Get("id")

	// The session is valid but the game is inexistant or does not
	// match the game hash stored in the session. This can have the
	// following reasons:
	//
	// 1. The user came here from another game
	// 2. The game does not exist on the server (anymore)
	// 3. The user forged the session
	//
	// We do not support multiple games at once at this moment,
	// therefore all cases invalidate the session. The UI should
	// warn the user and make a check before submitting to the
	// server.
	//
	// TODO: Implement check in UI
	if session.IsInitialized() && len(gameId) > 0 {
		if game, err := session.GetGame(); err != nil || game.Hash() != gameId {
			// Log the (assumed) physical loss of a game when err != nil.
			// We can't know if we actually lost a game so better be safe.
			if err != nil {
				log.Printf("Game with hash %s was requested by %#v but not found on disk!", gameId, session)
			}

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

	summary, err := GetFirstWikiParagraph(game.WikiUrl, game.Goal)

	if err != nil {
		summary = err.Error()

		log.Printf("Error fetching summary for game %#v: %s\n", game, err)
	}

	player, err := PlayerFromSession(session)

	if err != nil {
		panic(err)
	}

	wikiUrl := serviceVisitUrl(player.LastVisited())

	templates.ExecuteTemplate(w, "game.html", struct {
		Game    *Game
		Summary string
		WikiURL string
	}{game, summary, wikiUrl})
}

// Serves initial page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	defer errorHandler(w, r)
	templates.ExecuteTemplate(w, "index.html", wikis)
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
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("assets/img"))))

	log.Fatal(http.ListenAndServe(":8080", nil))

	// TODO: Save all active games.
}
