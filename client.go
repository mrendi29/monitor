package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Client struct {
	// unique ID for each client
	// id string
	ID string
	mu sync.RWMutex

	// Hub object
	// hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// connection --> (what should the connection property be?)
	// conn_ID string
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// connection is an middleman between the websocket connection and the hub.
type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
func (client *Client) readPump() {
	c := client.conn
	defer func() {
		h.unregister <- client
		log.Println("readPump closing connection")
		c.Close()
	}()

	c.SetReadLimit(maxMessageSize)
	c.SetReadDeadline(time.Now().Add(pongWait))
	c.SetPongHandler(func(string) error { c.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}

		log.Printf("recv: %s %d", msg, mt)

		h.send <- message{ID: client.ID, data: msg}
	}
}

// write writes a message with the given message type and payload.
func (client *Client) write(mt int, payload []byte) error {
	client.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return client.conn.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (client *Client) writePump() {
	// c := client.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Println("writePump Closing connection")
		ticker.Stop()
		client.conn.Close()
	}()
	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				client.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := client.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	vars := mux.Vars(r)
	log.Println(vars["client_id"])
	client := &Client{conn: ws, ID: vars["client_id"], send: make(chan []byte, 256)}
	h.register <- client

	go client.writePump()
	client.readPump()

}
