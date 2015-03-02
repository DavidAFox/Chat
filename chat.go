package main
/* A Chat Server */
import (
	"net"
	"io"
	"fmt"
	"strings"
	"time"
	"encoding/json"
	"os"
	"container/list"
	"log"
	"sync"
	"sort"
	"net/http"
)

//Client interface for working with the Room type.
type Client interface {
	Equals (other Client) bool
	Name() string
	Recieve(m Message)
}

//ClientTelnet is the telnet version of the client type.
type ClientTelnet struct {
	name string
	Connection net.Conn
	Blocked *list.List
	Room *Room
	Rooms *RoomList
	ChatLog *os.File
}

//Name returns the name of the client.
func (cl *ClientTelnet) Name () string {
	return cl.name
}

//NewClient creates and returns a new client.
func NewClientTelnet(name string,conn net.Conn, rooms *RoomList, chl *os.File) *ClientTelnet {
	cl := new(ClientTelnet)
	cl.name = name
	cl.Connection = conn
	cl.Blocked = list.New()
	cl.Room = nil
	cl.Rooms = rooms
	cl.ChatLog = chl
	return cl
}

//Equals compares two clients to see if they're the same.
func (cl *ClientTelnet) Equals (other Client) bool {
	if c,ok := other.(*ClientTelnet);ok{
		return cl.Name() == c.Name() && cl.Connection == c.Connection
	}
	return false
}

//IsBlocked checks if other is blocked by the client.
func (cl *ClientTelnet) IsBlocked(other Client) (blocked bool) {
	blocked = false
	for i := cl.Blocked.Front(); i != nil; i = i.Next() {
		if i.Value == other.Name() {
			blocked = true
		}
	}
	return
}

//UnBlock removes clients with name matching the args from clients block list.
func (cl *ClientTelnet) UnBlock(name []string) {
	if len(name) == 0 {
		cl.Tell("Must enter user to unblock")
		return
	}
	clname:= strings.Join(name, " ")
	found := false
	for i, x := cl.Blocked.Front(), cl.Blocked.Front(); i != nil; {
		x = i
		i = i.Next()
		if x.Value == clname {
			cl.Blocked.Remove(x)
			found = true
		}
	}
	if found {
		cl.Tell(fmt.Sprintf("No longer blocking %v.", clname))
	} else {
		cl.Tell(fmt.Sprintf("You are not blocking %v.", clname))
	}
}

//Block adds clients with name matching the args to clients block list.
func (cl *ClientTelnet) Block (name []string) {
	if len(name) == 0 {
		cl.Tell("Must enter user to block")
		return
	}
	clname := strings.Join(name, " ")
	cl.Blocked.PushBack(clname)
	cl.Tell(fmt.Sprintf("Now Blocking %v.", clname))
}

//Leave removes cl from current room.
func (cl *ClientTelnet) Leave () {
	if cl.Room != nil {
		cl.Room.Tell(fmt.Sprintf("%v leaves the room.",cl.Name()))
		_ = cl.Room.Remove(cl)
		cl.Room = nil
	}
}

//Log writes the string to the chat log.
func (cl ClientTelnet) Log (s string) {
	var err error
	_, err = io.WriteString(cl.ChatLog, s + "\n")
	if err != nil {
		log.Println(err)
	}
}

//Send sends the message to the clients room.
func (cl ClientTelnet) Send (m Message) {
	if cl.Room == nil {
		cl.Tell("You're not in a room.  Type /join roomname to join a room or /help for other commands.")
		return
	}
	cl.Room.Send(m)
	cl.Log(fmt.Sprint(m))
}

//Recieve takes messages and transmits them to the client
func (cl *ClientTelnet) Recieve (m Message) {
	if msg,ok := m.(*clientMessage);ok {
		if cl.IsBlocked(msg.Sender) {
			return
		}
	}
	_,err := io.WriteString(cl.Connection, m.String()+"\r\n")
	if err != nil {
		log.Println(err)
		cl.Leave()//remove the client that caused the error from the room
	}
}

/*
//Refresh clears the clients screen and then sends them all the messages sent in the room they are in.
func (cl ClientTelnet) Refresh () error {
	if cl.Room != nil {
		cl.Cls()
		for i := cl.Room.Messages.Front(); i != nil; i = i.Next() {
			_, err := io.WriteString(cl.Connection,fmt.Sprint(i.Value.(Message)))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
*/

//Who sends to the client a list of all the people in the same room as the client.
func (cl *ClientTelnet) Who(rms []string) {
	var clist []string
	if len(rms) == 0 {
		if cl.Room == nil {
			cl.Tell("You're not in a room.  Type /join roomname to join a room or /help for other commands.")
			return
		}
		clist = cl.Room.Who()
		cl.Tell(fmt.Sprintf("Room: %v",cl.Room.Name))
	}else {
		name := strings.Join(rms, " ")
		rm := cl.Rooms.FindRoom(name)
		if rm == nil {
			cl.Tell("Room not Found")
			return
		}
		clist = rm.Who()
		cl.Tell(fmt.Sprintf("Room: %v", rm.Name))
	}
	for _,i := range clist {
		cl.Tell(i)
	}
}

/*
//Cls sends then client 100 new lines to clear their screen.
func (cl *ClientTelnet) Cls () {
	for i := 0; i<100;i++ {
		cl.Tell("")
	}
}
*/

//List sends to the client a list of the current open rooms.
func (cl *ClientTelnet) List() {
	cl.Tell("Rooms:")
	rlist := make([]string,0,0)
	for i := cl.Rooms.Front(); i != nil; i = i.Next() {
		rlist = append(rlist, i.Value.(*Room).Name)
	}
	sort.Strings(rlist)
	for _,i := range rlist {
		cl.Tell(i)
	}
}


//Quit logs the client out, removes the client from all rooms, and closes the connection.
func (cl *ClientTelnet) Quit() {
	cl.Leave()
	err := cl.Connection.Close()
	if err != nil {
		log.Println(err)
	}
}

//Join adds a client to rm or creats a room if it doesn't exist.
func (cl *ClientTelnet) Join (rms []string) {
	if len(rms) == 0 {
		cl.Tell("Must enter a Room to join")
		return
	}
	name := strings.Join(rms, " ")
	cl.Leave()//leave old room first
	rm := cl.Rooms.FindRoom(name)
	if rm == nil {
		newRoom := NewRoom(name)
		cl.Room = newRoom //set the room as the clients room
		cl.Room.Add(cl) //add the client to the room
		cl.Rooms.Lock()
		cl.Rooms.PushBack(cl.Room) //add the room to the room list
		cl.Rooms.Unlock()
	} else {
	cl.Room = rm
	rm.Add(cl)
	}
	cl.Room.Tell(fmt.Sprintf("%v has joined the room.",cl.Name()))
}

//Help tells the client a list of valid commands.
func (cl ClientTelnet) Help () {
	cl.Tell("/quit to quit")
	cl.Tell("/join roomname to join a room")
	cl.Tell("/leave to leave the current room")
	cl.Tell("/who to see a list of people in the current room")
	cl.Tell("/list to see a list of the current rooms")
	cl.Tell("/block name to block messages from a user")
	cl.Tell("/unblock name to unblock messages from a user")
	cl.Tell("/help to see a list of commands")
	cl.Tell("/close to close all empty rooms")
}

//Close closes all empty rooms.
func (cl *ClientTelnet) Close() {
	if cl.Rooms != nil {
		cl.Rooms.CloseEmpty()
	}
}

//Tell sends a message to the client from the server.
func (cl ClientTelnet) Tell(s string) {
	msg := serverMessage{s}
	cl.Recieve(msg)
}

//RoomList is a linked list of rooms with a mutex.
type RoomList struct {
	*list.List
	*sync.Mutex
}

//FindRoom returns the first room with name.
func (rml *RoomList) FindRoom (name string) *Room {
	for i := rml.Front(); i !=nil; i = i.Next() {
		if i.Value.(*Room).Name == name {
			return i.Value.(*Room)
		}
	}
	return nil
}

//Names returns a list of the names of the rooms in the list.
func (rml *RoomList) Names() []string {
	l := make([]string, rml.Len(), rml.Len())
	for i,x := rml.Front(), 0; i != nil; i, x = i.Next(), x+1 {
		l[x] = i.Value.(*Room).Name
	}
	return l
}

//CloseEmpty closes all empty rooms.
func (rml *RoomList) CloseEmpty () {
	rml.Lock()
	defer rml.Unlock()
	for entry,x := rml.Front(), rml.Front(); entry != nil; {//Close any empty rooms
		x = entry
		entry = entry.Next()
		if x.Value.(*Room).Clients.Front() == nil {
			rml.Remove(x)
		}
	}
}

//Room is a room name and a linked list of clients in the room.
type Room struct {
	Name string
	Clients *list.List
	Messages *list.List
	mux *sync.Mutex
}

//NewRoom creates a room with name.
func NewRoom (name string) *Room {
	newRoom := new(Room)
	newRoom.Name = name
	newRoom.Clients = list.New()
	newRoom.Messages = list.New()
	newRoom.mux = new(sync.Mutex)
	return newRoom
}

//Who returns a []string with all the names of the clients in the room sorted.
func (rm *Room) Who() []string {
	clist := make([]string,0,0)
	for i:= rm.Clients.Front();i != nil;i = i.Next() {
		clist = append(clist, i.Value.(Client).Name())
	}
	sort.Strings(clist)
	return clist
}

//Remove removes a client from the room.
func (rm *Room) Remove (cl Client) bool {
	rm.mux.Lock()
	found := false
	for i, x := rm.Clients.Front(), rm.Clients.Front(); i != nil; {
		x = i
		i = i.Next()
		if x.Value.(Client).Equals(cl) {
			rm.Clients.Remove(x)
			found = true
		}
	}
	rm.mux.Unlock()
	return found
}

//Add adds a client to a room.
func (rm *Room) Add (cl Client) {
	rm.mux.Lock()
	rm.Clients.PushBack(cl)
	rm.mux.Unlock()
}

//Tell sends a string to the room from the server.
func (rm Room) Tell(s string) {
	msg := serverMessage{s}
	rm.Send(msg)
}

//Send puts the message into each client in the room's recieve function.
func (rm *Room) Send (m Message) {
	for i := rm.Clients.Front(); i != nil; i = i.Next() {
		i.Value.(Client).Recieve(m)
	}
	rm.mux.Lock()
	rm.Messages.PushBack(m)
	rm.mux.Unlock()
}

//GetMessages gets the messages from the room message list and returns them as a []string.
func (rm Room) GetMessages() []string {
/*	if rm == nil {
		return make([]Message,0,0)
	}
*/	m := make([]string,rm.Messages.Len(), rm.Messages.Len())
	for i,x := rm.Messages.Front(), 0; i != nil; i,x = i.Next(),x+1 {
		m[x] = fmt.Sprint(i.Value)
	}
	return m
}

//Message is an interface for dealing with various types of messages.
type Message interface {
	String() string
}

//serverMessage is a message containing only a string sent from the server.
type serverMessage struct {
	text string
}

//String returns the string representation of the serverMessage.
func (m serverMessage) String() string {
	return m.text
}

//clientMessage includes the text of the message, the time it was sent and the client who sent it.
type clientMessage struct {
	text string
	time time.Time
	Sender Client
}

//String formats the clientMessage as time [Sender]: text.
func (m clientMessage) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v",m.time.Format(layout),m.Sender.Name(),m.text)
}

//newMessage creates a new client message
func newClientMessage(t string, s Client) *clientMessage {
	msg := new(clientMessage)
	msg.text = t
	msg.time = time.Now()
	msg.Sender = s
	return msg
}

//config stores the configuration data from the config file.
type config struct {
	ListeningIP string
	ListeningPort string
	HTTPListeningIP string
	HTTPListeningPort string
	LogFile string
}

//readString reads a string from the connection ending with a '\n'and removes a '\r' if present.
func readString(conn net.Conn) (string, error) {
	r := make([]byte,1)
	var ip string
	var err error
	_, err = conn.Read(r)
	for r[0] != '\n'{
		ip = ip + string(r[0])
		_, err = conn.Read(r)
	}
	return strings.TrimSuffix(ip,"\r"), err
}



//handleConnection handles overall connection.
func handleConnection(conn net.Conn, rooms *RoomList,chl *os.File) {
	_, err := io.WriteString(conn, "What is your name? ")//set up the client
	if err != nil {
		log.Println("Error Writing",err)
	}
	name, err := readString(conn)
	if err != nil {
		log.Println("Error Reading",err)
	}
	cl := NewClientTelnet(name,conn,rooms,chl)//Client{name,conn,list.New(),nil,rooms,chl}
	for{
		input, err := readString(cl.Connection)
		if err != nil {
			log.Println("Error Reading",err)
		}
/*		err = cl.Refresh()
		if err != nil {
			log.Println("Error writing refresh",err)
		}
*/		if strings.HasPrefix(input,"/") {// handle commands
			cmd := strings.Fields(input)
			switch cmd[0]{
			case "/quit":
				cl.Quit()
				return
			case "/join":
				cl.Join(cmd[1:])
			case "/leave":
				cl.Leave()
			case "/help":
				cl.Help()
			case "/who":
				cl.Who(cmd[1:])
			case "/list":
				cl.List()
			case "/block":
				cl.Block(cmd[1:])
			case "/unblock":
				cl.UnBlock(cmd[1:])
			case "/close":
				cl.Close()
			default:
				cl.Tell("Invalid Command Type /help for list of commands")
			}
		}else {
			cl.Send(newClientMessage(input,cl))
		}
	}
}

//configure loads the config file.
func configure (filename string) (c *config) {
	file, err := os.Open(filename)
	if err != nil {
		log.Panic("Error opening config file",err)
	}
	dec := json.NewDecoder(file)
	c = new(config)
	err = dec.Decode(c)
	if err !=nil {
		log.Panic("Error decoding config file",err)
	}
	return
}

//server listens for connections and sends them to handleConnection().
func server (rooms *RoomList, chl *os.File, c *config) {

	// go documentation code
	ln, err := net.Listen("tcp", net.JoinHostPort(c.ListeningIP,c.ListeningPort))
	if err != nil{
		log.Panic(err)
	}
	for{
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		go handleConnection(conn,rooms,chl)
	}
	// end go documentation code
}

//serverHTTP sets up the http handlers and then runs ListenAndServe
func serverHTTP (rooms *RoomList, chl *os.File, c *config) {
	room := newRoomHandler(rooms,chl)
	http.Handle("/", room)
	rest := newRestHandler(rooms,chl)
	http.Handle("/rest/", rest)
	err := http.ListenAndServe(net.JoinHostPort(c.HTTPListeningIP,c.HTTPListeningPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

//restHandler is the http.Handler for handling the REST API
type restHandler struct {
	rooms *RoomList
	chl *os.File
}

//newRestHandler initializes a new restHandler.
func newRestHandler(rooms *RoomList, chl *os.File) *restHandler{
	m := new(restHandler)
	m.rooms = rooms
	m.chl = chl
	return m
}

//ServeHTTP handles the restServer's http requests.
func (m *restHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	roomName := rq.URL.Path[len("/rest/"):]
	room := m.rooms.FindRoom(roomName)
	if room == nil {
		log.Println("ServeHTTP: room not found ", roomName)
		return
	}
	if rq.Method == "GET" {
		m.getMessages(room, w)
	}
	if rq.Method == "POST" {
		m.sendMessages(room,w,rq)
	}
}

//sendMessages handles REST requests for messages and writes them to the response.
func (m *restHandler) sendMessages (room *Room, w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	message := new(restMessage)
	err := dec.Decode(message)
	if err != nil {
		log.Println("Error decoding messages in sendMessages",err)
	}
	message.Time = time.Now()
	room.Send(message)
	m.log(message.String())
}

//log logs REST messages sent to the chatlog.
func (m *restHandler) log (s string) {
	var err error
	_, err = io.WriteString(m.chl, s + "\n")
	if err != nil {
		log.Println(err)
	}
}

//restMessage is a message sent from the REST API.
type restMessage struct {
	Name string
	Text string
	Time time.Time
}

//String returns a rest message string formated as Time [Name]: Text.
func (m *restMessage) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v", m.Time.Format(layout),m.Name, m.Text)
}


//GetMessage handles REST request for messages and writes them to the response.
func (m *restHandler) getMessages(room *Room, w http.ResponseWriter) {
	enc := json.NewEncoder(w)
	messages := room.GetMessages()
	err := enc.Encode(messages)
	if err != nil {
		log.Println("Error encoding messages in restHandler", err)
	}
}

func main() {
	c := configure("Config")
	rooms := &RoomList{list.New(),new(sync.Mutex)}
	chl, err := os.OpenFile(c.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	defer chl.Close()
	if err != nil {
		log.Panic(err)
	}
	go server(rooms, chl,c)
	serverHTTP(rooms,chl,c)
}
