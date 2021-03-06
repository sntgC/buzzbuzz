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
//Each host acts as a collection of clients and transmits and controls the messages between them
type Host struct {
	// Registered clients.
	clients map[string]*Client

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
		clients:    make(map[string]*Client),
        id: randID(5),
        conn: c,
        listening: true,
        teamReg: make(chan *Team),
        teams: make(map[string]*Team),
        lastBuzz:nil,
	}
}

//Action regex templates
var team=regexp.MustCompile("team/(reset|create|remove)/(.+)")
var score=regexp.MustCompile(`score/(last|custom)/(\d+)`)
var playerControl=regexp.MustCompile(`player/(kick)/(.+)`)

/*
Host message codes:
    0 - Client event
        j - join
        l - leave
        b - buzz
    1 - reset buzzer
    2 - Team event
        c - create
        l - leave
        u - update score
    4 - Player score event
*/

//Basically equivalent to the readPump function of the client class
func (h *Host) control(){
    defer func() {
        log.Println("CONTROL EXITED")
		h.end()
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
        if string(message)=="reset"{
            h.listening=true
            h.broadcast<-[]byte("1 reset")
        }
        if d:=team.FindStringSubmatch(string(message));d!=nil{
        	if d[1]=="reset"{
				if t,ok:=h.teams[d[2]];ok{
					t.muted=false
				}
			}else if d[1]=="create"{
				newTeam(h,d[2])
			}else if d[1]=="remove"{
				if t,ok:=h.teams[d[2]];ok{
					t.destroy()
				}
			}
        }
        if d:=score.FindStringSubmatch(string(message));d!=nil{
            if h.lastBuzz!=nil {
                sc,err:=strconv.Atoi(d[2])
                if err==nil {
                    h.lastBuzz.score+=sc
                }
                if team:=h.lastBuzz.team;team!=nil{
                    h.broadcast<-[]byte("4 "+h.lastBuzz.id+" "+strconv.Itoa(h.lastBuzz.score)+" "+team.id)
                    team.score+=sc
                    h.broadcast<-[]byte("2 "+team.id+" u "+team.name+" "+strconv.Itoa(team.score))

                }else{
                    h.broadcast<-[]byte("4 "+h.lastBuzz.id+" "+strconv.Itoa(h.lastBuzz.score))
                }
            }
            
        }
        if d:=playerControl.FindStringSubmatch(string(message));d!=nil {
			if d[1] == "kick" {
				if c, ok := h.clients[d[2]]; ok {
					h.unregister<-c
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
    for _,v:=range h.clients{
        v.send<-[]byte(msg)
    }
}

func (h *Host) end(){
	h.conn.Close()
	delete(rooms, h.id)
	for _,v:=range h.clients{
		v.conn.Close()
	}
}

func (h *Host) run() {
    ticker := time.NewTicker(pingPeriod)
    h.sendMessage(h.id)
    defer func() {
		ticker.Stop()
		h.end()
	}()
   go h.control()
	for {
		select {
		case client := <-h.register:
            msg := "0 "+client.id+" j "+client.name+" "+strconv.Itoa(client.score)
			h.clients[client.id] = client
            h.sendMessage(msg)
            h.sendAll(msg)
		case client := <-h.unregister:
			if _, ok := h.clients[client.id]; ok {
				delete(h.clients, client.id)
				client.conn.Close()
                msg := "0 "+client.id+" l"
                h.sendMessage(msg)
				close(client.send)
                h.sendAll(msg)
			}
        case client := <-h.buzzer:
            if h.listening {
                if client.team!=nil {
                            log.Println("Client team is not nil")
                   team,ok:=h.teams[client.team.id]
                    if ok {
                        
                            log.Println("Client team exists")
                        if team.muted {
                            log.Println("Team has buzzed already")
                           break 
                        }
                        team.muted=true
                    }
                }
                h.listening=false
                h.lastBuzz=client
				msg:="0 "+client.id+" b"
                h.sendMessage(msg)
                h.sendAll(msg)
                log.Println(client.name+" BUZZED")
            }
        case team := <-h.teamReg:
        	var msg string
            if t,ok:=h.teams[team.id];ok{
            	delete(h.teams,team.id)
				msg="2 "+t.id+" l "
			}else{
				h.teams[team.id]=team
				msg="2 "+team.id+" c "+team.name+" "+strconv.Itoa(team.score)
			}
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