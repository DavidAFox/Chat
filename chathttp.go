package main

import (
	"net/http"
	"container/list"
	"time"
	"crypto/rand"
	"log"
	"os"
	"fmt"
	"encoding/json"
	"encoding/hex"
	"strings"
	"io"
)

//NewToken creats a new random token string encoded as hex.
func NewToken () string {
	b := make([]byte, 256,256)
	_, err:= rand.Read(b)
	if err != nil {
		log.Println("Error creating token: ",err)
	}
	return hex.EncodeToString(b)
}


//ClientHTTP is a client for the HTTP API
type ClientHTTP struct {
	name string
	Messages *list.List
	Room *Room
	timeOut *time.Timer
	Token string
	Rooms *RoomList
	ChatLog *os.File
	Clients map[string]*ClientHTTP
	Blocked *list.List
}

//log writes the string to the clients ChatLog.
func (cl *ClientHTTP) log (s string) {
	var err error
	_, err = io.WriteString(cl.ChatLog, s +"\n")
	if err != nil {
		log.Println(err)
	}
}
//GetMessage gets all the messages for a client since the last time they were checked and then removes them from their message list.
func (cl *ClientHTTP) GetMessages (w http.ResponseWriter, rq *http.Request) {
	m := make([]string, cl.Messages.Len(), cl.Messages.Len())
	for i,x := cl.Messages.Front(), 0; i != nil; i,x = i.Next(), x+1 {
		m[x] = fmt.Sprint(i.Value)
	}
	for i,x  := cl.Messages.Front(), cl.Messages.Front(); i != nil; {
		x = i
		i = i.Next()
		cl.Messages.Remove(x)
	}
	enc := json.NewEncoder(w)
	err := enc.Encode(m)
	if err != nil {
		log.Println("Error encoding messages: ", err)
	}
}

//NewClientHTTP initializes and returns a new client.
func NewClientHTTP(name string, rooms *RoomList, chl *os.File, m map[string]*ClientHTTP) *ClientHTTP {
	cl := new(ClientHTTP)
	cl.name = name
	cl.Messages = list.New()
	cl.Room = nil
	cl.Rooms = rooms
	d := 5*time.Minute
	cl.timeOut = time.AfterFunc(d,cl.Quit)
	cl.Token = NewToken()
	cl.ChatLog = chl
	cl.Clients = m
	cl.Blocked = list.New()
	return cl
}

//IsBlocked checks if other is blocked by the client.
func (cl *ClientHTTP) IsBlocked(other Client) (blocked bool) {
	blocked = false
	for i := cl.Blocked.Front(); i != nil; i = i.Next() {
		if i.Value == other.Name() {
			blocked = true
		}
	}
	return
}

//UnBlock removes clients with name in body of req from clients block list.
func (cl *ClientHTTP) UnBlock(w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	var name string
	err := dec.Decode(&name)
	if err != nil {
		log.Println("Error decoding in unblock: ", err)
	}
	found := false
	for i,x := cl.Blocked.Front(), cl.Blocked.Front(); i != nil; {
		x = i
		i = i.Next()
		if x.Value == name {
			cl.Blocked.Remove(x)
			found = true
		}
	}
	enc := json.NewEncoder(w)
	err = enc.Encode(found)
	if err != nil {
		log.Println("Error encoding in unblock: ", err)
	}
}

//Block adds clients with name matching the body of req to clients block list.
func (cl *ClientHTTP) Block(w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	var name string
	err := dec.Decode(&name)
	if err != nil {
		log.Println("Error decoding in Block: ", err)
	}
	cl.Blocked.PushBack(name)
}

//Name returns the clients name.
func (cl *ClientHTTP) Name() string {
	return cl.name
}

//Equals compares the client to other and returns true if they have the same name and token.
func (cl *ClientHTTP) Equals(other Client) bool {
	if c, ok := other.(*ClientHTTP); ok {
		return cl.Name() == c.Name() && cl.Token == c.Token
	}
	return false
}

//Recieve adds the message to the clients message list.
func (cl *ClientHTTP) Recieve(m Message) {
	if msg,ok := m.(*clientMessage);ok {
		if cl.IsBlocked(msg.Sender) {
			return
		}
	}
	cl.Messages.PushBack(m)
}

//Quit removes the client from their current room and removes their token entry from the client map.
func (cl *ClientHTTP) Quit () {
	cl.Leave()
	delete(cl.Clients, cl.Token) //remove client/Token from map
}

//Removes the client from their current room.
func (cl *ClientHTTP) Leave() {
	if cl.Room != nil {
		cl.Room.Tell(fmt.Sprintf("%v leaves the room.", cl.Name()))
		_ = cl.Room.Remove(cl)
		cl.Room = nil
	}
}

//Send sends the clients message(contained in their request body) to thier current room.
func (cl *ClientHTTP) Send(w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	var mtext string
	err := dec.Decode(&mtext)
	if err != nil {
		log.Println("Error decoding message in Send: ", err)
	}
	message := newClientMessage(mtext,cl)
	cl.log(fmt.Sprint(message))
	cl.Room.Send(message)
}

//ResetTimeOut resets the clients timeout timer.
func (cl *ClientHTTP) ResetTimeOut() {
	_ = cl.timeOut.Reset(5*time.Minute)
}

//Join adds the client to the room or creates the room if it doesn't exist.
func (cl *ClientHTTP) Join(w http.ResponseWriter, rq *http.Request) {
	path := strings.Split(rq.URL.Path,"/")
	rmName := path[2]
	cl.Leave()
	rm := cl.Rooms.FindRoom(rmName)
	if rm == nil {
		newRoom := NewRoom(rmName)
		cl.Room = newRoom
		cl.Room.Add(cl)
		cl.Rooms.Add(cl.Room)
	}else {
	cl.Room = rm
	rm.Add(cl)
	}
	cl.Clients[cl.Token] = cl
	enc := json.NewEncoder(w)
	err := enc.Encode(cl.Token)
	if err != nil {
		log.Println("Error encoding client token in join: ", err)
	}
	cl.Room.Tell(fmt.Sprintf("%v has joined the room.", cl.Name()))
}

//Who writes the a list of the people currently in the room to the response.
func (cl *ClientHTTP) Who(w http.ResponseWriter, rq *http.Request) {
	path := strings.Split(rq.URL.Path,"/")
	rm := cl.Room
	if len(path) == 4 {
		rm = cl.Rooms.FindRoom(path[2])
	}
	if rm == nil {//room does not exist
		w.WriteHeader(http.StatusNotFound)
		return
	}
	enc := json.NewEncoder(w)
	clients := rm.Who()
	err := enc.Encode(rm.Name())
	if err != nil {
		log.Println("Error encoding room name in who: ", err)
	}
	err = enc.Encode(clients)
	if err != nil {
		log.Println("Error encoding in who: ",err)
	}
}

//List sends a list of the current rooms as a response.
func (cl *ClientHTTP) List(w http.ResponseWriter, rq *http.Request) {
	enc := json.NewEncoder(w)
	list := cl.Rooms.Who()
	err := enc.Encode(list)
	if err != nil {
		log.Println("Error encoding in List: :", err)
	}
}

//roomHandler handles the HTTP client requests.
type roomHandler struct {
	clients map[string]*ClientHTTP
	rooms *RoomList
	chl *os.File
}

//newRoomHandler initializes and returns a new roomHandler.
func newRoomHandler(rooms *RoomList, chl *os.File) *roomHandler {
	r := new(roomHandler)
	r.rooms = rooms
	r.chl = chl
	r.clients = make(map[string]*ClientHTTP)
	return r
}

//CheckToken returns true if the Token present and found in clients map.
func (h *roomHandler) CheckToken(rq *http.Request) bool{
	token := rq.Header.Get("Authorization")
	if token != "" {
		_, matched := h.clients[token]
		return matched
	}
	return false
}

//GetClient returns the client assisiated with the "Autorization" token in the header of the request if they are found.  If the Client is not present in the map a new client is created and returned.
func (h *roomHandler) GetClient(rq *http.Request) *ClientHTTP{
	if !h.CheckToken(rq) {
		path := strings.Split(rq.URL.Path, "/")
		var name string
		if len(path) == 4 && path[3] == "join" {
			dec := json.NewDecoder(rq.Body)
			err := dec.Decode(&name)
			if err != nil {
				log.Println ("Error decoding in GetClient: ", err)
			}
		} else {
			name = "Anon"
		}
		cl := NewClientHTTP(name, h.rooms, h.chl, h.clients)
		return cl
	}
	cl := h.clients[rq.Header.Get("Authorization")]
	cl.ResetTimeOut()
	return cl
}

//ServeHTTP handles requests other than those sent to the REST handler.
func (h *roomHandler) ServeHTTP (w http.ResponseWriter, rq *http.Request) {
	path := strings.Split(rq.URL.Path,"/")
	cl := h.GetClient(rq)
	switch path[1] {
		case "rooms":
			if len(path) == 3 && path[2] == "list" {
				cl.List(w, rq)
				return
			}
			if len(path) == 3 && path[2] == "quit" {
				cl.Quit()
				return
			}
			if len(path) == 3 && path[2] == "who" {
				cl.Who(w,rq)
				return
			}
			if len(path) < 4 {//fix later for more appropriate response
				log.Println("Error invalid path: ", rq.URL.Path)
				return
			}
			if !h.CheckToken(rq) && (path[3] != "join" && path[3] != "who") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			switch path[3] {
				case "join":
					cl.Join(w,rq)
				case "quit":
					cl.Quit()
				case "who":
					cl.Who(w,rq)
				default:
					log.Println("Error invald command: ", path[3])
			}
		case "messages":
			if !h.CheckToken(rq) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			switch rq.Method {
				case "GET":
					cl.GetMessages(w,rq)
				case "POST":
					cl.Send(w,rq)
			}
		case "block":
			if !h.CheckToken(rq) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			cl.Block(w,rq)
		case "unblock":
			if !h.CheckToken(rq) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			cl.UnBlock(w,rq)
	}
}
