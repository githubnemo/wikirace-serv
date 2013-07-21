package main

import (
	"github.com/gorilla/sessions"
	"net/http"
)

type GameSessionStore struct {
	*sessions.CookieStore
}

func NewGameSessionStore() *GameSessionStore {
	// FIXME: Configurable encryption key
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
	s.Values["initialized"] = true
}

func (s *GameSession) Invalidate() {
	s.Values["hash"] = ""
	s.Values["name"] = ""
	s.Values["initialized"] = false
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

func (s *GameSession) GetGame() (*Game, error) {
	return gameStore.GetGameByHash(s.GameHash())
}
