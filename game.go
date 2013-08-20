package main

type Game struct {
	// Cache for the game hash
	hash string

	// Name of the player who initiated the game
	Host string

	// All players including the host
	Players []Player

	// The winner of the game. Empty if the game is not finished yet
	Winner string

	// The path the winner took to the goal. Empty if the game is not finished
	WinnerPath []string

	// Name of the start and goal article of this game
	Start string
	Goal  string

	// The language of that certain Game
	WikiUrl string
}

func NewGame(hostingPlayerName string, wikiUrl string) *Game {
	game := &Game{
		Host:    hostingPlayerName,
		WikiUrl: wikiUrl,
	}

	game.AddPlayer(hostingPlayerName)

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
	g.Players = append(g.Players, Player{
		Name: name,
	})
}

func (g *Game) GetPlayer(name string) *Player {
	for i, e := range g.Players {
		if e.Name == name {
			return &g.Players[i]
		}
	}
	return nil
}

func (g *Game) HasPlayer(name string) bool {
	for _, e := range g.Players {
		if e.Name == name {
			return true
		}
	}
	return false
}
