package main

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
	Type int
	Message string
}

func createMessage(messagetype int, playername, message string) GameMessage {
	switch messagetype {
	case visit:
		return GameMessage{playername, messagetype, message}
	default:
			return GameMessage{}
	}	
}

func NewJoinMessage(session *GameSession) GameMessage {
	return createMessage(join, session.PlayerName(), session.PlayerName())
}

func NewLeaveMessage(session *GameSession) GameMessage {
	return createMessage(leave, session.PlayerName(), session.PlayerName())
}

func NewFinishMessage(session *GameSession) GameMessage {
	return createMessage(finish, session.PlayerName(), session.PlayerName())
}

func NewVisitMessage(session *GameSession, page string) GameMessage {

	return createMessage(visit, session.PlayerName(), page)
}

func NewGameOverMessage(session *GameSession) GameMessage {
	return createMessage(gameover, session.PlayerName(), session.PlayerName())
}