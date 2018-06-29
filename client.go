// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"log"
	"net/http"
	"strconv"
	"time"
    "regexp"
	"github.com/gorilla/websocket"
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

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	host *Host

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
    
    name string
    id string
    team *Team
    score int
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
var teamRequest = regexp.MustCompile(`(team)/(join|create)/(.+)`)
var buzzRequest = regexp.MustCompile(`Buzz`)
func (c *Client) readPump() {
	defer func() {
		c.host.unregister <- c
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
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
        //log.Println(message)
        switch{
            case buzzRequest.MatchString(string(message)):
                c.host.buzzer <- c
            case teamRequest.MatchString(string(message)):
                dat:=teamRequest.FindStringSubmatch(string(message))
                if dat[2]=="create" {
                    team:=newTeam(c.host,dat[3])
                    c.joinTeam(team)
                }else if dat[2]=="join" {
                    c.joinTeam(c.host.teams[dat[3]])
                }
        }
	}
}

func (c *Client) joinTeam(team *Team){
    team.members[c]=true
    msg:="3 "+c.id+" j "+team.id
    c.team=team
    c.host.broadcast<-[]byte(msg)
}

func (c *Client) sendMessage(msg string){
    c.conn.SetWriteDeadline(time.Now().Add(writeWait))
    nw, err := c.conn.NextWriter(websocket.TextMessage)
    if err != nil {
        log.Println(err)
        return
    }
    nw.Write([]byte(msg))
    if err := nw.Close(); err != nil {
        log.Println(err)
				return
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
    for _,v :=range c.host.clients{
        c.sendMessage("0 "+v.id+" j "+v.name+" "+strconv.Itoa(v.score))
    }
    for _,team :=range c.host.teams{
        c.sendMessage("2 "+team.id+" c "+team.name+" "+strconv.Itoa(team.score))
    }
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

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

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
func serveWs(host *Host, w http.ResponseWriter, r *http.Request,name string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
    client := &Client{host: host, conn: conn, send: make(chan []byte, 256),name:name,id:randID(16),team:nil,score:0}
    client.host.register <- client
	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}