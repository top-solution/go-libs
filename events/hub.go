package events

import (
	"encoding/json"
	"time"

	log "github.com/inconshreveable/log15"
)

// Hub maintains the set of active connections and broadcasts messages to the
// connections.
type Hub struct {
	// Registered clients listening for data
	connections map[*Client]bool

	// Inbound messages from the connections
	Broadcast chan *Message

	// Register requests from the connections
	Register chan *Client

	// Unregister requests from connections
	Unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:   make(chan *Message),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		connections: make(map[*Client]bool),
	}
}

func (h *Hub) Send(message interface{}, typ string) (err error) {
	m := Message{
		Data:      message,
		Type:      typ,
		Timestamp: time.Now(),
	}
	h.Broadcast <- &m
	return nil
}

func (h *Hub) Run() {
	go h.handleWebsockets()
}

// handleWebsockets handles WS client connections and message delivery
func (h *Hub) handleWebsockets() {
	for {
		select {
		case c := <-h.Register:
			h.connections[c] = true
		case c := <-h.Unregister:
			delete(h.connections, c)
			close(c.send)

		case m := <-h.Broadcast:
			encoded, err := json.Marshal(m)
			if err != nil {
				log.Error("encode message", "err", err)
			}

			for c := range h.connections {
				select {
				case c.send <- encoded:
				default:
					close(c.send)
					delete(h.connections, c)
				}
			}
		}
	}
}
