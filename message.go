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
	// Set the recipient name for the message
	AddressTo(string)

	// Return the name of the concerned player
	PlayerName() string

	Message() string
}

type BaseGameMessage struct {
	RecipientName  string
	PlayerName_    string `json:"PlayerName"`
	Message_       string `json:"Message"`
	Type           int
}

func (p *BaseGameMessage) AddressTo(name string) {
	p.RecipientName = name
}

func (p *BaseGameMessage) PlayerName() string {
	return p.PlayerName_
}

func (p *BaseGameMessage) Message() string {
	return p.Message_
}

type JoinMessage struct {
	*BaseGameMessage
	Player *Player
}

type LeaveMessage struct {
	*BaseGameMessage
}

type VisitMessage struct {
	*BaseGameMessage
	Player *Player
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

func createMessage(typeNum int, playername, message string) *BaseGameMessage {
	return &BaseGameMessage{"", playername, message, typeNum}
}

func NewJoinMessage(player *Player) JoinMessage {
	return JoinMessage{createMessage(join, player.Name, "joined"), player}
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

	return FinishMessage{
		createMessage(finish, player.Name, player.Name),
		len(player.Path),
	}
}

func NewVisitMessage(session *GameSession, page string, player *Player) VisitMessage {
	return VisitMessage{
		createMessage(visit, session.PlayerName(), page),
		player,
	}
}

func NewGameOverMessage(session *GameSession) GameOverMessage {
	return GameOverMessage{createMessage(gameover, session.PlayerName(), session.PlayerName())}
}
