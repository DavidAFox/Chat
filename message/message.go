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

func (ml MessageList) PushBack(v Message) *list.Element {
	if ml.Len() > 99 {
		ml.Remove(ml.Front())
	}
	return ml.List.PushBack(v)
}

//newMessageList creates a new message list.
func NewMessageList() *MessageList {
	return &MessageList{list.New(), new(sync.Mutex)}
}

//serverMessage is a message containing only a string sent from the server.
type ServerMessage struct {
	Text string
	Type string
}

//NewServerMessage returns a new ServerMessage.
func NewServerMessage(text string) *ServerMessage {
	return &ServerMessage{Text: text, Type: "Server"}
}

//String returns the string representation of the serverMessage.
func (m ServerMessage) String() string {
	return m.Text
}

//SendMessage includes the text of the message, the time it was sent and the client who sent it.  It is used primarily for normal messages sent to the room with send.
type SendMessage struct {
	Text       string
	Time       time.Time
	TimeString string
	Sender     string
	Type       string
}

//String formats the clientMessage as time [Sender]: text.
func (m SendMessage) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v", m.Time.Format(layout), m.Sender, m.Text)
}

//Name returns the name of the client that send the message.
func (m SendMessage) Name() string {
	return m.Sender
}

//NewSendMessage creates a new client message
func NewSendMessage(text string, sender string) *SendMessage {
	msg := new(SendMessage)
	msg.Text = text
	msg.Time = time.Now()
	msg.TimeString = msg.Time.Format("3:04pm")
	msg.Sender = sender
	msg.Type = "Send"
	return msg
}

type JoinMessage struct {
	Subject string
	Text    string
	Type    string
}

func NewJoinMessage(subject string) *JoinMessage {
	msg := new(JoinMessage)
	msg.Subject = subject
	msg.Text = "has joined the room."
	msg.Type = "Join"
	return msg
}

func (m JoinMessage) String() string {
	return fmt.Sprintf("%v %v", m.Subject, m.Text)
}

func NewLeaveMessage(subject string) *JoinMessage {
	msg := new(JoinMessage)
	msg.Subject = subject
	msg.Text = "has left the room."
	msg.Type = "Join"
	return msg
}

//TellMessage is a message sent by a tell.
type TellMessage struct {
	Text       string
	TimeString string
	Time       time.Time
	Sender     string
	Reciever   string
	ToReciever bool
	Type       string
}

func NewTellMessage(text string, sender string, reciever string, toReciever bool) *TellMessage {
	msg := new(TellMessage)
	msg.Text = text
	msg.Time = time.Now()
	msg.TimeString = msg.Time.Format("3:04pm")
	msg.Type = "Tell"
	msg.Sender = sender
	msg.Reciever = reciever
	msg.ToReciever = toReciever
	return msg
}

func (m TellMessage) String() string {
	const layout = "3:04pm"
	if m.ToReciever {
		return fmt.Sprintf("%s [From %v]>>>: %v", m.Time.Format(layout), m.Sender, m.Text)
	} else {
		return fmt.Sprintf("%s <<<[To %v]: %v", m.Time.Format(layout), m.Reciever, m.Text)
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
