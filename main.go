package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"crypto/rand"
	"io"
	"os"
	"syscall"
)

// Initialized in main()
var (
	session    *GameSessionStore
	gameStore  *GameStore
	templates  *MustTemplates
	pageCipher *PageCipher
)

func serviceVisitUrl(page string) string {
	if len(page) == 0 {
		panic("Empty page. This is quite likely a bug.")
	}

	page = pageCipher.EncryptPage(page)

	return "/visit?page=" + page
}

func mustParseQuery(q string) url.Values {
	values, err := url.ParseQuery(q)

	if err != nil {
		panic(ErrMalformedQuery(err))
	}

	return values
}

// Returned GameSession may be uninitialized.
func mustGetGameSession(r *http.Request) *GameSession {
	session, err := session.GetGameSession(r)

	if err != nil {
		panic(ErrGetGameSession(err))
	}

	return session
}

// Returned GameSession is initialized and ready to use.
func mustGetValidGameSession(r *http.Request) *GameSession {
	session := mustGetGameSession(r)

	if !session.IsInitialized() {
		panic(ErrGetGameSession(fmt.Errorf("Session is not initialized")))
	}

	return session
}

// Accepts visits and serves new wiki page.
//
// Parameters: page
func visitHandler(w http.ResponseWriter, r *http.Request) {
	values := mustParseQuery(r.URL.RawQuery)

	page := values.Get("page")

	page, err := pageCipher.DecryptPage(page)

	if err != nil {
		panic(err)
	}

	page, err = url.QueryUnescape(page)

	if err != nil {
		panic(err)
	}

	session := mustGetValidGameSession(r)

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
	fmt.Fprintf(w, "Player dump: %#v\n", player)
}

// start game session
// params:
// - your name
//
// sets randomly
// - start page
// - end page
func startHandler(w http.ResponseWriter, r *http.Request) {
	values := mustParseQuery(r.URL.RawQuery)

	playerName := values.Get("playerName")
	wikiUrl := values.Get("wikiLanguage")

	// FIXME: overwrites running game of the player
	game := gameStore.NewGame(playerName, wikiUrl)
	host := game.WikiUrl

	start, goal, err := DetermineStartAndGoal(host)

	if err != nil {
		panic(ErrStartAndGoal(err))
	}

	game.Start = start
	game.Goal = goal

	err = gameStore.PutMarshal(game.Hash(), game)

	if err != nil {
		panic(ErrGameMarshal(err))
	}

	session, err := session.GetGameSession(r)

	// TODO: kill previous game with hash `session.Values["hash"]`

	session.Init(playerName, game.Hash())
	session.Save(r, w)

	// Everything went well, tell him he shall go to the game session.
	// The URL to the game shall be shareable.
	http.Redirect(w, r, "/game?id="+game.Hash(), 301)
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	values := mustParseQuery(r.URL.RawQuery)

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
		panic(ErrNoSuchGame(gameId))
	}

	game, err := gameStore.GetGameByHash(gameId)

	if err != nil {
		panic(err)
	}

	if err := game.CanJoin(playerName); err != nil {
		panic(err)
	}

	session := mustGetGameSession(r)

	session.Init(playerName, gameId)
	session.Save(r, w)

	game.AddPlayer(playerName)

	http.Redirect(w, r, "/game?id="+gameId, 301)
}

// Serve game content
func gameHandler(w http.ResponseWriter, r *http.Request) {
	session := mustGetGameSession(r)
	values := mustParseQuery(r.URL.RawQuery)
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
		panic(ErrGetGame(err))
	}

	summary, err := GetFirstWikiParagraph(game.WikiUrl, game.Goal)

	if err != nil {
		summary = err.Error()
		log.Printf("Error fetching summary for game %#v: %s\n", game, err)
	}

	player, err := PlayerFromSession(session)

	if err != nil {
		panic(ErrPlayerLoad(err))
	}

	wikiUrl := serviceVisitUrl(player.LastVisited())

	templates.MustExecuteTemplate(w, "game.html", struct {
		Game    *Game
		Summary string
		WikiURL string
		Player  *Player
	}{game, summary, wikiUrl, player})
}

// Serves initial page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", wikis)
}

func reloadHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	templates, err = parseTemplates()

	if err != nil {
		panic(err)
	}

	fmt.Fprintf(w, "Reload OK.")
}

func setupPageCipher() (*PageCipher, error) {
	key, err := ioutil.ReadFile("./config/key")

	if err != nil {
		if pe, ok := err.(*os.PathError); ok && pe.Err == syscall.ENOENT {
			// Key not found, create a random key
			file, err := os.OpenFile("./config/key", os.O_WRONLY|os.O_CREATE, 0600)

			if err != nil {
				// Give up in being convenient
				return nil, err
			}

			defer file.Close()

			_, err = io.CopyN(file, rand.Reader, PAGE_CIPHER_KEY_LENGTH)

			if err != nil {
				return nil, err
			}

			// Try once again to read the file
			return setupPageCipher()
		}
		return nil, err
	}

	return NewPageCipher(key)
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
	var err error

	session = NewGameSessionStore()

	templates, err = parseTemplates()

	if err != nil || templates == nil {
		log.Fatal("Unable to parse templates: ", err)
	}

	pageCipher, err = setupPageCipher()

	if err != nil {
		log.Fatal(err)
	}

	gameStore = NewGameStore(NewStore("./games"))

	http.HandleFunc("/", errorHandler(indexHandler))
	http.HandleFunc("/reload", errorHandler(reloadHandler))
	http.HandleFunc("/visit", errorHandler(visitHandler))
	http.HandleFunc("/start", errorHandler(startHandler))
	http.HandleFunc("/game", errorHandler(gameHandler))
	http.HandleFunc("/join", errorHandler(joinHandler))

	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("assets/js"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("assets/css"))))
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("assets/img"))))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
