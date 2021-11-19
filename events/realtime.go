package events

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/inconshreveable/log15"
)

type Message struct {
	Data      interface{} `json:"data"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = pongWait / 2

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// IIS breaks with compression, apparently?
	EnableCompression: false,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	log.Info("ws client connected", "address", r.RemoteAddr)
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err.Error())
		return
	}
	// ws.EnableWriteCompression(true)
	c := &Client{send: make(chan []byte, 256), ws: ws, hub: hub}
	hub.Register <- c
	go c.writePump()
	go c.readPump()
}
