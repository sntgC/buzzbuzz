// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")
var rooms map[string]*Host


func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
        if r.URL.Path=="/beep.mp3"{
            http.ServeFile(w,r,"beep.mp3")
            return
        }else if r.URL.Path=="/style.css"{
            http.ServeFile(w,r,"style.css")
            return
        }
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {
	flag.Parse()
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        host := newHost(w,r)
        rooms[host.id]=host
        go host.run()
	})
    rooms =  make(map[string]*Host)
    http.HandleFunc("/client", func(w http.ResponseWriter, r *http.Request) {
        roomID:=r.URL.Query().Get("roomID")
        name:=r.URL.Query().Get("name")
        h,ok:=rooms[roomID]
        if(!ok){
            log.Println("Room not found")
        }else{
            serveWs(h,w,r,name)
        }
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}