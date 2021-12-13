package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	uuid "github.com/nu7hatch/gouuid"
	"log"
	"net/http"
	"time"
)

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
	ReadBufferSize:  128,
	WriteBufferSize: 128,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	clientData *ClientData
}

type ClientData struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	SessionId string `json:"sessionId"`
	Method    string `json:"method"`
}

// readMessage reads messages from the websocket connection to the hub.
func (c *Client) readMessage() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}

			// Delete cursor on screen of other users after connection is closed
			data := &ClientData{
				SessionId: c.clientData.SessionId,
				Method:    "leave",
			}
			message, err = json.Marshal(data)
			c.hub.broadcast <- message

			break
		}
		data := ClientData{}
		err = json.Unmarshal(message, &data)
		if err != nil {
			log.Printf("error: %v", err)
		}

		data.SessionId = c.clientData.SessionId
		data.Method = "move"
		message, err = json.Marshal(data)
		c.hub.broadcast <- message
	}
}

// writeMessage writes messages from the hub to the websocket connection.
func (c *Client) writeMessage() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	u, err := uuid.NewV4()
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), clientData: &ClientData{
		SessionId: u.String(),
	}}
	// Add new client to hub
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writeMessage()
	go client.readMessage()
}
