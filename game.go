package main

import (
	"fmt"
	"time"
	"crypto"
	_ "crypto/sha1"
)

var gameHasher = crypto.SHA1.New()

func init() {
	// Initialize the gameHasher with the current date so that
	// the first name entered does not produce the same game hash
	// everytime.
	gameHasher.Write([]byte(time.Now().String()))
}

type Game struct {
	// Cache for the game hash
	hash string

	// Name of the player who initiated the game
	Host string

	// All players including the host
	PlayerHashes []string

	// The winner of the game. Empty if the game is not finished yet
	Winner string

	// The path the winner took to the goal. Empty if the game is not finished
	WinnerPath []string

	// Name of the start and goal article of this game
	Start string
	Goal string
}

func NewGame(hostingPlayerName string) *Game {
	return &Game{
		Host: hostingPlayerName,
		PlayerHashes: []string{hostingPlayerName},
	}
}

func (g *Game) Hash() string {
	if len(g.hash) == 0 {
		g.hash = computeGameHash(g.Host)
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

func (g *Game) Save() error {
	return gameStore.PutMarshal(g.Hash(), g)
}


// Compute a game hash using the hosting player's name.
// This will produce a new hash on every call so that
// a player can create more than one game.
func computeGameHash(playerName string) string {
	gameHasher.Write([]byte(playerName))

	hash := gameHasher.Sum(nil)

	return fmt.Sprintf("%x", hash)
}

// The key has to be present.
func getGameByHash(hash string) (*Game, error) {
	var game Game

	err := gameStore.GetMarshal(hash, &game)

	if err != nil {
		return nil, err
	}

	return &game, nil
}


