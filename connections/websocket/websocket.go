package websocket

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/davidafox/chat/clientdata"
	"github.com/davidafox/chat/connections"
	"github.com/davidafox/chat/message"
	"github.com/davidafox/chat/room"
	"io"
	"log"
	"sync"
)

const TEXT_MESSAGE = 1
const USER_NAME_PWRD_DONT_MATCH = 21
const SERVER_ERROR = 50
const CLIENT_ALREADY_EXISTS = 10
const INVALID_NAME = 20

var ERR_NOT_LOGIN = errors.New("You are not logged in.")

type Connection struct {
	client    connections.Client
	socket    Socket
	writeLock *sync.Mutex
}

func New(client connections.Client, socket Socket) *Connection {
	c := new(Connection)
	c.socket = socket
	c.writeLock = new(sync.Mutex)
	c.client = client
	c.client.SetConnection(c)
	go c.inputHandler()
	return c
}

func NewWithNewClient(factory connections.ClientFactory, name string, socket Socket) *Connection {
	c := new(Connection)
	c.socket = socket
	c.writeLock = new(sync.Mutex)
	c.client = factory.New(name, c)
	go c.inputHandler()
	return c
}

func (con *Connection) Close() {
	con.client.LeaveRoom()
	con.socket.Close()
}

func (con *Connection) SendMessage(m message.Message) {
	con.writeLock.Lock()
	err := sendMessage(con.socket, &Message{Type: "Messages", Success: true, Code: 0, Data: []message.Message{m}})
	con.writeLock.Unlock()
	if err != nil {
		con.client.Execute([]string{"quit"})
	}
}

type Socket interface {
	Close() error
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, p []byte) (err error)
}

func Start(socket Socket, options *Options) {
	for {
		m, err := getInput(socket)
		if err != nil {
			socket.Close()
			return
		}
		cmd := parseCommand(m)
		switch cmd.Command {
		case "login":
			if Login(socket, options, cmd) {
				return
			}
		case "register":
			Register(socket, options, cmd)
			return
		case "quit":
			socket.Close()
			return
		}
	}
}

func Login(socket Socket, options *Options, cmd *Input) bool {
	if len(cmd.Args) < 2 {
		_ = sendMessage(socket, &Message{Type: "Login", Success: false, Code: USER_NAME_PWRD_DONT_MATCH, Data: "Must enter user name and password."})
		return false
	}
	name := cmd.Args[0]
	pwrd := cmd.Args[1]
	cd := options.DataFactory.Create(name)
	logged, err := cd.Authenticate(pwrd)
	if err != nil {
		log.Println(err)
		_ = sendMessage(socket, &Message{Type: "Login", Success: false, Code: SERVER_ERROR, Data: "Server error please try again."})
		return false
	}
	if !logged {
		_ = sendMessage(socket, &Message{Type: "Login", Success: false, Code: USER_NAME_PWRD_DONT_MATCH, Data: "User name and password do not match."})
		return false
	} else {

	}
	err = sendMessage(socket, &Message{Type: "Login", Success: true, Code: 0, Data: "Welcome"})
	if err != nil {
		log.Println(err)
		return false
	}
	NewWithNewClient(options.ClientFactory, name, socket)
	return true
}

func Register(socket Socket, options *Options, cmd *Input) {
	if len(cmd.Args) < 2 {
		_ = sendMessage(socket, &Message{Type: "Register", Success: false, Code: USER_NAME_PWRD_DONT_MATCH, Data: "Must enter user name and password."})
		return
	}
	name := cmd.Args[0]
	pwrd := cmd.Args[1]
	cd := options.DataFactory.Create(name)
	if !clientdata.ValidateName(name) {
		_ = sendMessage(socket, &Message{Type: "Register", Success: false, Code: INVALID_NAME, Data: "Invalid name.  Name can only contain alpha numeric characters."})
		return
	}
	err := cd.NewClient(pwrd)
	switch {
	case err == clientdata.ErrClientExists:
		_ = sendMessage(socket, &Message{Type: "Register", Success: false, Code: CLIENT_ALREADY_EXISTS, Data: "A client with that name already exists."})
		return
	case err == clientdata.ErrAccountCreationDisabled:
		_ = sendMessage(socket, &Message{Type: "Register", Success: false, Code: 0, Data: "Account creation has been disabled."})
	case err != nil:
		log.Println(err)
		_ = sendMessage(socket, &Message{Type: "Register", Success: false, Code: SERVER_ERROR, Data: "Server error please try again."})
		return
	default:
		_ = sendMessage(socket, &Message{Type: "Register", Success: true, Code: 0, Data: "Account Created"})
	}

}

func getInput(socket Socket) (string, error) {
	messageType, input, err := socket.ReadMessage()
	if err != nil {
		return "", err
	}
	if messageType != TEXT_MESSAGE {
		return "", errors.New("Incorrect Websocket Message Type: Expected Text")
	}
	bytes.TrimPrefix(input, []byte("\r\n"))
	return string(input), nil

}

func sendMessage(socket Socket, m *Message) error {
	j, err := json.Marshal(m)
	if err != nil {
		log.Printf("Error marshaling %v in sendMessage: %s", m, err)
		return err
	}
	err = socket.WriteMessage(TEXT_MESSAGE, j)
	if err != nil {
		return err
	}
	return nil
}

func getInputWithMessage(socket Socket, message string) (string, error) {
	err := socket.WriteMessage(TEXT_MESSAGE, []byte(message))
	if err != nil {
		return "", err
	}
	messageType, input, err := socket.ReadMessage()
	if err != nil {
		return "", err
	}
	if messageType != TEXT_MESSAGE {
		return "", errors.New("Incorrect Websocket Message Type: Expected Text")
	}
	bytes.TrimSuffix(input, []byte("\r\n"))
	return string(input), nil
}

func (c *Connection) getInput() (string, error) {
	return getInput(c.socket)
}

type Options struct {
	RoomList      *room.RoomList
	ChatLog       io.Writer
	DataFactory   clientdata.Factory
	ClientFactory connections.ClientFactory
}

func (c *Connection) inputHandler() {
	for {
		input, err := c.getInput()
		if err != nil {
			log.Println("Error in inputHandler(): ", err)
			c.Close()
			return
		}
		com := parseCommand(input)
		resp := c.HandleCommand(com)
		respStr := parseResponse(com.Command, resp)
		c.writeLock.Lock()
		err = c.socket.WriteMessage(TEXT_MESSAGE, []byte(respStr))
		c.writeLock.Unlock()
		if err != nil {
			log.Println("Error writing to websocket: ", err)
			c.Close()
			return
		}
	}
}

type Input struct {
	Command string
	Args    []string
}

func (c *Connection) HandleCommand(cmd *Input) connections.Response {
	CommandSlice := make([]string, 1, len(cmd.Args)+1)
	CommandSlice[0] = cmd.Command
	CommandSlice = append(CommandSlice, cmd.Args...)
	resp := c.client.Execute(CommandSlice)
	return resp
}

func parseCommand(cmd string) *Input {
	parsedCmd := new(Input)
	err := json.Unmarshal([]byte(cmd), parsedCmd)
	if err != nil {
		log.Println(err)
	}
	if parsedCmd.Args == nil {
		parsedCmd.Args = make([]string, 0, 0)
	}
	return parsedCmd
}

func parseResponse(t string, resp connections.Response) string {
	m := new(Message)
	m.Type = t
	m.Success = resp.Success()
	m.Code = resp.Code()
	m.String = resp.String()
	if m.Success {
		m.Data = resp.Data()
	} else {
		m.Data = resp.String()
	}
	j, err := json.Marshal(m)
	if err != nil {
		log.Println(err)
	}
	return string(j)
}

type Message struct {
	Type    string
	Success bool
	Code    int
	String  string
	Data    interface{}
}
