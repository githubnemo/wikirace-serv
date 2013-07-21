package main

import (
	"crypto"
	_ "crypto/sha1"
	"fmt"
	"time"
)

var gameHasher = crypto.SHA1.New()

func init() {
	// Initialize the gameHasher with the current date so that
	// the first name entered does not produce the same game hash
	// everytime.
	gameHasher.Write([]byte(time.Now().String()))
}

type GameStore struct {
	*Store

	activeGames map[string]*Game
}

func NewGameStore(s *Store) *GameStore {
	return &GameStore{s, make(map[string]*Game)}
}

func (g *GameStore) NewGameHash(playerName string) (shash string) {
	gameHasher.Write([]byte(playerName))

	for i := 0; ; i++ {
		hash := gameHasher.Sum(nil)
		shash = fmt.Sprintf("%x", hash)

		if !g.Contains(shash) {
			break
		}

		gameHasher.Write([]byte{byte(i & 0xFF)})
	}

	return shash
}

func (g *GameStore) Save(game *Game) error {
	// TODO: remove from activeGames when ended

	return g.PutMarshal(game.Hash(), game)
}

// Only one (pooled) instance of a game instance is returned.
// The key has to be present.
func (g *GameStore) GetGameByHash(hash string) (*Game, error) {
	if game, ok := g.activeGames[hash]; ok {
		return game, nil
	}

	game := NewGame("", "")

	err := g.GetMarshal(hash, game)

	if err != nil {
		return nil, err
	}

	// Only the game store knows the real hash, so we set it here.
	game.hash = hash

	g.activeGames[hash] = game

	return game, nil
}
