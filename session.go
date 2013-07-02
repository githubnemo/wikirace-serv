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
}

func (s *GameSession) PlayerName() string {
	return s.Values["name"].(string)
}

func (s *GameSession) GameHash() string {
	return s.Values["hash"].(string)
}

func (s *GameSession) Visited(page string) {
	visits := s.Values["visits"].([]string)

	visits = append(visits, page)

	s.Values["visits"] = visits
}
