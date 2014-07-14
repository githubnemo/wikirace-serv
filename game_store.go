package main

import (
	"crypto"
	_ "crypto/sha1"
	"fmt"
	"time"
	"wikirace-serv/wikis"
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

func (g *GameStore) gameSaveHandler(game *Game) {
	// Note that there is no lock needed here as PutMarshal works atomically.
	err := gameStore.PutMarshal(game.Hash(), game)

	if err != nil {
		// TODO: proper error handling
		panic(err)
	}
}

func (g *GameStore) NewGame(hostingPlayerName string, wiki *wikis.Wiki) *Game {
	return NewGame(hostingPlayerName, wiki, g.gameSaveHandler)
}

// Only one (pooled) instance of a game instance is returned.
// The key has to be present.
func (g *GameStore) GetGameByHash(hash string) (*Game, error) {
	if game, ok := g.activeGames[hash]; ok {
		return game, nil
	}

	// Create empty game, values don't matter as they'll be
	// overwritten by GetMarshal. What matters is the initialization
	// of private members and start of go routines and such.
	game := g.NewGame("", nil)

	err := g.GetMarshal(hash, game)

	if err != nil {
		return nil, err
	}

	// Only the game store knows the real hash, so we set it here.
	game.hash = hash

	g.activeGames[hash] = game

	return game, nil
}
