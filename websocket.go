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
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var (
	hub            *Hub
	maxMessageSize int64 // Maximum message size allowed from peer.
	wsInterval     int64
	writeWait      time.Duration
	pingPeriod     time.Duration
	pongWait       time.Duration

	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WsData is struct(json) of WebSocket communication.
type WsData struct {
	Type      string `json:"type"`
	Message   string `json:"message,omitempty"`
	SendTime  int64  `json:"sendTime,omitempty"`
	ConnCount int    `json:"connCount,omitempty"`
	User      User   `json:"user,omitempty"`
}

// User is part of UserList
type User struct {
	ClientID string `json:"clientId,omitempty"`
	HostIP   string `json:"hostIp,omitempty"`
	Color    string `json:"color,omitempty"`
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// client identifier
	id    string
	color string

	// closed flag
	closed bool
}

// Hub maintains the set of active clients and broadcasts messages to the clients.
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

// wsHandler handles websocket requests from the peer.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	logger := wsLogger(r)
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log().Str("error", fmt.Sprintf("upgrader.Upgrade error: %v", err)).Msg("")
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), id: getClientID(r, conn), color: getRandomColor()}
	client.hub.register <- client

	logger.Log().Str("color", client.color).Msg("connected")

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump(logger)
	go client.readPump(logger)
}

func getClientID(r *http.Request, conn *websocket.Conn) (clientID string) {
	clientID = fmt.Sprintf("%s, %s", r.RemoteAddr, conn.LocalAddr())
	if r.Header.Get("X-Forwarded-For") != "" {
		clientID = fmt.Sprintf("%s, %s", r.Header.Get("X-Forwarded-For"), clientID)
	}
	return strings.Replace(clientID, " ", "", -1)
}

var defColors = []string{"aqua", "black", "blue", "fuchsia", "gray", "green", "lime", "maroon", "navy", "olive", "purple", "silver", "teal", "white", "yellow"}

// Do not duplicate color selections until all color definitions are used
func getRandomColor() string {
	var usedColors []string
	for c := range hub.clients {
		if !slices.Contains(usedColors, c.color) {
			usedColors = append(usedColors, c.color)
		}
	}
	diffColors := diffSlice(defColors, usedColors)
	if len(diffColors) == 0 {
		randIdx := rand.Intn(len(defColors))
		return defColors[randIdx]
	}
	randIdx := rand.Intn(len(diffColors))
	return diffColors[randIdx]
}

// subtraction between arrays
func diffSlice[T comparable](slice1, slice2 []T) []T {
	diffSlice := []T{}
	cmpMap := map[T]int{}

	for _, v := range slice2 {
		cmpMap[v]++
	}

	for _, v := range slice1 {
		t, ok := cmpMap[v]
		if !ok {
			diffSlice = append(diffSlice, v)
			continue
		}
		if t == 1 {
			delete(cmpMap, v)
		} else {
			cmpMap[v]--
		}
	}
	return diffSlice
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump(logger *zerolog.Logger) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	var wsInData WsData
	var wsOutData WsData
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log().Time("closetime", time.Now()).
					Str("error", fmt.Sprintf("UnexpectedCloseError: %v", err)).Msg("")
			} else {
				logger.Log().Time("closetime", time.Now()).
					Str("error", fmt.Sprintf("ExpectedCloseError: %v", err)).Msg("")
			}
			// notification connection closed
			wsOutData.Type = "deliverMessage"
			wsOutData.Message = fmt.Sprintf("Disconnected due to [%v]", err)
			wsOutData.SendTime = time.Now().UTC().UnixNano() / int64(time.Millisecond)
			wsOutData.ConnCount = len(c.hub.clients) - 1
			wsOutData.User.ClientID = c.id
			wsOutData.User.Color = c.color

			message = convertWsData2JSON(wsOutData)
			c.closed = true
			c.hub.broadcast <- message
			break
		} else {
			logger.Log().Time("readtime", time.Now()).Int("msgsize", len(string(message))).Msg("")
		}

		if err := json.Unmarshal(message, &wsInData); err != nil {
			c.send <- message
		} else {
			if wsInData.Type == "whoAmI" {
				// reply connection info to the user
				wsOutData.Type = "yourInfo"
				wsOutData.User.ClientID = c.id
				wsOutData.User.HostIP = store.host.IP
				wsOutData.User.Color = c.color
				message = convertWsData2JSON(wsOutData)
				c.send <- message

				// send connection opened message to all users
				wsOutData.Type = "deliverMessage"
				wsOutData.Message = "Connection opened."
				wsOutData.SendTime = time.Now().UTC().UnixNano() / int64(time.Millisecond)
				wsOutData.ConnCount = len(c.hub.clients)
				wsOutData.User.ClientID = c.id
				wsOutData.User.Color = c.color
				message = convertWsData2JSON(wsOutData)
				c.hub.broadcast <- message
			} else {
				wsOutData.Message = wsInData.Message
				wsOutData.SendTime = time.Now().UTC().UnixNano() / int64(time.Millisecond)
				wsOutData.ConnCount = len(c.hub.clients)
				wsOutData.User.ClientID = c.id
				wsOutData.User.Color = c.color
				if wsInData.Type == "postToChat" {
					// send chat message to all users
					wsOutData.Type = "deliverMessage"
					message = convertWsData2JSON(wsOutData)
					c.hub.broadcast <- message
				} else if wsInData.Type == "echoMessage" {
					// send echo message to the user
					wsOutData.Type = "echoReply"
					message = convertWsData2JSON(wsOutData)
					c.send <- message
				}
			}
		}
	}
}

func convertWsData2JSON(wsData WsData) []byte {
	wsDataJSON, _ := json.Marshal(wsData)
	respJSON := bytes.TrimSpace(bytes.Replace(wsDataJSON, newline, space, -1))
	return respJSON
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(logger *zerolog.Logger) {
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
				logger.Log().Time("closetime", time.Now()).
					Str("error", fmt.Sprintf("conn.NextWriter error: %v", err)).Msg("")
				// notification connection closed
				c.closed = true
				c.hub.broadcast <- []byte(fmt.Sprintf("Disconnected due to [%v] in NextWriter", err))
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
				logger.Log().Time("closetime", time.Now()).
					Str("error", fmt.Sprintf("conn.NextWriter.Close error: %v", err)).Msg("")
				// notification connection closed
				c.closed = true
				c.hub.broadcast <- []byte(fmt.Sprintf("Disconnected due to [%v] in NextWriter.Close", err))
				return
			}
			logger.Log().Time("writetime", time.Now()).Int("msgsize", len(string(message))).Msg("")

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Log().Time("closetime", time.Now()).
					Str("error", fmt.Sprintf("c.conn.WriteMessage error: %v", err)).Msg("")
				// notification connection closed
				c.closed = true
				c.hub.broadcast <- []byte(fmt.Sprintf("Disconnected due to [%v] in WriteMessage", err))
				return
			}
			//logger.Log().Time("pingtime", time.Now()).Msg("")

		}
	}
}
