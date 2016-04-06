package http

/*
Package http provides a connection implementation for use with the client package in the chat server.  It uses http requests/responses to connect the client with the server.  A new token is generated when a client logs in and should then be sent with each request.  The token is used in a client map to match the request with the client that sent it.
*/

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/DavidAFox/Chat/clientdata"
	"github.com/DavidAFox/Chat/connections"
	"github.com/DavidAFox/Chat/connections/websocket"
	"github.com/DavidAFox/Chat/message"
	"github.com/DavidAFox/Chat/room"
	gorilla "github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//Connection is used to pass information between the client and the client object.
type Connection struct {
	client   connections.Client
	messages *message.MessageList
	timeOut  *time.Timer
	token    string
	cMap     *ClientMap
}

//New creates a new Connection and associated client.
func (h *RoomHandler) New(m *ClientMap, name string, roomlist *room.RoomList, chatlog io.Writer, data clientdata.ClientData) *Connection {
	c := new(Connection)
	c.messages = message.NewMessageList()
	c.token = newToken()
	d := 5 * time.Minute
	c.timeOut = time.AfterFunc(d, c.Close)
	c.cMap = m
	c.client = h.clientFactory.New(name, c)
	return c
}

//RoomHandler handles the HTTP client requests.
type RoomHandler struct {
	clients       *ClientMap
	rooms         *room.RoomList
	chl           io.Writer
	datafactory   clientdata.Factory
	clientFactory connections.ClientFactory
	origin        string
}

type Options struct {
	RoomList      *room.RoomList
	ChatLog       io.Writer
	DataFactory   clientdata.Factory
	ClientFactory connections.ClientFactory
	Origin        string
}

//NewRoomHandler initializes and returns a new roomHandler.
func NewRoomHandler(options Options) *RoomHandler {
	r := new(RoomHandler)
	if options.RoomList == nil || options.DataFactory == nil || options.ClientFactory == nil {
		panic("Missing options in NewRoomHandler")
	}
	r.rooms = options.RoomList
	if options.ChatLog != nil {
		r.chl = options.ChatLog
	} else {

	}
	r.clients = NewClientMap()
	r.datafactory = options.DataFactory
	r.clientFactory = options.ClientFactory
	if options.Origin != "" {
		r.origin = options.Origin
	} else {
		r.origin = "*"
	}
	return r
}

//CheckToken returns true if the token present and found in clients map.
func (h *RoomHandler) CheckToken(rq *http.Request) bool {
	token := rq.Header.Get("Authorization")
	if token != "" {
		return h.clients.Check(token)
	}
	return false
}

//GetConnection returns the Connection associated with the "Autorization" token in the header of the request if they are found.  If the Client is not present in the map a new client is created and returned.
func (h *RoomHandler) GetConnection(rq *http.Request) *Connection {
	if !h.CheckToken(rq) {
		return nil
	}
	c := h.clients.Get(rq.Header.Get("Authorization"))
	c.ResetTimeOut()
	return c
}

//ServeHTTP handles requests other than those sent to the REST handler.
func (h *RoomHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	if rq.Body != nil {
		defer rq.Body.Close()
	}
	path := strings.Split(rq.URL.Path, "/")
	if rq.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", h.origin)
		w.Header().Add("Access-Control-Allow-Methods", "POST")
		w.Header().Add("Access-Control-Allow-Methods", "GET")
		w.Header().Add("Access-Control-Allow-Methods", "OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "1728000")
		w.Header().Set("Content-Type", "application/json")
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", h.origin)
	w.Header().Set("Access-Control-Expose-Headers", "Success")
	w.Header().Add("Access-Control-Expose-Headers", "Code")
	if strings.ToLower(rq.Header.Get("Upgrade")) == "websocket" {
		upgrader := &gorilla.Upgrader{HandshakeTimeout: (time.Minute * 2), ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == h.origin }}
		socket, err := upgrader.Upgrade(w, rq, nil)
		if err != nil {
			log.Println(err)
			return
		}
		go websocket.Start(socket, &websocket.Options{RoomList: h.rooms, ClientFactory: h.clientFactory, DataFactory: h.datafactory, ChatLog: h.chl})
		return
	}
	if len(path) < 2 {
		return
	}
	switch path[1] {
	case "login":
		h.Login(w, rq)
	case "register":
		h.Register(w, rq)
	default:
		if !h.CheckToken(rq) {
			w.Header().Set("WWW-Authenticate", "token")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		c := h.GetConnection(rq)
		if path[1] == "messages" && rq.Method == "GET" {
			c.GetMessages(w, rq)
			return
		}
		com := make([]string, 1, 1)
		com[0] = path[1]
		args := make([]string, 0, 0)
		dec := json.NewDecoder(rq.Body)
		err := dec.Decode(&args)
		if err != nil {
			log.Println(err)
		}
		com = append(com, args...)
		resp := c.client.Execute(com) //do the stuff
		w.Header().Set("success", strconv.FormatBool(resp.Success()))
		w.Header().Set("code", strconv.Itoa(resp.Code()))
		switch resp.Code() {
		case 70:
			w.WriteHeader(http.StatusNotFound)
			return
		case 60:
			w.Header().Set("Allow", resp.Data().(string))
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		case 50:
			w.WriteHeader(http.StatusInternalServerError)
			return
		default:
			enc := json.NewEncoder(w)
			if rq.Header.Get("Data") == "simple" || !resp.Success() {
				err := enc.Encode(resp.String())
				if err != nil {
					log.Println(err)
				}
			} else {
				info := new(Response)
				info.Data = resp.Data()
				info.String = resp.String()
				enc := json.NewEncoder(w)
				err := enc.Encode(info)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}

type Response struct {
	Data   interface{}
	String string
}

//Register is used to create new accounts through the http api.  It expects a login object in the body representing the account to be created.
func (h *RoomHandler) Register(w http.ResponseWriter, rq *http.Request) {
	l := make([]string, 0, 0)
	if rq.Method != "POST" {
		w.Header().Set("success", "false")
		w.Header().Set("code", "60")
		w.Header().Set("Allow", "POST")
		enc := json.NewEncoder(w)
		err := enc.Encode("Unsupported Method: Use POST to register.")
		if err != nil {
			log.Println("Error encoding in Register: ", err)
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	dec := json.NewDecoder(rq.Body)
	err := dec.Decode(&l)
	if err != nil {
		log.Println("Error decoding in Register: ", err)
	}
	if len(l) < 2 {
		log.Println("not enough args in Register")
	}
	if !clientdata.ValidateName(l[0]) {
		w.Header().Set("success", "false")
		w.Header().Set("code", "20")
		enc := json.NewEncoder(w)
		err2 := enc.Encode("Invalid name.  Name can only contain alpha numeric characters")
		if err2 != nil {
			log.Println("Error encoding in Register: ", err)
		}
		return
	}
	data := h.datafactory.Create(l[0])
	err = data.NewClient(l[1])
	switch {
	case err == clientdata.ErrClientExists:
		w.Header().Set("success", "false")
		w.Header().Set("code", "10")
		enc := json.NewEncoder(w)
		err2 := enc.Encode("A client with that name already exists.")
		if err2 != nil {
			log.Println("Error encoding in Register: ", err)
		}
	case err == clientdata.ErrAccountCreationDisabled:
		w.Header().Set("success", "false")
		w.Header().Set("code", "0")
		enc := json.NewEncoder(w)
		err2 := enc.Encode("Account creation has been disabled.")
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
func (h *RoomHandler) Login(w http.ResponseWriter, rq *http.Request) {
	var success bool
	l := make([]string, 0, 0)
	dec := json.NewDecoder(rq.Body)
	err := dec.Decode(&l)
	if err != nil {
		ServerError(w, err)
		return
	}
	if len(l) < 2 {
		log.Println("Not enough args in login")
		return
	}
	if !clientdata.ValidateName(l[0]) {
		w.Header().Set("success", "false")
		w.Header().Set("code", "20")
		enc := json.NewEncoder(w)
		err2 := enc.Encode("Invalid name.  Name can only contain alpha numeric characters")
		if err2 != nil {
			log.Println("Error encoding in Login: ", err)
		}
		return
	}
	data := h.datafactory.Create(l[0])
	success, err = data.Authenticate(l[1])
	if err != nil {
		ServerError(w, err)
		return
	}
	enc := json.NewEncoder(w)
	if success {
		if h.rooms.GetClient(l[0]) != nil {
			w.Header().Set("success", "false")
			w.Header().Set("code", "21")
			err = enc.Encode("That user is already logged in.")
			if err != nil {
				log.Println("Error encoding in login: ", err)
			}
			return
		}
		w.Header().Set("success", "true")
		c := h.New(h.clients, l[0], h.rooms, h.chl, data)
		c.cMap.Add(c)
		err = enc.Encode(c.token)
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

//ResetTimeOut resets the clients timeout timer.
func (cl *Connection) ResetTimeOut() {
	_ = cl.timeOut.Reset(5 * time.Minute)
}

//ServerError handles server errors writing a response to the client and logging the error.
func ServerError(w http.ResponseWriter, err error) {
	log.Println(err)
	w.Header().Set("success", "false")
	w.Header().Set("code", "50")
	w.WriteHeader(http.StatusInternalServerError)
}

//SendMessage is used by th client package to forward messages to the connection to be sent to the user.
func (cl *Connection) SendMessage(m message.Message) {
	cl.messages.Lock()
	cl.messages.PushBack(m)
	cl.messages.Unlock()
}

//Close removes the client from any room, deletes its token from the map and stops its timeout function.
func (cl *Connection) Close() {
	cl.client.LeaveRoom()
	cl.cMap.Delete(cl.token)
	_ = cl.timeOut.Stop()
}

//GetMessage gets all the messages for a client since the last time they were checked and then removes them from their message list.
func (cl *Connection) GetMessages(w http.ResponseWriter, rq *http.Request) {
	cl.messages.Lock()
	m := make([]message.Message, cl.messages.Len(), cl.messages.Len())
	for i, x := cl.messages.Front(), 0; i != nil; i, x = i.Next(), x+1 {
		m[x] = i.Value.(message.Message)
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

//Update is an alternative to GetMessages that an http client can use to update data with one request.  The body should contain a slice of strings representing the different data that the client wants.
func (cl *Connection) Update(w http.ResponseWriter, rq *http.Request) {
	requests := make([]string, 0, 0)
	dec := json.NewDecoder(rq.Body)
	err := dec.Decode(&requests)
	if err != nil {
		log.Println(err)
	}
	resp := make(map[string]interface{})
	failed := false
	for _, i := range requests {
		switch i {
		case "messages":
			cl.messages.Lock()
			m := make([]message.Message, cl.messages.Len(), cl.messages.Len())
			for i, x := cl.messages.Front(), 0; i != nil; i, x = i.Next(), x+1 {
				m[x] = i.Value.(message.Message)
			}
			for i, x := cl.messages.Front(), cl.messages.Front(); i != nil; {
				x = i
				i = i.Next()
				cl.messages.Remove(x)
			}
			cl.messages.Unlock()
			resp["messages"] = m
		case "friendlist":
			r := cl.client.Execute([]string{"friendlist"})
			if r.Success() {
				resp["friendlist"] = r.Data()
			} else {
				failed = true
				resp["friendlist"] = "failed"
			}
		case "who":
			r := cl.client.Execute([]string{"who"})
			if r.Success() {
				resp["who"] = r.Data()
			} else {
				failed = true
				resp["who"] = "failed"
			}
		default:
			resp[i] = "failed"
			failed = true
		}
	}
	enc := json.NewEncoder(w)
	if failed {
		w.Header().Set("success", "false")
	} else {
		w.Header().Set("success", "true")
	}
	err = enc.Encode(resp)
	if err != nil {
		log.Println("Error encoding in Update: ", err)
	}
}

//ClientMap is a concurrent safe map of clients
type ClientMap struct {
	clients map[string]*Connection
	in      chan interface{}
}

//NewClientMap makes a new ClientMap and starts its handle function.
func NewClientMap() *ClientMap {
	clm := new(ClientMap)
	clm.clients = make(map[string]*Connection)
	clm.in = make(chan interface{})
	go clm.handle()
	return clm
}

//mapAdd is an object to send to the clientmap to add a client to the map.
type mapAdd struct {
	client *Connection
	added  chan bool
}

func newMapAdd(cl *Connection) *mapAdd {
	cmd := new(mapAdd)
	cmd.client = cl
	cmd.added = make(chan bool)
	return cmd
}

//mapGet is an object to send to the client map to get a client from the map.
type mapGet struct {
	token    string
	response chan *Connection
}

func newMapGet(token string) *mapGet {
	cmd := new(mapGet)
	cmd.token = token
	cmd.response = make(chan *Connection)
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
func (clm *ClientMap) Add(c *Connection) bool {
	cmd := newMapAdd(c)
	clm.in <- cmd
	return <-cmd.added
}

//Check returns true if there is a client matching token in the map.
func (clm *ClientMap) Check(token string) bool {
	cmd := newMapGet(token)
	clm.in <- cmd
	c := <-cmd.response
	if c != nil {
		return true
	}
	return false
}

//Get returns client with matching token from the map.
func (clm *ClientMap) Get(token string) *Connection {
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
