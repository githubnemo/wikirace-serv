package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"log"
	"net/http"
)

var (
	JSON          = websocket.JSON           // codec for JSON
	Message       = websocket.Message        // codec for string, []byte
	ActiveClients = make(map[ClientConn]struct{}) // map containing clients

	ClientHandler = SocketHandler(make(map[*Game]gameClients))
)

// Client connection consists of the websocket and the client ip
type ClientConn struct {
	websocket *websocket.Conn
	clientIP  string
	inputChan *chan *GameMessage
}

func init() {
	http.Handle("/client", websocket.Handler(SockServer))
}

type gameClients map[ClientConn]struct{}

type SocketHandler map[*Game]gameClients

// TODO: error returned?
func (handler SocketHandler) Broadcast(game *Game, msg *GameMessage) {
	clients := handler[game]

	for client, _ := range clients {
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


// WebSocket server to handle chat between clients
func SockServer(ws *websocket.Conn) {
	var game *Game

	clientIP := ws.Request().RemoteAddr
	inputChan := make(chan *GameMessage)
	sockCli := ClientConn{ws, clientIP, &inputChan}

	// for loop so the websocket stays open otherwise
	// it'll close after one Receieve and Send

	request := ws.Request()
	if sess, err := session.GetGameSession(request); err != nil {
		log.Println("there is no game associated to this session, wad?")
		return
	} else {
		game, _ = sess.GetGame()
	}

	// Register client connection in global ClientHandler
	ClientHandler.NewConnection(game, sockCli)

	log.Println("client connect ...", clientIP)

	// cleanup on server side
	defer func() {
		ClientHandler.LazyRemoveClient(game, sockCli)

		if err := ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}
	}()

	for {
		select {
		case msg := <-inputChan:
			log.Printf("attempting to send message %#v\n", msg)
			res, err := json.Marshal(msg)

			if err != nil {
				log.Fatal(err)
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


