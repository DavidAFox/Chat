package main

import (
	"container/list"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/davidafox/chat/clientdata"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

/*
Code 10 A Client with that name already exists
Code 20 Invalid Name
Code 21 User name and password don't match
Code 30 Already Blocking
Code 31 Not Blocking
Code 32 Can't block self
Code 40 Not in a room
Code 50 Server Error
Code 60 Unsupported Method
*/

//ClientMap is a concurrent safe map of clients
type ClientMap struct {
	clients map[string]*ClientHTTP
	in      chan interface{}
}

//NewClientMap makes a new ClientMap and starts its handle function.
func NewClientMap() *ClientMap {
	clm := new(ClientMap)
	clm.clients = make(map[string]*ClientHTTP)
	clm.in = make(chan interface{})
	go clm.handle()
	return clm
}

//mapAdd is an object to send to the clientmap to add a client to the map.
type mapAdd struct {
	client *ClientHTTP
	added  chan bool
}

func newMapAdd(cl *ClientHTTP) *mapAdd {
	cmd := new(mapAdd)
	cmd.client = cl
	cmd.added = make(chan bool)
	return cmd
}

//mapGet is an object to send to the client map to get a client from the map.
type mapGet struct {
	token    string
	response chan *ClientHTTP
}

func newMapGet(token string) *mapGet {
	cmd := new(mapGet)
	cmd.token = token
	cmd.response = make(chan *ClientHTTP)
	return cmd
}

//mapDelete is an object to send to the client map to delete a client from the map.
type mapDelete struct {
	token   string
	deleted chan bool
}

func newMapDelete(token string) *mapDelete {
	cmd := new(mapDelete)
	cmd.token = token
	cmd.deleted = make(chan bool)
	return cmd
}

//handle takes objects off the ClientMap's in channel, adjusts the map, and then passes back the results on the channel in the object from the in channel.
func (clm *ClientMap) handle() {
	for cmd := range clm.in {
		switch x := cmd.(type) {
		case *mapAdd:
			if _, ok := clm.clients[x.client.token]; ok {
				x.added <- false
			} else {
				clm.clients[x.client.token] = x.client
				x.added <- true
			}
		case *mapGet:
			if cl, ok := clm.clients[x.token]; ok {
				x.response <- cl
			} else {
				x.response <- nil
			}
		case *mapDelete:
			if _, ok := clm.clients[x.token]; !ok {
				x.deleted <- false
			} else {
				delete(clm.clients, x.token)
				x.deleted <- true
			}
		default:
			log.Println("Error invalid type in clientmap.handle().")
		}
	}
}

//Add adds the client to the map.
func (clm *ClientMap) Add(cl *ClientHTTP) bool {
	cmd := newMapAdd(cl)
	clm.in <- cmd
	return <-cmd.added
}

//Check returns true if there is a client matching token in the map.
func (clm *ClientMap) Check(token string) bool {
	cmd := newMapGet(token)
	clm.in <- cmd
	cl := <-cmd.response
	if cl != nil {
		return true
	}
	return false
}

//Get returns client with matching token from the map.
func (clm *ClientMap) Get(token string) *ClientHTTP {
	cmd := newMapGet(token)
	clm.in <- cmd
	return <-cmd.response
}

//Delete deletes client with matching token from the map if present.
func (clm *ClientMap) Delete(token string) bool {
	cmd := newMapDelete(token)
	clm.in <- cmd
	return <-cmd.deleted
}

//newToken creats a new random token string encoded as hex.
func newToken() string {
	b := make([]byte, 256, 256)
	_, err := rand.Read(b)
	if err != nil {
		log.Println("Error creating token: ", err)
	}
	return hex.EncodeToString(b)
}

//ClientHTTP is a client for the HTTP API
type ClientHTTP struct {
	name     string
	messages *messageList
	room     *Room
	timeOut  *time.Timer
	token    string
	rooms    *RoomList
	chatlog  *os.File
	clients  *ClientMap
	blocked  *list.List
	data     clientdata.ClientData
}

//log writes the string to the clients chatlog.
func (cl *ClientHTTP) log(s string) {
	var err error
	_, err = io.WriteString(cl.chatlog, s+"\n")
	if err != nil {
		log.Println(err)
	}
}

//GetMessage gets all the messages for a client since the last time they were checked and then removes them from their message list.
func (cl *ClientHTTP) GetMessages(w http.ResponseWriter, rq *http.Request) {
	cl.messages.Lock()
	m := make([]string, cl.messages.Len(), cl.messages.Len())
	for i, x := cl.messages.Front(), 0; i != nil; i, x = i.Next(), x+1 {
		m[x] = fmt.Sprint(i.Value)
	}
	for i, x := cl.messages.Front(), cl.messages.Front(); i != nil; {
		x = i
		i = i.Next()
		cl.messages.Remove(x)
	}
	cl.messages.Unlock()
	w.Header().Set("success", "true")
	enc := json.NewEncoder(w)
	err := enc.Encode(m)
	if err != nil {
		log.Println("Error encoding messages: ", err)
	}
}

//NewClientHTTP initializes and returns a new client.
func NewClientHTTP(name string, rooms *RoomList, chl *os.File, m *ClientMap, data clientdata.ClientData) *ClientHTTP {
	cl := new(ClientHTTP)
	cl.name = name
	cl.messages = newMessageList()
	cl.room = nil
	cl.rooms = rooms
	d := 5 * time.Minute
	cl.timeOut = time.AfterFunc(d, cl.Quit)
	cl.token = newToken()
	cl.chatlog = chl
	cl.clients = m
	cl.blocked = list.New()
	cl.data = data
	return cl
}

//IsBlocked checks if other is blocked by the client.
func (cl *ClientHTTP) IsBlocked(other Client) (blocked bool) {
	blocked, err := cl.data.IsBlocked(other.Name())
	if err != nil {
		log.Println(err)
	}
	return
}

//ServerError handles server errors writing a response to the client and logging the error.
func ServerError(w http.ResponseWriter, err error) {
	log.Println(err)
	w.Header().Set("success", "false")
	w.Header().Set("code", "50")
	w.WriteHeader(http.StatusInternalServerError)
}

//UnBlock removes clients with name in body of req from clients block list.
func (cl *ClientHTTP) UnBlock(w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	var name string
	err := dec.Decode(&name)
	if err != nil {
		ServerError(w, err)
	}
	enc := json.NewEncoder(w)
	if !clientdata.ValidateName(name) {
		w.Header().Set("success", "false")
		w.Header().Set("code", "20")
		err2 := enc.Encode("Invalid name.  Name must be alphanumeric characters only.")
		if err2 != nil {
			log.Println("Error encoding in unblock: ", err)
		}
		return
	}
	err = cl.data.Unblock(name)
	switch {
	case err == clientdata.ErrNotBlocking:
		w.Header().Set("success", "false")
		w.Header().Set("code", "31")
		err2 := enc.Encode(fmt.Sprintf("You are not blocking %v", name))
		if err2 != nil {
			log.Println("Error encoding in unblock: ", err)
		}
	case err != nil:
		ServerError(w, err)
	default:
		w.Header().Set("success", "true")
	}
}

//Block adds clients with name matching the body of req to clients block list.
func (cl *ClientHTTP) Block(w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	var name string
	err := dec.Decode(&name)
	if err != nil {
		ServerError(w, err)
	}
	enc := json.NewEncoder(w)
	if !clientdata.ValidateName(name) {
		w.Header().Set("success", "false")
		w.Header().Set("code", "20")
		err2 := enc.Encode("Invalid name.  Name must be alphanumeric characters only.")
		if err2 != nil {
			log.Println("Error encoding in block: ", err)
		}
		return
	}
	if name == cl.Name() {
		w.Header().Set("success", "false")
		w.Header().Set("code", "32")
		err2 := enc.Encode("You can't block yourself.")
		if err2 != nil {
			log.Println("Error encoding in block: ", err)
		}
		return
	}
	err = cl.data.Block(name)
	switch {
	case err == clientdata.ErrBlocking:
		w.Header().Set("success", "false")
		w.Header().Set("code", "30")
		err2 := enc.Encode(fmt.Sprintf("You are already blocking %v.", name))
		if err2 != nil {
			log.Println("Error encoding in block: ", err)
		}
	case err != nil:
		ServerError(w, err)
	default:
		w.Header().Set("success", "true")
	}
}

//Name returns the clients name.
func (cl *ClientHTTP) Name() string {
	return cl.name
}

//Equals compares the client to other and returns true if they have the same name and token.
func (cl *ClientHTTP) Equals(other Client) bool {
	if c, ok := other.(*ClientHTTP); ok {
		return cl.Name() == c.Name() && cl.token == c.token
	}
	return false
}

//Recieve adds the message to the clients message list.
func (cl *ClientHTTP) Recieve(m Message) {
	if msg, ok := m.(*clientMessage); ok {
		if cl.IsBlocked(msg.Sender) {
			return
		}
	}
	cl.messages.Lock()
	cl.messages.PushBack(m)
	cl.messages.Unlock()
}

//Quit removes the client from their current room and removes their token entry from the client map.
func (cl *ClientHTTP) Quit() {
	cl.Leave()
	cl.clients.Delete(cl.token)
	_ = cl.timeOut.Stop()
}

//QuitAction calls Quit() and then writes the response.
func (cl *ClientHTTP) QuitAction(w http.ResponseWriter, rq *http.Request) {
	cl.Quit()
	w.Header().Set("success", "true")
	return
}

//Removes the client from their current room.
func (cl *ClientHTTP) Leave() {
	if cl.room != nil {
		rm2 := cl.room
		_ = cl.room.Remove(cl)
		cl.room = nil
		rm2.Tell(fmt.Sprintf("%v leaves the room.", cl.Name()))
		cl.Recieve(serverMessage{fmt.Sprintf("%v leaves the room.", cl.Name())})
	}
}

//LeaveAction calls Leave() and then writes the response.
func (cl *ClientHTTP) LeaveAction(w http.ResponseWriter, rq *http.Request) {
	cl.Leave()
	w.Header().Set("success", "true")
	return
}

//Send sends the clients message(contained in their request body) to thier current room.
func (cl *ClientHTTP) Send(w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	var mtext string
	err := dec.Decode(&mtext)
	if err != nil {
		log.Println("Error decoding message in Send: ", err)
	}
	message := newClientMessage(mtext, cl)
	if cl.room != nil {
		w.Header().Set("success", "true")
		cl.log(fmt.Sprint(message))
		cl.room.Send(message)
	} else {
		cl.Recieve(serverMessage{"You're not in a room.  Type /join roomname to join a room or /help for other commands."})
		w.Header().Set("success", "false")
		w.Header().Set("code", "40")
		enc := json.NewEncoder(w)
		err = enc.Encode("You're not in a room.")
		if err != nil {
			log.Println("Error encoding message in Send: ", err)
		}
	}
}

//ResetTimeOut resets the clients timeout timer.
func (cl *ClientHTTP) ResetTimeOut() {
	_ = cl.timeOut.Reset(5 * time.Minute)
}

//Join adds the client to the room or creates the room if it doesn't exist.
func (cl *ClientHTTP) Join(w http.ResponseWriter, rq *http.Request) {
	path := strings.Split(rq.URL.Path, "/")
	rmName := path[2]
	cl.Leave()
	if !clientdata.ValidateName(rmName) {
		w.Header().Set("success", "false")
		w.Header().Set("code", "20")
		enc := json.NewEncoder(w)
		err2 := enc.Encode("Invalid room name.  Name can only contain alpha numeric characters")
		if err2 != nil {
			log.Println("Error encoding in Join: ", err2)
		}
		return
	}
	rm := cl.rooms.FindRoom(rmName)
	if rm == nil {
		newRoom := NewRoom(rmName)
		cl.room = newRoom
		cl.room.Add(cl)
		cl.rooms.Add(cl.room)
	} else {
		cl.room = rm
		rm.Add(cl)
	}
	cl.room.Tell(fmt.Sprintf("%v has joined the room.", cl.Name()))
	w.Header().Set("success", "true")
}

//Who writes the a list of the people currently in the room to the response.
func (cl *ClientHTTP) Who(w http.ResponseWriter, rq *http.Request) {
	path := strings.Split(rq.URL.Path, "/")
	rm := cl.room
	if len(path) == 4 {
		rm = cl.rooms.FindRoom(path[2])
	}
	if rm == nil { //room does not exist
		w.WriteHeader(http.StatusNotFound)
		return
	}
	enc := json.NewEncoder(w)
	clients := rm.Who()
	w.Header().Set("success", "true")
	err := enc.Encode(rm.Name())
	if err != nil {
		ServerError(w, err)
		return
	}
	err = enc.Encode(clients)
	if err != nil {
		ServerError(w, err)
		return
	}
}

//List sends a list of the current rooms as a response.
func (cl *ClientHTTP) List(w http.ResponseWriter, rq *http.Request) {
	enc := json.NewEncoder(w)
	list := cl.rooms.Who()
	w.Header().Set("success", "true")
	err := enc.Encode(list)
	if err != nil {
		ServerError(w, err)
		return
	}
}

//login is an object used for decoding login information.
type login struct {
	Name     string
	Password string
}

//Register is used to create new accounts through the http api.  It expects a login object in the body representing the account to be created.
func (h *roomHandler) Register(w http.ResponseWriter, rq *http.Request) {
	l := new(login)
	if rq.Method != "POST" {
		w.Header().Set("success", "false")
		w.Header().Set("Code", "60")
		enc := json.NewEncoder(w)
		err := enc.Encode("Unsupported Method: Use POST to register.")
		if err != nil {
			log.Println("Error encoding in Register: ", err)
		}
		return
	}
	dec := json.NewDecoder(rq.Body)
	err := dec.Decode(&l)
	if err != nil {
		log.Println("Error decoding in Register: ", err)
	}
	if !clientdata.ValidateName(l.Name) {
		w.Header().Set("success", "false")
		w.Header().Set("code", "20")
		enc := json.NewEncoder(w)
		err2 := enc.Encode("Invalid name.  Name can only contain alpha numeric characters")
		if err2 != nil {
			log.Println("Error encoding in Register: ", err)
		}
		return
	}
	data := h.datafactory.Create(l.Name)
	err = data.NewClient(l.Password)
	switch {
	case err == clientdata.ErrClientExists:
		w.Header().Set("success", "false")
		w.Header().Set("code", "10")
		enc := json.NewEncoder(w)
		err2 := enc.Encode("A client with that name already exists.")
		if err2 != nil {
			log.Println("Error encoding in Register: ", err)
		}
	case err != nil:
		ServerError(w, err)
	default:
		w.Header().Set("success", "true")
	}
}

//Login takes a login(name, password) from the rq body and tries to log the client in.  It will return a response with header "success" = "true" if the login is successful.
func (h *roomHandler) Login(w http.ResponseWriter, rq *http.Request) {
	var success bool
	l := new(login)
	dec := json.NewDecoder(rq.Body)
	err := dec.Decode(&l)
	if err != nil {
		ServerError(w, err)
		return
	}
	if !clientdata.ValidateName(l.Name) {
		w.Header().Set("success", "false")
		w.Header().Set("code", "20")
		enc := json.NewEncoder(w)
		err2 := enc.Encode("Invalid name.  Name can only contain alpha numeric characters")
		if err2 != nil {
			log.Println("Error encoding in Login: ", err)
		}
		return
	}
	data := h.datafactory.Create(l.Name)
	success, err = data.Authenticate(l.Password)
	if err != nil {
		ServerError(w, err)
		return
	}
	enc := json.NewEncoder(w)
	if success {
		w.Header().Set("success", "true")
		cl := NewClientHTTP(l.Name, h.rooms, h.chl, h.clients, data)
		cl.clients.Add(cl)
		err = enc.Encode(cl.token)
		if err != nil {
			log.Println("Error encoding client token in login: ", err)
		}
	} else {
		w.Header().Set("success", "false")
		w.Header().Set("code", "21")
		err = enc.Encode("User name and password don't match.")
		if err != nil {
			log.Println("Error encoding in login: ", err)
		}
	}

}

//roomHandler handles the HTTP client requests.
type roomHandler struct {
	clients     *ClientMap
	rooms       *RoomList
	chl         *os.File
	datafactory clientdata.Factory
}

//newRoomHandler initializes and returns a new roomHandler.
func newRoomHandler(rooms *RoomList, chl *os.File, df clientdata.Factory) *roomHandler {
	r := new(roomHandler)
	r.rooms = rooms
	r.chl = chl
	r.clients = NewClientMap()
	r.datafactory = df
	return r
}

//CheckToken returns true if the token present and found in clients map.
func (h *roomHandler) CheckToken(rq *http.Request) bool {
	token := rq.Header.Get("Authorization")
	if token != "" {
		return h.clients.Check(token)
	}
	return false
}

//GetClient returns the client associated with the "Autorization" token in the header of the request if they are found.  If the Client is not present in the map a new client is created and returned.
func (h *roomHandler) GetClient(rq *http.Request) *ClientHTTP {
	if !h.CheckToken(rq) {
		name := "Anon"
		cl := NewClientHTTP(name, h.rooms, h.chl, h.clients, h.datafactory.Create(name))
		return cl
	}
	cl := h.clients.Get(rq.Header.Get("Authorization"))
	cl.ResetTimeOut()
	return cl
}

//ServeHTTP handles requests other than those sent to the REST handler.
func (h *roomHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	path := strings.Split(rq.URL.Path, "/")
	cl := h.GetClient(rq)
	switch path[1] {
	case "rooms":
		if !h.CheckToken(rq) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if len(path) == 3 && path[2] == "list" {
			cl.List(w, rq)
			return
		}
		if len(path) == 3 && path[2] == "quit" {
			cl.QuitAction(w, rq)
			return
		}
		if len(path) == 3 && path[2] == "who" {
			cl.Who(w, rq)
			return
		}
		if len(path) == 3 && path[2] == "leave" {
			cl.LeaveAction(w, rq)
			return
		}
		if len(path) < 4 {
			log.Println("Error invalid path: ", rq.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch path[3] {
		case "join":
			cl.Join(w, rq)
		case "quit":
			cl.QuitAction(w, rq)
		case "who":
			cl.Who(w, rq)
		case "leave":
			cl.LeaveAction(w, rq)
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
			cl.GetMessages(w, rq)
		case "POST":
			cl.Send(w, rq)
		}
	case "block":
		if !h.CheckToken(rq) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		cl.Block(w, rq)
	case "unblock":
		if !h.CheckToken(rq) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		cl.UnBlock(w, rq)
	case "login":
		h.Login(w, rq)
	case "register":
		h.Register(w, rq)
	}
}
