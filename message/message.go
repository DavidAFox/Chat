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

//ClientMessage is an interface for messages that can be blocked.
type ClientMessage interface {
	String() string
	Name() string //name of the client that sent the message
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

//SendMessage includes the text of the message, the time it was sent and the client who sent it.  It is used primarily for normal messages sent to the room with send.
type SendMessage struct {
	text   string
	time   time.Time
	Sender string
}

//String formats the clientMessage as time [Sender]: text.
func (m SendMessage) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v", m.time.Format(layout), m.Sender, m.text)
}

//Name returns the name of the client that send the message.
func (m SendMessage) Name() string {
	return m.Sender
}

//NewSendMessage creates a new client message
func NewSendMessage(t string, s string) *SendMessage {
	msg := new(SendMessage)
	msg.text = t
	msg.time = time.Now()
	msg.Sender = s
	return msg
}

//TellMessage is a message sent by a tell.
type TellMessage struct {
	text	string
	time	time.Time
	Sender	string
	Reciever string
	ToReciever	bool
}

func NewTellMessage(text string, sender string,reciever string, toReciever bool) *TellMessage {
	msg := new(TellMessage)
	msg.text = text
	msg.time = time.Now()
	msg.Sender = sender
	msg.Reciever = reciever
	msg.ToReciever = toReciever
	return msg
}

func (m TellMessage) String() string {
	const layout = "3:04pm"
	if m.ToReciever {
		return fmt.Sprintf("%s [From %v]>>>: %v",m.time.Format(layout), m.Sender, m.text)
	} else {
		return fmt.Sprintf("%s <<<[To %v]: %v",m.time.Format(layout), m.Reciever, m.text)
	}
}

func (m TellMessage) Name() string {
	return m.Sender
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
