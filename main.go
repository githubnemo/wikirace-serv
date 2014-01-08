package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"

	"crypto/rand"
	"io"
	"os"
	"syscall"
)

// Initialized in main()
var (
	session    *GameSessionStore
	gameStore  *GameStore
	templates  *template.Template
	pageCipher *PageCipher
)

func serviceVisitUrl(page string) string {
	if len(page) == 0 {
		panic("Empty page. This is quite likely a bug.")
	}

	page = pageCipher.EncryptPage(page)

	return "/visit?page=" + page
}


// Accepts visits and serves new wiki page
func visitHandler(w http.ResponseWriter, r *http.Request) {
	defer commonErrorHandler(w, r)

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(err)
	}

	page := values.Get("page")

	page = pageCipher.DecryptPage(page)
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
	fmt.Fprintf(w, "Player dump: %#v\n", player)
}

type MessageGeneratorFunc func(error) string

type ErrStartAndGoal error
type ErrMalformedQuery error
type ErrGameMarshal error
type ErrGetGameSession error
type ErrGetGame error
type ErrNoSuchGame string

func (e ErrNoSuchGame) Error() string {
	return "Game " + string(e) + " could not be found in game store."
}

func userFriendlyError(e error) string {
	switch e.(type) {
	case ErrStartAndGoal:
		return "I could not find where the wiki is in the intertubes."
	case ErrMalformedQuery:
		return "The stuff you typed in the URI I don't understand."
	case ErrGameMarshal:
		return "I could not save th1s game! M4ybe m_ disks a-re f-f-f---aulll-.."
	case ErrGetGameSession:
		return "Tried to get your game session but failed. Maybe you threw away your cookies? Who would do that to his cookies?!"
	case ErrNoSuchGame:
		return "Pretty much what it says, there does not seem to be such a game. Maybe your game expired or there's an error on our side."
	case ErrGetGame:
		return "I failed to retrieve this game, maybe there's something wrong?"
	}

	// We don't know this error. Panic to let the commonErrorHandler handle
	// this unknown error.
	panic(e)
}

func commonErrorHandler(w http.ResponseWriter, r *http.Request) {
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

func userFriendlyErrorHandler(w http.ResponseWriter, r *http.Request) {
	// In case the error can't be handled in here fall back to the
	// common error handler.
	defer commonErrorHandler(w, r)

	if e, ok := recover().(error); false && ok && e != nil {
		understandableErrorMessage := userFriendlyError(e)

		templates.ExecuteTemplate(w, "error.html", struct {
			ErrorMessage               string
			UnderstandableErrorMessage string
		}{e.Error(), understandableErrorMessage})

	} else if e != nil {
		// Re-panic as we haven't handled the error yet.
		panic(e)
	}
}

// start game session
// params:
// - your name
//
// sets randomly
// - start page
// - end page
func startHandler(w http.ResponseWriter, r *http.Request) {
	defer userFriendlyErrorHandler(w, r)



	values, err := url.ParseQuery(r.URL.RawQuery)

	// FIXME FIXME FIXME should use commonErrorHandler
	panic("FOOO")

	if err != nil {
		panic(ErrMalformedQuery(err))
	}

	playerName := values.Get("playerName")
	wikiUrl := values.Get("wikiLanguage")

	// FIXME: overwrites running game
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

	if err != nil {
		panic(ErrGetGameSession(err))
	}

	// TODO: kill previous game with hash `session.Values["hash"]`

	session.Init(playerName, game.Hash())
	session.Save(r, w)

	// Everything went well, tell him he shall go to the game session.
	// The URL to the game shall be shareable.
	http.Redirect(w, r, "/game?id="+game.Hash(), 301)
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	defer commonErrorHandler(w, r)
	defer userFriendlyErrorHandler(w, r)

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(ErrMalformedQuery(err))
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
		panic(ErrNoSuchGame(gameId))
	}

	game, err := gameStore.GetGameByHash(gameId)

	if err != nil {
		panic(err)
	}

	// Check if player name is not taken
	if game.HasPlayer(playerName) {
		// TODO: User friendly error message, maybe redirect the
		// player back to the login form and display a hint.
		panic("Player name already taken.")
	}

	// Don't allow join when there's already a winner
	if len(game.Winner) > 0 {
		// TODO: user friendly error message. See TODO above.
		panic("Game is locked as it has already a winner.")
	}

	session, err := session.GetGameSession(r)
	session.Init(playerName, gameId)
	session.Save(r, w)

	game.AddPlayer(playerName)

	http.Redirect(w, r, "/game?id="+gameId, 301)
}

// Serve game content
func gameHandler(w http.ResponseWriter, r *http.Request) {
	defer commonErrorHandler(w, r)
	defer userFriendlyErrorHandler(w, r)

	session, err := session.GetGameSession(r)

	if err != nil {
		panic(err)
	}

	values, err := url.ParseQuery(r.URL.RawQuery)

	if err != nil {
		panic(ErrMalformedQuery(err))
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
		panic(ErrGetGame(err))
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
		Player  *Player
	}{game, summary, wikiUrl, player})
}

// Serves initial page
func indexHandler(w http.ResponseWriter, r *http.Request) {
	defer commonErrorHandler(w, r)
	templates.ExecuteTemplate(w, "index.html", wikis)
}

func parseTemplates() (err error) {
	templates, err = template.ParseGlob("templates/*.html")

	return err
}

func reloadHandler(w http.ResponseWriter, r *http.Request) {
	defer commonErrorHandler(w, r)

	err := parseTemplates()

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
	session = NewGameSessionStore()

	err := parseTemplates()

	if err != nil {
		log.Fatal(err)
	}

	pageCipher, err = setupPageCipher()

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
}
