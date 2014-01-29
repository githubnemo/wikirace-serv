package main

import (
	"fmt"
	"sort"
	"sync"
)

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

	// Lock for Players
	playerLock sync.RWMutex

	// Called every time changes that are worth saving to disk are made
	saveHandler func(*Game)
}

// Usually not called directly as the save handler is relevant to the
// game store which gets notified through the saveHandler that it is time
// to save this game.
func NewGame(hostingPlayerName string, wikiUrl string, saveHandler func(*Game)) *Game {
	game := &Game{
		Host:        hostingPlayerName,
		WikiUrl:     wikiUrl,
		saveHandler: saveHandler,
	}

	game.AddPlayer(hostingPlayerName)

	return game
}

func (g *Game) save() {
	if g.saveHandler != nil {
		g.saveHandler(g)
	}
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
	g.playerLock.Lock()
	defer g.playerLock.Unlock()

	g.Players = append(g.Players, Player{
		Name: name,
	})

	g.save()
}

func (g *Game) GetPlayer(name string) *Player {
	g.playerLock.RLock()
	defer g.playerLock.RUnlock()

	for i, e := range g.Players {
		if e.Name == name {
			return &g.Players[i]
		}
	}
	return nil
}

func (g *Game) HasPlayer(name string) bool {
	g.playerLock.RLock()
	defer g.playerLock.RUnlock()

	for _, e := range g.Players {
		if e.Name == name {
			return true
		}
	}
	return false
}

func (g *Game) SortedPlayers() []Player {
	// FIXME race when Player (through GetPlayer()) is modified while sorting.
	g.playerLock.Lock()
	defer g.playerLock.Unlock()

	sort.Sort(SortablePlayers(g.Players))

	return g.Players
}

func (g *Game) setWinner(player *Player) {
	g.winnerLock.Lock()
	defer g.winnerLock.Unlock()

	g.Winner = player.Name
	g.WinnerPath = player.Path

	player.LeftGame = true

	g.save()
}

func (g *Game) evaluateWinner(player *Player) (isWinner, isTempWinner bool) {
	g.winnerLock.RLock()
	defer g.winnerLock.RUnlock()

	if player.Path[len(player.Path)-1] != g.Goal {
		return false, false
	}

	isTempWinner = g.Winner == player.Name || len(g.WinnerPath) == 0 || len(g.WinnerPath) > len(player.Path)
	isWinner = isTempWinner

	// The player is NOT the full winner if there is an active player with
	// a shorter path.
	for _, p := range g.Players {
		if p.Name != player.Name && len(p.Path) < len(player.Path) && !p.LeftGame {
			isWinner = false
			break
		}
	}

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

// Check whether the player can join this game or not.
// Returns an error if he can't explaining the reason.
func (game *Game) CanJoin(playerName string) error {

	// Check if player name is not taken
	if game.HasPlayer(playerName) {
		// TODO: User friendly error message, maybe redirect the
		// player back to the login form and display a hint.
		return fmt.Errorf("Player name already taken.")
	}

	// Don't allow join when there's already a winner
	if len(game.Winner) > 0 {
		// TODO: user friendly error message. See TODO above.
		return fmt.Errorf("Game is locked as it has already a winner.")
	}

	return nil
}
