package message

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

//Message is an interface for dealing with various types of messages.
type Message interface {
	String() string
}

//messageList is a mutex enhanced linked list of messages.
type MessageList struct {
	*list.List
	*sync.Mutex
}

//newMessageList creates a new message list.
func NewMessageList() *MessageList {
	return &MessageList{list.New(), new(sync.Mutex)}
}

//serverMessage is a message containing only a string sent from the server.
type ServerMessage struct {
	text string
}

//NewServerMessage returns a new ServerMessage.
func NewServerMessage(text string) *ServerMessage {
	return &ServerMessage{text: text}
}

//String returns the string representation of the serverMessage.
func (m ServerMessage) String() string {
	return m.text
}

//clientMessage includes the text of the message, the time it was sent and the client who sent it.
type ClientMessage struct {
	text   string
	time   time.Time
	Sender string
}

//String formats the clientMessage as time [Sender]: text.
func (m ClientMessage) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v", m.time.Format(layout), m.Sender, m.text)
}

//newMessage creates a new client message
func NewClientMessage(t string, s string) *ClientMessage {
	msg := new(ClientMessage)
	msg.text = t
	msg.time = time.Now()
	msg.Sender = s
	return msg
}

//restMessage is a message sent from the REST API.
type RestMessage struct {
	Name string
	Text string
	Time time.Time
}

//String returns a rest message string formated as Time [Name]: Text.
func (m *RestMessage) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v", m.Time.Format(layout), m.Name, m.Text)
}
