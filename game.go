package main

type Game struct {
	// Cache for the game hash
	hash string

	// Name of the player who initiated the game
	Host string

	// Host of the wiki page that is used to get articles from
	WikiHost string

	// All players including the host
	PlayerHashes []string

	// The winner of the game. Empty if the game is not finished yet
	Winner string

	// The path the winner took to the goal. Empty if the game is not finished
	WinnerPath []string

	// Name of the start and goal article of this game
	Start string
	Goal  string
}

func NewGame(hostingPlayerName, wikiHost string) *Game {
	game := &Game{
		Host:         hostingPlayerName,
		WikiHost:     wikiHost,
		PlayerHashes: []string{hostingPlayerName},
	}

	return game
}

func (g *Game) Broadcast(msg GameMessage) {
	ClientHandler.Broadcast(g, msg)
}

func (g *Game) Hash() string {
	if len(g.hash) == 0 {
		g.hash = gameStore.NewGameHash(g.Host)
	}

	return g.hash
}

func (g *Game) AddPlayer(name string) {
	g.PlayerHashes = append(g.PlayerHashes, name)
}

func (g *Game) HasPlayer(name string) bool {
	for _, e := range g.PlayerHashes {
		if e == name {
			return true
		}
	}
	return false
}

