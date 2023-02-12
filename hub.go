package main

import "log"

type message struct {
	data []byte
	ID   string
}

type Hub struct {
	// Registered clients.
	// clients map[*Client]bool
	clients map[string]*Client
	// Inbound messages from the clients.
	send chan message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	connections map[string]*connection
}

// var h = hub{
// 	broadcast:   make(chan message),
// 	register:    make(chan subscription),
// 	unregister:  make(chan subscription),
// 	connections: make(map[string]map[*connection]bool),
// }

var h = newHub()

func newHub() *Hub {
	return &Hub{
		send:       make(chan message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		// clients:     make(map[*Client]bool),
		clients:     make(map[string]*Client),
		connections: make(map[string]*connection),
	}
}

func (h *Hub) run() {
	// for {
	// 	select {
	// 	case s := <-h.register:
	// 		connections := h.rooms[s.room]
	// 		if connections == nil {
	// 			connections = make(map[*connection]bool)
	// 			h.rooms[s.room] = connections
	// 		}
	// 		h.rooms[s.room][s.conn] = true
	// 	case s := <-h.unregister:
	// 		connections := h.rooms[s.room]
	// 		if connections != nil {
	// 			if _, ok := connections[s.conn]; ok {
	// 				delete(connections, s.conn)
	// 				close(s.conn.send)
	// 				if len(connections) == 0 {
	// 					delete(h.rooms, s.room)
	// 				}
	// 			}
	// 		}
	// 	case m := <-h.broadcast:
	// 		connections := h.rooms[m.room]
	// 		for c := range connections {
	// 			select {
	// 			case c.send <- m.data:
	// 			default:
	// 				close(c.send)
	// 				delete(connections, c)
	// 				if len(connections) == 0 {
	// 					delete(h.rooms, m.room)
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	for {
		select {
		case client := <-h.register:
			log.Println("registering  client")
			h.clients[client.ID] = client
		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				log.Println("unregistering  client")
				delete(h.clients, client.ID)
				close(client.send)
			}
		case message := <-h.send:
			if client, ok := h.clients[message.ID]; ok {
				select {
				case client.send <- message.data:
				default:
					log.Println("closing connections  client")
					close(client.send)
					delete(h.connections, client.ID)
				}
			}
		}
	}
}
