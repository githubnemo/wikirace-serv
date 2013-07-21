package main

const (
	visit = iota
	join
	leave
	finish
	gameover
	fatalstuff
)

type GameMessage interface {
	PlayerName() string
	Message() string
}

type BaseGameMessage struct {
	PlayerName_ string `json:"PlayerName"`
	Message_    string `json:"Message"`
	Type        int
}

func (p *BaseGameMessage) PlayerName() string {
	return p.PlayerName_
}

func (p *BaseGameMessage) Message() string {
	return p.Message_
}

type JoinMessage struct {
	*BaseGameMessage
}

type LeaveMessage struct {
	*BaseGameMessage
}

type VisitMessage struct {
	*BaseGameMessage
}

type FinishMessage struct {
	*BaseGameMessage
	Visits int
}

type GameOverMessage struct {
	*BaseGameMessage
}

type FatalStuffMessage struct {
	*BaseGameMessage
}

// TODO: Does this work with type aliases as well?

func createMessage(typeNum int, playername, message string) *BaseGameMessage {
	return &BaseGameMessage{playername, message, typeNum}
}

func NewJoinMessage(playername string) JoinMessage {
	return JoinMessage{createMessage(join, playername, "joined")}
}

func NewLeaveMessage(session *GameSession) LeaveMessage {
	return LeaveMessage{createMessage(leave, session.PlayerName(), session.PlayerName())}
}

func NewFinishMessage(session *GameSession) FinishMessage {
	player, err := PlayerFromSession(session)

	if err != nil {
		// Attempt to create a finish message but the player does not
		// exist in the session's game OR the game does not exist.
		// Both conditions should've been handled before.
		panic(err)
	}

	return FinishMessage{createMessage(finish, player.Name, player.Name), len(player.Path)}
}

func NewVisitMessage(session *GameSession, page string) VisitMessage {
	return VisitMessage{createMessage(visit, session.PlayerName(), page)}
}

func NewGameOverMessage(session *GameSession) GameOverMessage {
	return GameOverMessage{createMessage(gameover, session.PlayerName(), session.PlayerName())}
}
