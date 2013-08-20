package main

import (
	"fmt"
)

type Player struct {
	Path    []string
	Name    string
	Session *GameSession `json:"-"`

	game *Game
}

// Return the player from the game with the session assigned.
func PlayerFromSession(session *GameSession) (*Player, error) {
	game, err := session.GetGame()

	if err != nil {
		return nil, err
	}

	p := game.GetPlayer(session.PlayerName())

	if p == nil {
		return nil, fmt.Errorf("Player %s is not in the game %s.", session.PlayerName(), game.Hash)
	}

	p.Session = session
	p.game = game

	return p, nil
}

func (p *Player) Visited(page string) {
	p.Path = append(p.Path, page)
}

func (p *Player) LastVisited() string {
	visits := p.Path

	if len(visits) == 0 {
		return p.game.Start
	}

	return visits[len(visits)-1]
}
