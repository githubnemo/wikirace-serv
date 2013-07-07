package main

import (
	"errors"
)

const (
	visit = iota
	join
	leave
	finish
	gameover
	fatalstuff
)

type GameMessage struct {
	PlayerName string
	Type       int
	Message    string
}

func createMessage(messagetype int, playername, message string) (GameMessage, error) {
	switch messagetype {
	case visit:
		return GameMessage{playername, messagetype, message}, nil
	default:
		return GameMessage{}, errors.New("not a valid messagetype")
	}
}

func NewJoinMessage(session *GameSession) (GameMessage, error) {
	return createMessage(join, session.PlayerName(), session.PlayerName())
}

func NewLeaveMessage(session *GameSession) (GameMessage, error) {
	return createMessage(leave, session.PlayerName(), session.PlayerName())
}

func NewFinishMessage(session *GameSession) (GameMessage, error) {
	return createMessage(finish, session.PlayerName(), session.PlayerName())
}

func NewVisitMessage(session *GameSession, page string) (GameMessage, error) {

	return createMessage(visit, session.PlayerName(), page)
}

func NewGameOverMessage(session *GameSession) (GameMessage, error) {
	return createMessage(gameover, session.PlayerName(), session.PlayerName())
}
