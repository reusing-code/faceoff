package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/gorilla/websocket"
)

func ServeWs(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	c := make(chan bool)
	connection := &websocketConnection{ws: ws, c: c, key: key}
	storeConnection(connection)
	go writer(connection)
	reader(connection)
}

func TriggerUpdate(key string) {
	values := openConnections[key]
	if values == nil {
		return
	}
	for _, conn := range values {
		conn.c <- true
	}
}

var openConnections map[string][]*websocketConnection = make(map[string][]*websocketConnection)

type websocketConnection struct {
	key string
	ws  *websocket.Conn
	c   chan bool
}

const (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 8) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func storeConnection(conn *websocketConnection) {
	values := openConnections[conn.key]
	if values == nil {
		values = make([]*websocketConnection, 0)
	}
	values = append(values, conn)
	openConnections[conn.key] = values
}

func removeConnection(conn *websocketConnection) {
	values := openConnections[conn.key]
	if values == nil {
		return
	}
	for i, existingConn := range values {
		if conn == existingConn {
			// https://github.com/golang/go/wiki/SliceTricks
			copy(values[i:], values[i+1:])
			values[len(values)-1] = nil
			values = values[:len(values)-1]
		}
	}
}

func writer(conn *websocketConnection) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		conn.ws.Close()
		removeConnection(conn)
	}()
	for {
		select {
		case <-pingTicker.C:
			conn.ws.SetWriteDeadline(time.Now().Add(pongWait))
			if err := conn.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		case <-conn.c:
			conn.ws.SetWriteDeadline(time.Now().Add(pongWait))
			if err := conn.ws.WriteMessage(websocket.TextMessage, []byte("refresh")); err != nil {
				return
			}
		}

	}
}

func reader(conn *websocketConnection) {
	defer conn.ws.Close()
	conn.ws.SetReadLimit(512)
	conn.ws.SetReadDeadline(time.Now().Add(pongWait))
	conn.ws.SetPongHandler(func(string) error {
		conn.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	conn.ws.SetCloseHandler(func(code int, text string) error {
		fmt.Printf("WS %p closed with code %d: %s\n", conn.ws, code, text)
		conn.ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := conn.ws.ReadMessage()
		if err != nil {
			break
		}
	}
}
