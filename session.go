package main

import (
	"net/http"
	"github.com/gorilla/sessions"
)

type GameSessionStore struct {
	*sessions.CookieStore
}

func NewGameSessionStore() *GameSessionStore {
	var encryptionKey = "lirumlarum"

	return &GameSessionStore{
		sessions.NewCookieStore([]byte(encryptionKey)),
	}
}

func (s *GameSessionStore) GetGameSession(r *http.Request) (*GameSession, error) {
	session, err := s.Get(r, "game")
	return &GameSession{session}, err
}

type GameSession struct {
	*sessions.Session
}

func (s *GameSession) Init(player, game string) {
	s.Values["hash"] = game
	s.Values["name"] = player
	s.Values["visits"] = []string{}
	s.Values["initialized"] = true
}

// Method to check whether the session was initialized properly
// and is generally safe to use.
func (s *GameSession) IsInitialized() bool {
	if _, ok := s.Values["initialized"]; ok {
		if v, ok := s.Values["initialized"].(bool); ok {
			return v
		}
	}
	return false
}

func (s *GameSession) PlayerName() string {
	return s.Values["name"].(string)
}

func (s *GameSession) GameHash() string {
	return s.Values["hash"].(string)
}

func (s *GameSession) Visits() []string {
	return s.Values["visits"].([]string)
}

func (s *GameSession) LastVisited() string {
	visits := s.Visits()

	if len(visits) == 0 {
		game, err := s.GetGame()

		if err != nil {
			// The session was not initialized properly
			// or a bug happened.
			panic(err)
		}

		return game.Start
	}

	return visits[len(visits)-1]
}

func (s *GameSession) Visited(page string) {
	visits := s.Visits()

	visits = append(visits, page)

	s.Values["visits"] = visits
}

func (s *GameSession) GetGame() (*Game, error) {
	return gameStore.GetGameByHash(s.GameHash())
}
