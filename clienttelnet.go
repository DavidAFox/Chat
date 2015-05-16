package main

import (
	"container/list"
	"fmt"
	"github.com/davidafox/chat/clientdata"
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
	data       clientdata.ClientData
}

//Name returns the name of the client.
func (cl *ClientTelnet) Name() string {
	return cl.name
}

//NewClient creates and returns a new client.
func NewClientTelnet(name string, conn net.Conn, rooms *RoomList, chl *os.File, cd clientdata.ClientData) *ClientTelnet {
	cl := new(ClientTelnet)
	cl.name = name
	cl.connection = conn
	cl.blocked = list.New()
	cl.room = nil
	cl.rooms = rooms
	cl.ChatLog = chl
	cl.data = cd
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
	var err error
	blocked, err = cl.data.IsBlocked(other.Name())
	if err != nil {
		log.Println("Error Telnet IsBlocked: ", err)
	}
	return
}

//UnBlock removes clients with name matching the args from clients block list.
func (cl *ClientTelnet) UnBlock(name []string) {
	if len(name) == 0 {
		cl.Tell("Must enter user to unblock")
		return
	}
	clname := name[0]
	if !clientdata.ValidateName(clname) {
		cl.Tell("Invalid name.  Name must be alphanumeric characters only.")
		return
	}
	err := cl.data.Unblock(clname)
	switch {
	case err == clientdata.ErrNotBlocking:
		cl.Tell(fmt.Sprintf("You are not blocking %v.", clname))
	case err != nil:
		log.Println("Telnet UnBlock: ", err)
	default:
		cl.Tell(fmt.Sprintf("No longer blocking %v.", clname))
	}
}

//Block adds clients with name matching the args to clients block list.
func (cl *ClientTelnet) Block(name []string) {
	if len(name) == 0 {
		cl.Tell("Must enter user to block")
		return
	}
	clname := name[0]
	if !clientdata.ValidateName(clname) {
		cl.Tell("Invalid name.  Name must be alphanumeric characters only.")
		return
	}
	if clname == cl.Name() {
		cl.Tell("You can't block yourself.")
		return
	}
	err := cl.data.Block(clname)
	switch {
	case err == clientdata.ErrBlocking:
		cl.Tell(fmt.Sprintf("You are already blocking %v.", clname))
	case err != nil:
		log.Println("Error Telnet Block: ", err)
	default:
		cl.Tell(fmt.Sprintf("Now Blocking %v.", clname))
	}
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
	name := rms[0]
	if !clientdata.ValidateName(name) {
		cl.Tell("Invalid room name.  Name must be alphanumeric characters only.")
		return
	}
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

//TelnetRegister is used to create new accounts using a telnet connection.
func TelnetRegister(conn net.Conn, cd clientdata.ClientData) {
	for {
		name := getInput(conn, "Enter Name.")
		if clientdata.ValidateName(name) {
			exists, err := cd.ClientExists(name)
			if err != nil {
				log.Println(err)
			}
			if !exists {
				pword1 := getInput(conn, "Enter Password.")
				pword2 := getInput(conn, "Please enter Password again.")
				for pword1 != pword2 {
					pword1 = getInput(conn, "Passwords don't match. Enter Password.")
					pword2 = getInput(conn, "Please enter Password again.")
				}
				cd.SetName(name)
				err := cd.NewClient(pword1)
				if err != nil {
					log.Println("Error registering client", err)
					_, err = io.WriteString(conn, "Error creating account.\n\r")
					if err != nil {
						log.Println("Error Writing in TelnetRegister", err)
					}
				} else {
					_, err = io.WriteString(conn, "Account Created.\n\r")
					if err != nil {
						log.Println("Error Writing in TelnetRegister", err)
					}
				}
				return
			}
			_, err = io.WriteString(conn, "A client with that name already exists.\n\r")
			if err != nil {
				log.Println("Error Writing", err)
			}
		} else {
			_, err := io.WriteString(conn, "Invalid Name.  Name must be alphanumeric characters only.")
			if err != nil {
				log.Println("Error Writing", err)
			}
		}
	}
}

//getInput sends the text string and then returns the response from the connection.
func getInput(conn net.Conn, text string) string {
	_, err := io.WriteString(conn, text+"\n\r")
	if err != nil {
		log.Println("Error Writing", err)
	}
	response, err := readString(conn)
	if err != nil {
		log.Println("Error Reading", err)
	}
	return response
}

//TelnetLogin is used to initiate clients.
func TelnetLogin(conn net.Conn, rooms *RoomList, chl *os.File, cd clientdata.ClientData) {
	logged := false
	var name string
	var err error
	for !logged {
		name = getInput(conn, "Enter Name or /new to create a new account.")
		if name == "/new" {
			TelnetRegister(conn, cd)
		} else if clientdata.ValidateName(name) {
			cd.SetName(name)
			pword := getInput(conn, "Enter Password.")
			logged, err = cd.Authenticate(pword)
			if err != nil {
				log.Println("Error Autheticating: ", err)
			}
			if logged == false {
				io.WriteString(conn, "User name and Password do not match.\n\r")
			}
		} else {
			_, err = io.WriteString(conn, "Invalid name.  Name must be alphanumeric characters only.")
			if err != nil {
				log.Println("Error Writing: ", err)
			}
		}
	}
	cl := NewClientTelnet(name, conn, rooms, chl, cd)
	cl.Tell("Welcome")
	go cl.inputhandler()
}

//inputhandler processes command from telnet connections.
func (cl *ClientTelnet) inputhandler() {
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
