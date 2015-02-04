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
)


type Client struct {
	Name string
	Connection net.Conn
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
		for entry := cl.Rooms.Front(); entry != nil; entry = entry.Next() {//Close any empty rooms
			if entry.Value.(*Room).Clients.Front() == nil {
				cl.Rooms.Remove(entry)
			}
		}
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
		_, err := io.WriteString(i.Value.(*Client).Connection,fmt.Sprint(m))
		if err != nil {
			log.Println(err)
			i.Value.(*Client).Leave()
//			cl.Room.Clients.Remove(i)//remove the client that caused the error from the room
		}
	}
	cl.Log(fmt.Sprint(m))
	return
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
func (cl *Client) Join (rm string) {
	cl.Leave()//leave old room first
	cl.Rooms.Lock()
	defer cl.Rooms.Unlock()
	exists := false
	for entry := cl.Rooms.Front(); entry != nil; entry = entry.Next() {
		if entry.Value.(*Room).Name == rm {//if the room exists add the client to the room
			exists = true
			cl.Room = entry.Value.(*Room)
			entry.Value.(*Room).Clients.PushBack(cl)
		}
	}
	if exists == false {
		fmt.Print("Room not found making Room")
		newRoom := new(Room) //make a new room
		newRoom.Name = rm
		newRoom.Clients = list.New()
		cl.Room = newRoom //set the room as the clients room
		cl.Room.Clients.PushBack(cl)//add the client to the room
		cl.Rooms.PushBack(cl.Room)//add the room to the room list
	}
	cl.Room.Tell(fmt.Sprintf("%v has joined the room.",cl.Name))
}

//Help tells the client a list of valid commands
func (cl Client) Help () {
	cl.Tell("/quit to quit")
	cl.Tell("/join roomname to join a room")
	cl.Tell("/leave to leave current room")
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

type Room struct {
	Name string
	Clients *list.List
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

type config struct {
	ListeningIP string
	ListeningPort string
	LogFile string
}

//newMessage creates a new message
func newMessage(t string, s Client) Message {
	return Message{t,time.Now(),s}
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
	cl := Client{name,conn,nil,rooms,chl}
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
				if len(cmd) > 1 {
					rmName := strings.Join(cmd[1:], " ")
					cl.Join(rmName)
				} else {
					cl.Tell("Must enter channel to join")
				}
			case "/leave":
				cl.Leave()
			case "/help":
				cl.Help()
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
