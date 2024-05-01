/*
```
Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file.
```
// refs https://github.com/gorilla/websocket/tree/main/examples/chat
// LICENSE: https://github.com/gorilla/websocket/blob/main/LICENSE
*/
package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gorilla/websocket"
)

var (
	hub            *Hub
	maxMessageSize int64 // Maximum message size allowed from peer.
	pingInterval   int64
	writeWait      time.Duration
	pingPeriod     time.Duration
	pongWait       time.Duration

	newline = []byte{'\n'}
	space   = []byte{' '}

	//go:embed websocket.html
	chatHtml string
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// client identifier
	id string

	// closed flag
	closed bool
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				if !client.closed {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}

func chatPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/chat/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Printf("chat RemoteAddr: %v, X-Forwarded-For: %v\n", r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(chatHtml)))
	fmt.Fprint(w, chatHtml)
}

// wsHandler handles websocket requests from the peer.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrader.Upgrade error: %v", err)
		return
	}
	fmt.Printf("openWS RemoteAddr: %v, X-Forwarded-For: %v\n", r.RemoteAddr, r.Header.Get("X-Forwarded-For"))

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), id: getClientID(r, conn)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

func getClientID(r *http.Request, conn *websocket.Conn) (clientID string) {
	clientID = fmt.Sprintf("%s, %s", r.RemoteAddr, conn.LocalAddr())
	if r.Header.Get("X-Forwarded-For") != "" {
		clientID = fmt.Sprintf("%s, %s", r.Header.Get("X-Forwarded-For"), clientID)
	}
	return strings.Replace(clientID, " ", "", -1)
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
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
			log.Printf("c.conn.ReadMessage error: %v", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket.IsUnexpectedCloseError error: %v", err)
			}
			// notification connection closed
			c.closed = true
			c.hub.broadcast <- []byte(fmt.Sprintf("Closed Connection [%s]", c.id))
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.hub.broadcast <- message
		log.Printf("msg from: %s, message: %s", c.id, message)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			log.Printf("msg   to: %s, message: %s", c.id, message)
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				log.Println("get message from c.send not ok")
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("c.send triggered c.conn.NextWriter error: %v", err)
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				log.Printf("c.send triggered w.Close error: %v", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			log.Printf("ping  to: %s", c.id)
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("ticker triggered c.conn.WriteMessage error: %v", err)
				return
			}
		}
	}
}
