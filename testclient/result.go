package testclient

import (
	"sort"
	"fmt"
)

//ResultHandler manages the result rooms and the interactions between the results of different clients.
type ResultHandler struct {
	join chan *joinrq
	list chan chan []string
}

//NewResultHandler creates a new ResultHandler.
func NewResultHandler() *ResultHandler {
	rh := new(ResultHandler)
	rh.join = make(chan *joinrq)
	rh.list = make(chan chan []string)
	go rh.roomManager()
	return rh
}

//Result is an object for keeping track of what responses the client should be getting from the server.  Result.Results contains all of the messages that should have been sent to the client.
type Result struct {
	Results []string
	join chan *joinrq
	client *roomClient
	room *room
	quit chan chan string
	blocklist []string
	list chan chan []string
}


//NewResult creates a new result object without a chattest.
func NewResult(name string, rh *ResultHandler) *Result {
	r := new(Result)
	r.join = rh.join
	r.client = &roomClient{name,make(chan *rMessage), make(chan bool), make(chan bool)}
	r.Results = make([]string, 0,10)
	r.blocklist = make([]string, 0, 10)
	r.list = rh.list
	go r.update()
	return r
}

//IsBlocked checks to see if name is in the result blocklist.
func (r *Result) IsBlocked(name string) bool {
	for i := 0; i<len(r.blocklist); i++{
		if r.blocklist[i] == name {
			return true
		}
	}
	return false
}

//List udpates the result for a list action, adding the list of current rooms in the Result.
func (r *Result) List() {
	resp := make(chan []string)
	r.list<-resp
	rlist := <-resp
	r.Add("Rooms:")
	for i := range rlist {
		r.Add(rlist[i])
	}
}

//Block adds name to the Result's blocklist.
func (r *Result) Block(name string) {
	if r.IsBlocked(name) {
		return
	}
	if name == r.client.name {
		return
	}
	r.blocklist = append(r.blocklist, name)
}

//UnBlock removes the name from the Result's blocklist.
func (r *Result) UnBlock(name string) bool{
	found := false
	for i:=0; i<len(r.blocklist);i++ {
		if r.blocklist[i] == name {
			r.blocklist = append(r.blocklist[:i], r.blocklist[i+1:]...)
			i--
			found = true
		}
	}
	return found
}

//Who updates the Result for a who action.  It adds the list of people in the specified room in the Result.
func (r *Result) Who(name string) {
	var rm *room
	if name == "" {
		if r.room == nil {
			r.Add("You're not in a room.  Type /join roomname to join a room or /help for other commands.")
			return
		}
		rm = r.room
	}else {
		rm = r.GetRoom(name)
	}
	if rm == nil {
		r.Add("Room not Found")
		return
	}
	r.Add(fmt.Sprintf("Room: %v", name))
	clist := make([]string,0,10)
	for i := range rm.clients {
		clist = append(clist,rm.clients[i].name)
	}
	sort.Strings(clist)
	for i := range clist {
		r.Add(clist[i])
	}
}

//RestGet updates the Result for a rest client getting messages.
func (r *Result) RestGet(name string) {
	var rm *room
	if name == "" {
		return
	}
	rm = r.GetRoom(name)
	if rm == nil {
		r.Add("Room not Found")
		return
	}
	for i := range rm.messages {
		r.Add(rm.messages[i])
	}
}

//RestSend updates all results for a rest client sending a message.
func (r *Result) RestSend(message,room string) {
	m := NewRMessage(r.client.name, message, r.client.done)
	rm := r.GetRoom(room)
	if rm == nil {
		r.Add("Room not Found")
		return
	}
	rm.in<-m
	<-r.client.done
}

//Room returns the name of the room the Result is currently "in."
func (r *Result) Room() string {
	return r.room.name
}

//Join changes the results room to the with name == rm and updates its channels.
func (r *Result) Join(rm string) {
	if r.room != nil {//leave previous room
		r.room.quit<-r.client
		<-r.client.done
	}
	j := newJoinRQ(rm, r.client)
	r.join <- j
	r.room = <-j.response
}

//Send adds the string to the results of everyone in the same room as the sender.
func (r *Result) Send(text string) {
	m := NewRMessage(r.client.name, text, r.client.done)
	r.room.in<-m
	<-r.client.done
}

//JoinSend adds the string to the results of everyone in the same room as the sender even if blocking the sender.  Primarily for use with join room messages.
func (r *Result) JoinSend(text string) {
	m := NewRMessage("", text, r.client.done)
	r.room.in<-m
	<-r.client.done
}

//update takes messages from the room and adds them to the Result's Results.
func (r *Result) update() {
	for{
		m := <-r.client.channel
		if !r.IsBlocked(m.name){
			r.Results = append(r.Results, m.text)
		}
		r.client.updatedone<-true
	}
}

//Add adds the string to the Result of only this Result not the others in the same room.
func (r *Result) Add(text string) {
	r.Results = append(r.Results, text)
}

//GetRoom returns a room with name matching rm.  Unlike join it will not create one if it doesn't exist, returning nil instead.
func (r *Result) GetRoom(rm string) *room {
	j := newJoinRQ(rm, nil)
	r.join <- j
	return <-j.response
}

//rMessage is a type used for sending messages on the channel to a room.  It includes a done channel for the room to signal that transmission is complete.
type rMessage struct {
	name string
	text string
	done chan bool
}

//NewRMessage creates a new rMessage.
func NewRMessage (name string, text string, done chan bool)*rMessage{
	rm := new(rMessage)
	rm.name = name
	rm.text = text
	rm.done = done
	return rm
}

//joinrq is a type for sending join requests to the roomhandler.  responnse is the channel on which the room joined will be sent back.
type joinrq struct {
	room string
	client *roomClient
	response chan *room
}

//newJoinRQ returns a new joinrq.
func newJoinRQ(rm string, cl *roomClient) *joinrq {
	j := new(joinrq)
	j.room = rm
	j.client = cl
	j.response = make(chan *room)
	return j
}

//roomClient is a type used by the room to keep track of who is in it for message sending purposes.
type roomClient struct {
	name string
	channel chan *rMessage
	done chan bool
	updatedone chan bool
}

//room is a type used to keep track of the servers rooms for updating results.
type room struct {
	name string
	clients []*roomClient
	addclient chan *joinrq
	in chan *rMessage
	quit chan *roomClient
	messages []string
}

//newRoom returns a new room.
func newRoom (name string) *room {
	rm := new(room)
	rm.name = name
	rm.clients = make([]*roomClient,0,10)
	rm.addclient = make(chan *joinrq)
	rm.in = make(chan *rMessage)
	rm.quit = make(chan *roomClient)
	rm.messages = make([]string, 0, 10)
	go rm.run()
	return rm
}

//addCl adds a client to the room.
func (rm *room) addCl(cl *roomClient) {
	rm.clients = append(rm.clients, cl)
}

//remCl removes a client from the room.
func (rm *room) remCl(cl *roomClient) {
	for i := 0; i<len(rm.clients);i++ {
		if rm.clients[i].channel == cl.channel {
			rm.clients[i]= nil
			rm.clients = append(rm.clients[:i], rm.clients[i+1:]...)
			i--
		}
	}
}

//who returns a list of the names of the clients in the room.
func (rm *room) who() []string {
	list := make([]string,0,10)
	for _, n := range rm.clients {
		list = append(list, n.name)
	}
	sort.Strings(list)
	return list
}

//run is started when a new room is created and handles the message transmiting, joining, and leaving the room.
func (rm *room) run() {
	var m *rMessage
	for {
		select {
			case m = <-rm.in:
				for i := range rm.clients {
					rm.clients[i].channel <- m
					<-rm.clients[i].updatedone
				}
				rm.messages = append(rm.messages, m.text)
				m.done<-true
			case cl := <-rm.quit:
				rm.remCl(cl)
				cl.done<-true
			case j := <-rm.addclient:
				rm.clients = append(rm.clients,j.client)
				j.response<-rm
		}
	}
}

//getroom returns a room with name from the list.
func getroom (name string, l []*room) *room {
	for i := range l {
		if l[i].name == name {
			return l[i]
		}
	}
	return nil
}

//roomManager is started when a new test is created and handles the creation of rooms, the roomlist, and sending clients to the right room.
func (rh *ResultHandler) roomManager () {
	roomlist := make([]*room,0,0)
	for {
		select{
			case j := <-rh.join:
				rm := getroom(j.room, roomlist)
				if rm == nil && j.client != nil{
					rm = newRoom(j.room)
					roomlist = append(roomlist,rm)
				}
				if j.client != nil {
					rm.addclient<- j
				} else {
					j.response<-rm
				}
			case r := <-rh.list:
				rlist := make([]string, 0, 0)
				for i:= range roomlist {
					rlist = append(rlist, roomlist[i].name)
				}
				sort.Strings(rlist)
				r<-rlist
		}
	}
}

