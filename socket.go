package main

import (
	"code.google.com/p/go.net/websocket"
	"log"
	"net/http"
	"encoding/json"
)

const (
	listenAddr = "localhost:8080" // server address
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
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("templates/js"))))
}

// WebSocket server to handle chat between clients
func SockServer(ws *websocket.Conn) {
	var err error

	// cleanup on server side
	defer func() {
		if err = ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}
	}()

	client := ws.Request().RemoteAddr
	log.Println("Client connected:", client)
	sockCli := ClientConn{ws, client}
	ActiveClients[sockCli] = 0
	log.Println("Number of clients connected ...", len(ActiveClients))

	// for loop so the websocket stays open otherwise
	// it'll close after one Receieve and Send
	for {
		select {
			case msg := <- VisitChannel:
				res, _ := json.Marshal(msg)
				for cs, _ := range ActiveClients {
					if err = Message.Send(cs.websocket, string(res)); err != nil {
						// we could not send the message to a peer
						log.Println("Could not send message to ", cs.clientIP, err.Error(), " - dropping client.")
						delete(ActiveClients, sockCli)
					}
				}
		}		
	}
}