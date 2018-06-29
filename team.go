package main


type Team struct {
	members map[*Client]bool


    
    id string 
    
    score int
    
    name string
    
    muted bool
    host *Host
}

func (t *Team) destroy(){
	for k:=range t.members{
		k.team=nil
	}
	t.host.teamReg<-t
}

func newTeam(host *Host,name string) *Team {
    team:=&Team{
        make(map[*Client]bool),
        randID(10),
        0,
        name,
        false,
        host,
	}
    host.teamReg<-team
    return team
}