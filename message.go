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
	Type       int
	Message    string
}

type JoinMessage GameMessage
type LeaveMessage GameMessage
type VisitMessage GameMessage
type FinishMessage GameMessage
type GameOverMessage GameMessage
type FatalStuffMessage GameMessage

func createMessage(messagetype int, playername, message string) GameMessage {
	return GameMessage{playername, messagetype, message}
}

func NewJoinMessage(playername string) JoinMessage {
	return JoinMessage(createMessage(join, playername, "joined"))
}

func NewLeaveMessage(session *GameSession) LeaveMessage {
	return LeaveMessage(createMessage(leave, session.PlayerName(), session.PlayerName()))
}

func NewFinishMessage(session *GameSession) FinishMessage {
	return FinishMessage(createMessage(finish, session.PlayerName(), session.PlayerName()))
}

func NewVisitMessage(session *GameSession, page string) VisitMessage {
	return VisitMessage(createMessage(visit, session.PlayerName(), page))
}

func NewGameOverMessage(session *GameSession) GameOverMessage {
	return GameOverMessage(createMessage(gameover, session.PlayerName(), session.PlayerName()))
}
