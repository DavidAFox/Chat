package main

import (
	"container/list"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"sort"
	"strings"
)

//ClientTelnet is the telnet version of the client type.
type ClientTelnet struct {
	name       string
	connection net.Conn
	blocked    *list.List
	room       *Room
	rooms      *RoomList
	ChatLog    *os.File
}

//Name returns the name of the client.
func (cl *ClientTelnet) Name() string {
	return cl.name
}

//NewClient creates and returns a new client.
func NewClientTelnet(name string, conn net.Conn, rooms *RoomList, chl *os.File) *ClientTelnet {
	cl := new(ClientTelnet)
	cl.name = name
	cl.connection = conn
	cl.blocked = list.New()
	cl.room = nil
	cl.rooms = rooms
	cl.ChatLog = chl
	return cl
}

//Equals compares two clients to see if they're the same.
func (cl *ClientTelnet) Equals(other Client) bool {
	if c, ok := other.(*ClientTelnet); ok {
		return cl.Name() == c.Name() && cl.connection == c.connection
	}
	return false
}

//IsBlocked checks if other is blocked by the client.
func (cl *ClientTelnet) IsBlocked(other Client) (blocked bool) {
	blocked = false
	for i := cl.blocked.Front(); i != nil; i = i.Next() {
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
	clname := strings.Join(name, " ")
	found := false
	for i, x := cl.blocked.Front(), cl.blocked.Front(); i != nil; {
		x = i
		i = i.Next()
		if x.Value == clname {
			cl.blocked.Remove(x)
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
func (cl *ClientTelnet) Block(name []string) {
	if len(name) == 0 {
		cl.Tell("Must enter user to block")
		return
	}
	clname := strings.Join(name, " ")
	if clname == cl.Name() {
		cl.Tell("You can't block yourself.")
		return
	}
	cl.blocked.PushBack(clname)
	cl.Tell(fmt.Sprintf("Now Blocking %v.", clname))
}

//Leave removes cl from current room.
func (cl *ClientTelnet) Leave() {
	if cl.room != nil {
		_ = cl.room.Remove(cl)
		rm2 := cl.room
		cl.room = nil
		rm2.Tell(fmt.Sprintf("%v leaves the room.", cl.Name()))
		cl.Tell(fmt.Sprintf("%v leaves the room.", cl.Name()))
	}
}

//Log writes the string to the chat log.
func (cl ClientTelnet) Log(s string) {
	var err error
	_, err = io.WriteString(cl.ChatLog, s+"\n")
	if err != nil {
		log.Println(err)
	}
}

//Send sends the message to the clients room.
func (cl ClientTelnet) Send(m Message) {
	if cl.room == nil {
		cl.Tell("You're not in a room.  Type /join roomname to join a room or /help for other commands.")
		return
	}
	cl.room.Send(m)
	cl.Log(fmt.Sprint(m))
}

//Recieve takes messages and transmits them to the client
func (cl *ClientTelnet) Recieve(m Message) {
	if msg, ok := m.(*clientMessage); ok {
		if cl.IsBlocked(msg.Sender) {
			return
		}
	}
	_, err := io.WriteString(cl.connection, m.String()+"\r\n")
	if err != nil {
		log.Println(err)
		cl.Leave() //remove the client that caused the error from the room
	}
}

/*
//Refresh clears the clients screen and then sends them all the messages sent in the room they are in.
func (cl ClientTelnet) Refresh () error {
	if cl.room != nil {
		cl.Cls()
		for i := cl.room.Messages.Front(); i != nil; i = i.Next() {
			_, err := io.WriteString(cl.connection,fmt.Sprint(i.Value.(Message)))
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
		if cl.room == nil {
			cl.Tell("You're not in a room.  Type /join roomname to join a room or /help for other commands.")
			return
		}
		clist = cl.room.Who()
		cl.Tell(fmt.Sprintf("Room: %v", cl.room.Name()))
	} else {
		name := strings.Join(rms, " ")
		rm := cl.rooms.FindRoom(name)
		if rm == nil {
			cl.Tell("Room not Found")
			return
		}
		clist = rm.Who()
		cl.Tell(fmt.Sprintf("Room: %v", rm.Name()))
	}
	for _, i := range clist {
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
	rlist := cl.rooms.Who()
	sort.Strings(rlist)
	for _, i := range rlist {
		cl.Tell(i)
	}
}

//Quit logs the client out, removes the client from all rooms, and closes the connection.
func (cl *ClientTelnet) Quit() {
	cl.Leave()
	err := cl.connection.Close()
	if err != nil {
		log.Println(err)
	}
}

//Join adds a client to rm or creats a room if it doesn't exist.
func (cl *ClientTelnet) Join(rms []string) {
	if len(rms) == 0 {
		cl.Tell("Must enter a Room to join")
		return
	}
	name := strings.Join(rms, " ")
	cl.Leave() //leave old room first
	rm := cl.rooms.FindRoom(name)
	if rm == nil {
		newRoom := NewRoom(name)
		cl.room = newRoom     //set the room as the clients room
		cl.room.Add(cl)       //add the client to the room
		cl.rooms.Add(cl.room) //add the room to the room list
	} else {
		cl.room = rm
		rm.Add(cl)
	}
	cl.room.Tell(fmt.Sprintf("%v has joined the room.", cl.Name()))
}

//Help tells the client a list of valid commands.
func (cl ClientTelnet) Help() {
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
	if cl.rooms != nil {
		cl.rooms.CloseEmpty()
	}
}

//Tell sends a message to the client from the server.
func (cl ClientTelnet) Tell(s string) {
	msg := serverMessage{s}
	cl.Recieve(msg)
}

//readString reads a string from the connection ending with a '\n'and removes a '\r' if present.
func readString(conn net.Conn) (string, error) {
	r := make([]byte, 1)
	var ip string
	var err error
	_, err = conn.Read(r)
	for r[0] != '\n' {
		ip = ip + string(r[0])
		_, err = conn.Read(r)
	}
	if err != nil {
		log.Println(err)
	}
	re, err := regexp.Compile("[^\010]\010") //get rid of backspace and character in front of it
	if err != nil {
		log.Println("Error with regex in readString: ", err)
	}
	for re.MatchString(ip) { //keep getting rid of characters and backspaces as long as there are pairs left
		ip = re.ReplaceAllString(ip, "")
	}
	re2, err := regexp.Compile("^*\010") //get rid of any leading backspaces
	if err != nil {
		log.Println("Error with second regex in readString: ", err)
	}
	ip = re2.ReplaceAllString(ip, "")
	return strings.TrimSuffix(ip, "\r"), err
}

//handleConnection handles overall telnet connection.
func handleConnection(conn net.Conn, rooms *RoomList, chl *os.File) {
	_, err := io.WriteString(conn, "What is your name? ") //set up the client
	if err != nil {
		log.Println("Error Writing", err)
	}
	name, err := readString(conn)
	if err != nil {
		log.Println("Error Reading", err)
	}
	cl := NewClientTelnet(name, conn, rooms, chl)
	for {
		input, err := readString(cl.connection)
		if err != nil {
			log.Println("Error Reading", err)
		}
		if strings.HasPrefix(input, "/") { // handle commands
			cmd := strings.Fields(input)
			switch cmd[0] {
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
		} else {
			cl.Send(newClientMessage(input, cl))
		}
	}
}
