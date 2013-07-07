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
	ActiveClients = make(map[ClientConn]int) // map containing clients
)

// Client connection consists of the websocket and the client ip
type ClientConn struct {
	websocket *websocket.Conn
	clientIP  string
}

func init() {
	http.Handle("/client", websocket.Handler(SockServer))
}

// WebSocket server to handle chat between clients
func SockServer(ws *websocket.Conn) {
	var err error
	var game *Game

	// cleanup on server side
	defer func() {
		if err = ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}
	}()

	client := ws.Request().RemoteAddr
	log.Println("client connect ...", client)
	sockCli := ClientConn{ws, client}
	ActiveClients[sockCli] = 0

	// for loop so the websocket stays open otherwise
	// it'll close after one Receieve and Send

	request := ws.Request()
	if sess, err := session.GetGameSession(request); err != nil {
		log.Println("there is no game associated to this session, wad?")
	} else {
		game, _ = sess.GetGame()
	}

	log.Println(game.GetChannel())

	for {
		select {
		case msg := <-game.GetChannel():
			log.Printf("attempting to send message %#v\n", msg)
			res, err := json.Marshal(msg)

			if err != nil {
				log.Fatal(err)
			}

			for cs, _ := range ActiveClients {
				if err = Message.Send(cs.websocket, string(res)); err != nil {
					// we could not send the message to a peer
					log.Println("Could not send message to ",
						cs.clientIP, err.Error(), " - dropping client.")
					delete(ActiveClients, sockCli)
				}
			}
		}
	}
}
