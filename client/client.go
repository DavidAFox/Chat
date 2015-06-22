package client

import (
	"fmt"
	"github.com/davidafox/chat/clientdata"
	"github.com/davidafox/chat/message"
	"github.com/davidafox/chat/room"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

/*
Codes - will be found in header under "code" if the action fails and indicates the reason for the failure
10 A Client with that name already exists
20 Invalid Name
21 User name and password don't match
22 No argument provided
30 Already blocking that user
31 Not blocking that user
32 Can't block self
35 Already friend that user
36 Not friending that user
37 Can't friend self
40 Not in a Room
41 Room does not exist
42 Client not found
43 Client is blocking you
50 Server Error
60 Unsupported Method
70 Invalid Command
*/

//Connection represents the connection to the user.
type Connection interface {
	SendMessage(m message.Message)
	Close()
}

//Response is used to reply to commands from the clients connection.
type Response struct {
	Success        bool
	Code           int
	StringResponse string
	Data           interface{}
}

//NewResponse returns a new response.
func NewResponse(success bool, code int, sresp string, data interface{}) *Response {
	resp := new(Response)
	resp.Success = success
	resp.Code = code
	resp.StringResponse = sresp
	resp.Data = data
	return resp
}

//Client is used to represent the client in rooms and do server actions.
type Client struct {
	name       string
	room       *room.Room
	rooms      *room.RoomList
	chatlog    *os.File
	data       clientdata.ClientData
	connection Connection
}

//New returns a new client.
func New(name string, roomlist *room.RoomList, chatlog *os.File, data clientdata.ClientData, connection Connection) *Client {
	cl := new(Client)
	cl.name = name
	cl.rooms = roomlist
	cl.chatlog = chatlog
	cl.data = data
	cl.connection = connection
	err := cl.data.UpdateOnline(time.Now())
	if err != nil {
		log.Println(err)
	}
	return cl
}

//Recieve will pass messages along to the client.
func (cl *Client) Recieve(m message.Message) {
	if msg, ok := m.(message.ClientMessage); ok {
		if cl.IsBlocked(msg.Name()) {
			return
		}
	}
	cl.connection.SendMessage(m)
}

//Name returns the clients name.
func (cl *Client) Name() string {
	return cl.name
}

//Equals returns true if the client name and connection match.
func (cl *Client) Equals(other room.Client) bool {
	if cl.Name() == other.Name() {
		return true
	}
	return false
}

//Execute parses and then runs commands from the client connection.
func (cl *Client) Execute(command []string) *Response {
	if len(command) < 2 {
		command = append(command, "")
	}
	switch command[0] {
	case "messages":
		log.Println("messages")
		return cl.Send(command[1])
	case "join":
		return cl.Join(command[1])
	case "block":
		return cl.Block(command[1])
	case "unblock":
		return cl.Unblock(command[1])
	case "leave":
		return cl.Leave()
	case "quit":
		return cl.Quit()
	case "list":
		return cl.List()
	case "who":
		return cl.Who(command[1])
	case "send":
		return cl.Send(command[1])
	case "blocklist":
		return cl.BlockList()
	case "friend":
		return cl.Friend(command[1])
	case "unfriend":
		return cl.Unfriend(command[1])
	case "friendlist":
		return cl.FriendList()
	case "tell":
		if len(command) < 3 {
			command = append(command, "")
		}
		return cl.Tell(command[1], command[2])				
	default:
		return NewResponse(false, 70, "Invalid Command", nil)
	}
}

//IsBlocked returns true if other's name is on clients blocklist.
func (cl *Client) IsBlocked(other string) bool {
	blocked, err := cl.data.IsBlocked(other)
	if err != nil {
		log.Println("Error IsBlocked: ", err)
	}
	return blocked
}

//BlockList provides a list of the names the client is currently blocking.
func (cl *Client) BlockList () *Response {
	clist, err := cl.data.BlockList()
	if err != nil {
		log.Println("BlockList: ", err)
		return NewResponse(false, 50, "", nil)
	}
	sresp := "Block List:"
	for _, i := range clist {
		sresp = sresp + "\r\n" + i
	}
	return NewResponse(true, 0, sresp, clist)
}

//Unblock removes name from this clients block list.
func (cl *Client) Unblock(name string) *Response {
	if name == "" {
		return NewResponse(false, 22, "You must enter user to unblock.", nil)
	}
	if !clientdata.ValidateName(name) {
		return NewResponse(false, 20, "Invalid name.  Name must be alphanumeric characters only.", nil)
	}
	err := cl.data.Unblock(name)
	switch {
	case err == clientdata.ErrNotBlocking:
		return NewResponse(false, 31, fmt.Sprintf("You are not blocking %v.", name), nil)
	case err != nil:
		log.Println("Unblock: ", err)
		return NewResponse(false, 50, "", nil)
	default:
		return NewResponse(true, 0, fmt.Sprintf("No longer blocking %v.", name), nil)
	}
}

//Block adds name to this clients block list.
func (cl *Client) Block(name string) *Response {
	if name == "" {
		return NewResponse(false, 22, "You must enter a user to block.", nil)
	}
	if !clientdata.ValidateName(name) {
		return NewResponse(false, 20, "Invalid name.  Name must be alphanumeric characters only.", nil)
	}
	if cl.Name() == name {
		return NewResponse(false, 32, "You can't block yourself.", nil)
	}
	err := cl.data.Block(name)
	switch {
	case err == clientdata.ErrBlocking:
		return NewResponse(false, 30, fmt.Sprintf("You are already blocking %v.", name), nil)
	case err != nil:
		log.Println("Error Block: ", err)
		return NewResponse(false, 50, "", nil)
	default:
		return NewResponse(true, 0, fmt.Sprintf("Now blocking %v.", name), nil)
	}
}

//Friend adds name to this clients friend list.
func (cl *Client) Friend(name string) *Response {
	if name == "" {
		return NewResponse(false, 22, "You must enter a user to friend.", nil)
	}
	if !clientdata.ValidateName(name) {
		return NewResponse(false, 20, "Invalid name.  Name must be alphanumeric characters only.", nil)
	}
	if cl.Name() == name {
		return NewResponse(false, 37, "You can't friend yourself.", nil)
	}
	err := cl.data.Friend(name)
	switch {
	case err == clientdata.ErrFriend:
		return NewResponse(false, 35, fmt.Sprintf("%v is already on your friends list.", name), nil)
	case err != nil:
		log.Println("Error Friend: ", err)
		return NewResponse(false, 50, "", nil)
	default:
		return NewResponse(true, 0, fmt.Sprintf("%v is now on your friends list.", name), nil)
	}
}

//Unfriend removes name from this clients friend list.
func (cl *Client) Unfriend(name string) *Response {
	if name == "" {
		return NewResponse(false, 22, "You must enter a user to unfriend.", nil)
	}
	if !clientdata.ValidateName(name) {
		return NewResponse(false, 20, "Invalid name.  Name must be alphanumeric characters only.", nil)
	}
	err := cl.data.Unfriend(name)
	switch {
	case err == clientdata.ErrNotFriend:
		return NewResponse(false, 36, fmt.Sprintf("%v is not on your friends list.", name), nil)
	case err != nil:
		return NewResponse(false, 50, "", nil)
	default:
		return NewResponse(true, 0, fmt.Sprintf("%v is no longer on your friends list.", name), nil)		
	}
}

//Friend represents a person on your friends list.  room will instead be the last online string if they are not online now.
type Friend struct {
	Name string
	Room string
}

//FriendList gives a list of people on the clients friendlist and the room they are in or when they were last logged in.
func (cl *Client) FriendList() *Response {
	list, err := cl.data.FriendList()
	if err != nil {
		return NewResponse(false, 50, "", nil)
	}
	flist := make([]Friend, len(list), len(list))
	for i := range list {
		flist[i] = Friend{Name: list[i], Room: cl.rooms.FindClientRoom(list[i])}
		if flist[i].Room == "" {
			lo, err := cl.data.LastOnline(flist[i].Name)
			if err != nil && err != clientdata.ErrClientNotFound {
				log.Println("Friend List error: ", err)
			}
			if err == clientdata.ErrClientNotFound {
				flist[i].Room = "Not Found"
			} else {
				flist[i].Room = durationString(time.Since(lo))
			}
		}
	}
	sresp := "Friend \t\t Room/Last Online"
	for i := range flist {
		sresp = sresp + "\n\r" + flist[i].Name + "\t\t" + flist[i].Room
	}
	return NewResponse(true, 0, sresp, flist)
}

func (cl *Client) Tell(name, m string) *Response {
	if name == "" {
		return NewResponse(false, 42, "You must enter a name and a message.", nil)
	}
	other := cl.rooms.GetClient(name)
	if other != nil {
		if othc, ok := other.(*Client); ok {
			if othc.IsBlocked(cl.Name()) {
				return NewResponse(false, 43, fmt.Sprintf("%v is blocking you.", other.Name()), nil)
			}
		}
		mess := message.NewTellMessage(m, cl.Name(), other.Name(), true)
		other.Recieve(mess)
		sentMessage := message.NewTellMessage(m, cl.Name(), other.Name(), false)
		cl.Recieve(sentMessage)
		return NewResponse(true, 0, "", nil)	
	}
	return NewResponse(false, 42, "Could not find a client with that name.", nil)
}

//LeaveRoom removes the client from its room.
func (cl *Client) LeaveRoom() {
	if cl.room != nil {
		rm2 := cl.room
		_ = cl.room.Remove(cl)
		cl.room = nil
		rm2.Tell(fmt.Sprintf("%v leaves the room.", cl.Name()))
		cl.Recieve(message.NewServerMessage(fmt.Sprintf("%v leaves the room.", cl.Name())))
	}
}

//Leave is the client action to leave a room.  Moves the client back to the Lobby room.
func (cl *Client) Leave() *Response {
/*	if cl.room == nil {
		return NewResponse(false, 40, "You are not in a room.", nil)
	}
	cl.LeaveRoom()
	*/
	return cl.Join("Lobby")
}

func (cl *Client) Quit() *Response {
	cl.LeaveRoom()
	cl.connection.Close()
	return NewResponse(true, 0, "", nil)
}

//Send sends the message to the clients room.
func (cl *Client) Send(m string) *Response {
	message := message.NewSendMessage(m, cl.Name())
	if cl.room != nil {
		cl.log(fmt.Sprint(message))
		cl.room.Send(message)
		return NewResponse(true, 0, "", nil)
	} else {
		return NewResponse(false, 40, "You are not in a room.", nil)
	}
}

func durationString(d time.Duration) string {
	switch {
	case d > (time.Hour*24*365):
		return strconv.Itoa(int(d/(time.Hour*24*365))) + " Years ago"
	case d > (time.Hour*24*7):
		return strconv.Itoa(int(d/(time.Hour*24*7))) + " Weeks ago"
	case d > (time.Hour*24):
		return strconv.Itoa(int(d/(time.Hour*24))) + " Days ago"
	case d > time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + " Hours ago"
	case d > time.Minute:
		return strconv.Itoa(int(d/time.Minute)) + " Minutes ago"
	default:
		return strconv.Itoa(int(d/time.Second)) + " Seconds ago"						
	}
}

//Join adds a client to a room or creates a room if ti doesn't exist.
func (cl *Client) Join(rmName string) *Response {
	if rmName == "" {
		return NewResponse(false, 22, "You must enter a room to join.", nil)
	}
	if !clientdata.ValidateName(rmName) {
		return NewResponse(false, 20, "Invalid room name.  Name may only contain alphanumeric characters.", nil)
	}
	cl.LeaveRoom()
	rm := cl.rooms.FindRoom(rmName)
	if rm == nil {
		newRoom := room.NewRoom(rmName)
		cl.room = newRoom
		cl.room.Add(cl)
		cl.rooms.Add(cl.room)
	} else {
		cl.room = rm
		rm.Add(cl)
	}
	cl.room.Tell(fmt.Sprintf("%v has joined the room.", cl.Name()))
	return NewResponse(true, 0, "", nil)
}

//WhoData is an object used to return the advanced format in a response from who.
type WhoData struct {
	Room    string
	Clients []string
}

//Who sends the client a list of all the people in the room specified or the clients room if none is provided.
func (cl *Client) Who(rmName string) *Response {
	if rmName == "" && cl.room == nil {
		return NewResponse(false, 40, "You are not in a room.", nil)
	}
	if rmName == "" {
		rmName = cl.room.Name()
	}
	rm := cl.rooms.FindRoom(rmName)
	if rm == nil {
		return NewResponse(false, 41, "That room was not found.", nil)
	}
	clist := rm.Who()
	data := WhoData{Room: rmName, Clients: clist}
	sresp := fmt.Sprintf("Room: %v", rmName)
	for i := range clist {
		sresp = sresp + "\r\n" + clist[i]
	}
	return NewResponse(true, 0, sresp, data)
}

//List sends to the client a list of the current open rooms.
func (cl *Client) List() *Response {
	rlist := cl.rooms.Who()
	sresp := "Rooms:"
	for i := range rlist {
		sresp = sresp + "\r\n" + rlist[i]
	}
	return NewResponse(true, 0, sresp, rlist)
}

//log writes the string to the clients chatlog.
func (cl *Client) log(s string) {
	var err error
	_, err = io.WriteString(cl.chatlog, s+"\n")
	if err != nil {
		log.Println(err)
	}
}
