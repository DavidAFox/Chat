package telnet

/*
Package telnet provides a connection implementation for use with the client package in the chat server.  It uses net.Conn to connect the client with the server. It appends each line with a \r\n so it will work with windows based telnet clients.
*/

import (
	"github.com/davidafox/chat/client"
	"github.com/davidafox/chat/clientdata"
	"github.com/davidafox/chat/message"
	"github.com/davidafox/chat/room"
	"io"
	"log"
	"net"
	"regexp"
	"strings"
)

//Connection is used to connect the user to the server.
type Connection struct {
	client *client.Client
	conn   net.Conn
}

//New creates a new connection and associated client.
func New(name string, roomlist *room.RoomList, chatlog io.Writer, data clientdata.ClientData, conn net.Conn) *Connection {
	c := new(Connection)
	c.conn = conn
	c.client = client.New(name, roomlist, chatlog, data, c)
	return c
}

//SendMessage is used by the client package to forward messages to the connection so they can be send to the user.  This version appends a \r\n and sends the message out the conn.
func (c *Connection) SendMessage(m message.Message) {
	_, err := io.WriteString(c.conn, m.String()+"\r\n")
	if err != nil {
		log.Println(err)
		c.client.Quit()
	}
}

//Close closes out the telnet connection.
func (c *Connection) Close() {
	c.conn.Close()
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
				if err == clientdata.ErrAccountCreationDisabled {
					_, err = io.WriteString(conn, "New account creation has been disabled.")
					return
				}
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
func TelnetLogin(conn net.Conn, rooms *room.RoomList, chl io.Writer, cd clientdata.ClientData) {
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
	c := New(name, rooms, chl, cd, conn)
	_, err = io.WriteString(conn, "Welcome\r\n")
	if err != nil {
		log.Println("Error Wrting: ", err)
	}
	go c.inputhandler()
}

//inputhandler processes command from telnet connections.
func (c *Connection) inputhandler() {
	for {
		input, err := readString(c.conn)
		if err != nil {
			log.Println("Error Reading", err)
			c.client.LeaveRoom()
			return
		}
		if strings.HasPrefix(input, "/") { // handle commands
			input = strings.TrimPrefix(input, "/")
		} else {
			input = "send " + input
		}
		cmd := strings.Fields(input)
		resp := c.client.Execute(cmd)
		if resp.String() != "" && cmd[0] != "quit" {
			_, err = io.WriteString(c.conn, resp.String()+"\r\n")
		}
		if cmd[0] == "quit" {
			c.client.LeaveRoom()
			return
		}
		if err != nil {
			log.Println(err)
		}
	}
}
