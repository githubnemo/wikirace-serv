package main

import "sync"

type Game struct {
	// Cache for the game hash
	hash string

	// Name of the player who initiated the game
	Host string

	// All players including the host.
	// Players are never deleted from the game.
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

	// Lock for Winner / WinnerPath
	winnerLock sync.RWMutex
}

func NewGame(hostingPlayerName string, wikiUrl string) *Game {
	game := &Game{
		Host:          hostingPlayerName,
		WikiUrl:       wikiUrl,
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

func (g *Game) setWinner(player *Player) {
	g.winnerLock.Lock()
	defer g.winnerLock.Unlock()

	g.Winner = player.Name
	g.WinnerPath = player.Path
}

func (g *Game) evaluateWinner(player *Player) (isWinner, isTempWinner bool) {
	g.winnerLock.RLock()
	defer g.winnerLock.RUnlock()

	isTempWinner = len(g.WinnerPath) == 0 || len(g.WinnerPath) > len(player.Path)

	// TODO: check if full winner
	isWinner = false

	return
}

// If the player is winner, he is also the temporary winner but this
// is not the case the other way around.
//
// tempWinner means that the player is the current winner but can be
// defeated by another player with a shorter path. The player is a
// winner if nobody can achieve (given up) or has a shorter path.
func (g *Game) EvaluateWinner(player *Player) (isWinner, isTempWinner bool) {
	isWinner, isTempWinner = g.evaluateWinner(player)

	if isTempWinner || isWinner {
		g.setWinner(player)
	}

	return
}

// nil if no player has won yet.
func (g *Game) GetWinner() *Player {
	g.winnerLock.RLock()
	defer g.winnerLock.RUnlock()

	if len(g.Winner) == 0 {
		return nil
	}

	return g.GetPlayer(g.Winner)
}