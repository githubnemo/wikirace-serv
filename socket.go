package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"log"
	"net/http"
)

var (
	JSON          = websocket.JSON                // codec for JSON
	Message       = websocket.Message             // codec for string, []byte
	ActiveClients = make(map[ClientConn]struct{}) // map containing clients

	ClientHandler = SocketHandler(make(map[*Game]gameClients))
)

// Client connection consists of the websocket and the client ip
type ClientConn struct {
	websocket   *websocket.Conn
	clientIP    string
	inputChan   *chan GameMessage
	playerName  string
}

func init() {
	http.Handle("/client", websocket.Handler(SockServer))
}

type gameClients map[ClientConn]struct{}

type SocketHandler map[*Game]gameClients

// TODO: error returned?
func (handler SocketHandler) Broadcast(game *Game, msg GameMessage) {
	clients := handler[game]

	for client, _ := range clients {
		msg.AddressTo(client.playerName)
		*client.inputChan <- msg
	}
}

// Just drop it if it exists, otherwise ignore
func (handler SocketHandler) LazyRemoveClient(game *Game, con ClientConn) {
	if clients, ok := handler[game]; ok {
		delete(clients, con)
	}
}

func (handler SocketHandler) NewConnection(game *Game, con ClientConn) {
	// TODO: error reporting
	if !gameStore.Contains(game.Hash()) {
		return
	}

	if _, ok := ClientHandler[game]; !ok {
		ClientHandler[game] = map[ClientConn]struct{}{
			con: struct{}{},
		}
	} else {
		ClientHandler[game][con] = struct{}{}
	}
}

// WebSocket server to handle chat between clients.
//
// Accept incoming connections and associate the game session with the
// connection. As the session information is encrypted we can be sure that
// this is a valid player.
//
func SockServer(ws *websocket.Conn) {
	var game *Game

	clientIP := ws.Request().RemoteAddr
	inputChan := make(chan GameMessage)

	request := ws.Request()
	sess, err := session.GetGameSession(request)

	if err != nil {
		panic("there is no game associated to this session, wad?")
	} else {
		game, err = sess.GetGame()

		if err != nil {
			panic("SocketServer: game not found: " + err.Error())
		}
	}

	sockCli := ClientConn{ws, clientIP, &inputChan, sess.PlayerName()}

	// Register client connection in global ClientHandler
	ClientHandler.NewConnection(game, sockCli)

	log.Println("client connect ...", clientIP)

	player, err := PlayerFromSession(sess)

	if err != nil {
		panic(err)
	}

	// Schedule sending the broadcast as the listener
	// is the for loop below.
	go game.Broadcast(NewJoinMessage(player))

	// cleanup on server side
	defer func() {
		ClientHandler.LazyRemoveClient(game, sockCli)

		if err := ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}
	}()

	// for loop so the websocket stays open otherwise
	// it'll close after one Receieve and Send
	for {
		select {
		case msg := <-inputChan:
			res, err := json.Marshal(msg)

			if err != nil {
				log.Fatal("error encoding message: ", msg, err)
			}

			if err = Message.Send(ws, string(res)); err != nil {
				// we could not send the message to a peer
				log.Println("Could not send message to ",
					clientIP, err.Error(), " - dropping client.")

				break
			}
		}
	}
}
