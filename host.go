package main

import (
    "math/rand"
    "time"
	"net/http"
    "log"
    "bytes"
    "regexp"
    "strconv"
	"github.com/gorilla/websocket"
)
// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Host struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte
    teamReg chan *Team
    buzzer chan *Client
    
	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
    
    id string 
    
	conn *websocket.Conn
    
    listening bool
    teams map[string] *Team
    lastBuzz *Client
}

func randID(length int) string{
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	out :=""
	valid :="qwertyuiopasdfghjklzxcvbnm1234567890"
	for i:=0;i<length;i++{
		out+=string(valid[r1.Intn(len(valid))])
	}
	return out
}

func newHost(w http.ResponseWriter, r *http.Request) *Host {
    c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	return &Host{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
        buzzer: make(chan *Client),
		clients:    make(map[*Client]bool),
        id: randID(5),
        conn: c,
        listening: true,
        teamReg: make(chan *Team),
        teams: make(map[string]*Team),
        lastBuzz:nil,
	}
}
var resetTeam=regexp.MustCompile("treset/(.+)")
var score=regexp.MustCompile(`score/(last|custom)/(\d+)`)
func (h *Host) control(){
    defer func() {
        log.Println("CONTROL FAILED")
		h.conn.Close()
        delete(rooms, h.id)
	}()
	h.conn.SetReadLimit(maxMessageSize)
    h.conn.SetReadDeadline(time.Now().Add(pongWait))
    h.conn.SetPongHandler(func(string) error { h.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := h.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
            log.Println("ERROR")
            log.Println(err)
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
        log.Println(message)
        if string(message)=="reset"{
            h.listening=true
            h.broadcast<-[]byte("1 reset")
        }
        if d:=resetTeam.FindStringSubmatch(string(message));d!=nil{
            if t,ok:=h.teams[d[1]];ok{
                t.muted=false
            }
        }
        if d:=score.FindStringSubmatch(string(message));d!=nil{
            if(h.lastBuzz!=nil){
                sc,err:=strconv.Atoi(d[2])
                if(err==nil){
                    h.lastBuzz.score+=sc
                }
                if team:=h.lastBuzz.team;team!=nil{
                    h.broadcast<-[]byte("4 "+h.lastBuzz.id+" "+strconv.Itoa(h.lastBuzz.score)+" "+team.id)
                }else{
                    h.broadcast<-[]byte("4 "+h.lastBuzz.id+" "+strconv.Itoa(h.lastBuzz.score))
                }
            }
            
        }
	}
}

func (h *Host) sendMessage(msg string){
    
    h.conn.SetWriteDeadline(time.Now().Add(writeWait))
    nw, err := h.conn.NextWriter(websocket.TextMessage)
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

func (h *Host) sendAll(msg string){
    for k,_:=range h.clients{
        k.send<-[]byte(msg)
    }
}

func (h *Host) run() {
    ticker := time.NewTicker(pingPeriod)
    h.sendMessage(h.id)
    defer func() {
		ticker.Stop()
		h.conn.Close()
        log.Println("Run function ended")
	}()
   go h.control()
	for {
		select {
		case client := <-h.register:
            msg := "0 "+client.id+" j "+client.name
            h.sendMessage(msg)
            h.sendAll(msg)
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
                msg := "0 "+client.id+" l"
                h.sendMessage(msg)
				close(client.send)
                h.sendAll(msg)
			}
        case client := <-h.buzzer:
            if(h.listening){
                if(client.team!=nil){
                            log.Println("Client team is not nil")
                   team,ok:=h.teams[client.team.id]
                    if(ok){
                        
                            log.Println("Client team exists")
                        if(team.muted){
                            log.Println("Team has buzzed already")
                           break 
                        }
                        team.muted=true
                    }
                }
                h.listening=false
                h.lastBuzz=client;
                msg:="0 "+client.id+" b"
                h.sendMessage(msg)
                h.sendAll(msg)
                log.Println(client.name+" BUZZED")
            }
        case team := <-h.teamReg:
            h.teams[team.id]=team
            msg:="2 "+team.id+" c "+team.name
            h.sendMessage(msg)
            h.sendAll(msg)
		case message := <-h.broadcast:
            h.sendMessage(string(message))
            h.sendAll(string(message))
        case <-ticker.C:
            h.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := h.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
        
	}
}