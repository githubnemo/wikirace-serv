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


type GameStore struct {
	*Store
}

func NewGameStore(s *Store) *GameStore {
	return &GameStore{s}
}

func (g *GameStore) NewGameHash(playerName string) (shash string) {
	gameHasher.Write([]byte(playerName))

	for {
		hash := gameHasher.Sum(nil)
		shash := fmt.Sprintf("%x", hash)

		if !g.Contains(shash) {
			break
		}
	}

	return shash
}

func (g *GameStore) Save(game *Game) error {
	return g.PutMarshal(game.Hash(), game)
}

// The key has to be present.
func (g *GameStore) GetGameByHash(hash string) (*Game, error) {
	var game Game

	err := g.GetMarshal(hash, &game)

	if err != nil {
		return nil, err
	}

	// Only the game store knows the real hash, so we set it here.
	game.hash = hash

	return &game, nil
}

