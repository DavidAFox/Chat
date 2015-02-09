package main

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
)


type Client struct {
	Name string
	Connection net.Conn
	Blocked *list.List
	Room *Room
	Rooms *RoomList
	ChatLog *os.File
}

//Equals compares two clients to see if they're the same
func (cl Client) Equals (other *Client) bool {
	if cl.Name == other.Name && cl.Connection == other.Connection {
		return true
	} else {
		return false
	}
}
func (cl Client) IsBlocked(other *Client) (blocked bool) {
	blocked = false
	for i := cl.Blocked.Front(); i != nil; i = i.Next() {
		if i.Value == other.Name {
			blocked = true
		}
	}
	return
}
func (cl *Client) UnBlock(name []string) {
	if len(name) == 0 {
		cl.Tell("Must enter user to unblock")
		return
	}
	clname:= strings.Join(name, " ")
	found := false
	for i := cl.Blocked.Front(); i != nil; i = i.Next() {
		if i.Value == clname {
			cl.Blocked.Remove(i)
			found = true
		}
	}
	if found {
		cl.Tell(fmt.Sprintf("No longer blocking %v.", clname))
	} else {
		cl.Tell(fmt.Sprintf("You are not blocking %v.", clname))
	}
}

func (cl *Client) Block (name []string) {
	if len(name) == 0 {
		cl.Tell("Must enter user to block")
		return
	}
	clname := strings.Join(name, " ")
	cl.Blocked.PushBack(clname)
	cl.Tell(fmt.Sprintf("Now Blocking %v.", clname))
}

//Leave removes cl from current room and closes any rooms left empty
func (cl *Client) Leave () {
	cl.Rooms.Lock()
	defer cl.Rooms.Unlock()
	if cl.Room != nil {
		cl.Room.Tell(fmt.Sprintf("%v leaves the room.",cl.Name))
		for entry := cl.Room.Clients.Front(); entry != nil; entry = entry.Next() { //Remove the client from room
			if cl.Equals(entry.Value.(*Client)) {
				cl.Room.Clients.Remove(entry)
			}
		}
		cl.Rooms.CloseEmpty()
		cl.Room = nil
	}
}

//Log writes the string to the chat log
func (cl Client) Log (s string) {
	_,err := io.WriteString(cl.ChatLog, strings.TrimSuffix(s,"\r"))
	if err != nil {
		log.Println(err)
	}
	return
}

//Send sends the message to the clients room
func (cl Client) Send (m Message) {
	if cl.Room == nil {
		cl.Tell("You're not in a room.  Type /join roomname to join a room or /help for other commands.")
		return
	}
	for i := cl.Room.Clients.Front();i != nil;i = i.Next() {
		if !i.Value.(*Client).IsBlocked(&cl) {
			_, err := io.WriteString(i.Value.(*Client).Connection,fmt.Sprint(m))
			if err != nil {
				log.Println(err)
				i.Value.(*Client).Leave()//remove the client that caused the error from the room
			}
		}
	}
	cl.Log(fmt.Sprint(m))
	return
}
//Who sends to the client a list of all the people in the same room as the client
func (cl *Client) Who(rms []string) {
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

func (cl *Client) Cls () {
	for i := 0; i<100;i++ {
		cl.Tell("")
	}
}

func (cl *Client) List() {
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


//Quit logs the client out, removes the client from all rooms, and closes the connection
func (cl *Client) Quit() {
	cl.Leave()
	err := cl.Connection.Close()
	if err != nil {
		log.Println(err)
	}
	return
}

//join adds a client to rm or creats a room if it doesn't exist
func (cl *Client) Join (rms []string) {
	if len(rms) == 0 {
		cl.Tell("Must enter a Room to join")
		return
	}
	name := strings.Join(rms, " ")
	cl.Leave()//leave old room first
	cl.Rooms.Lock()
	defer cl.Rooms.Unlock()
	rm := cl.Rooms.FindRoom(name)
	if rm == nil {
		newRoom := NewRoom(name)
		cl.Room = newRoom //set the room as the clients room
		cl.Room.Clients.PushBack(cl) //add the client to the room
		cl.Rooms.PushBack(cl.Room) //add the room to the room list
	} else {
	cl.Room = rm
	rm.Clients.PushBack(cl)
	}
	cl.Room.Tell(fmt.Sprintf("%v has joined the room.",cl.Name))
}

//Help tells the client a list of valid commands
func (cl Client) Help () {
	cl.Tell("/quit to quit")
	cl.Tell("/join roomname to join a room")
	cl.Tell("/leave to leave the current room")
	cl.Tell("/who to see a list of people in the current room")
	cl.Tell("/list to see a list of the current rooms")
	cl.Tell("/block name to block messages from a user")
	cl.Tell("/unblock name to unblock messages from a user")
	cl.Tell("/help to see a list of commands")
}

//Tell sends a message to the client from the server
func (cl Client) Tell(msg string) {
	_,err := io.WriteString(cl.Connection, msg + "\r\n")
	if err != nil {
		log.Println(err)
	}
}

type RoomList struct {
	*list.List
	*sync.Mutex
}

//FindRoom returns the first room with name
func (rml *RoomList) FindRoom (name string) *Room {
	for i := rml.Front(); i !=nil; i = i.Next() {
		if i.Value.(*Room).Name == name {
			return i.Value.(*Room)
		}
	}
	return nil
}

//CloseEmpty closes all empty rooms
func (rml *RoomList) CloseEmpty () {
	for entry := rml.Front(); entry != nil; entry = entry.Next() {//Close any empty rooms
		if entry.Value.(*Room).Clients.Front() == nil {
			rml.Remove(entry)
		}
	}
}


type Room struct {
	Name string
	Clients *list.List
}

//NewRoom creates a room with name and returns it
func NewRoom (name string) *Room {
	newRoom := new(Room)
	newRoom.Name = name
	newRoom.Clients = list.New()
	return newRoom
}

//Who returns a []string with all the names of the clients in the room sorted
func (rm *Room) Who() []string {
	clist := make([]string,0,0)
	for i:= rm.Clients.Front();i != nil;i = i.Next() {
		clist = append(clist, i.Value.(*Client).Name)
	}
	sort.Strings(clist)
	return clist
}


//Tell sends a message to the room from the server
func (rm Room) Tell(msg string) {
	for i := rm.Clients.Front();i != nil;i = i.Next() {
		_, err := io.WriteString(i.Value.(*Client).Connection,fmt.Sprint(msg,"\r\n"))
		if err != nil {
			log.Println(err)
		}
	}
	return
}


type Message struct {
	text string
	time time.Time
	Sender Client
}
func (m Message) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v\n\r",m.time.Format(layout),m.Sender.Name,m.text)
}

//newMessage creates a new message
func newMessage(t string, s Client) Message {
	return Message{t,time.Now(),s}
}

type config struct {
	ListeningIP string
	ListeningPort string
	LogFile string
}

//readString reads a string from the connection ending with a '\n'
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

//handleConnection handles overall connection
func handleConnection(conn net.Conn, rooms *RoomList,chl *os.File) {
	_, err := io.WriteString(conn, "What is your name? ")//set up the client
	if err != nil {
		log.Println("Error Writing",err)
	}
	name, err := readString(conn)
	if err != nil {
		log.Println("Error Reading",err)
	}
	cl := Client{name,conn,list.New(),nil,rooms,chl}
	for{
		input, err := readString(cl.Connection)
		if err != nil {
			log.Println("Error Reading",err)
		}
		if strings.HasPrefix(input,"/") {// handle commands
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
			default:
				cl.Tell("Invalid Command Type /help for list of commands")
			}
		}else {
			cl.Send(newMessage(input,cl))
		}
	}
}

//configure loads the config file
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

//server listens for connections and sends them to handleConnection()
func server (c *config) {
	rooms := &RoomList{list.New(),new(sync.Mutex)}
	chl,err := os.Create(c.LogFile)
	defer chl.Close()
	if err != nil {
		log.Panic(err)
	}
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


func main() {
	c := configure("Config")
	server(c)
}
