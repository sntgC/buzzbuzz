package main


type Team struct {
	members map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte
    
	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
    
    id string 
    
    score int
    
    name string
    
    muted bool
}

func newTeam(host *Host,name string) *Team {
    team:=&Team{
        make(map[*Client]bool),
        make(chan []byte),
        make(chan *Client),
        make(chan *Client),
        randID(10),
        0,
        name,
        false,
	}
    host.teamReg<-team
    return team
}