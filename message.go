package main

import (
	"time"
	"fmt"
)

//serverMessage is a message containing only a string sent from the server.
type serverMessage struct {
	text string
}

//String returns the string representation of the serverMessage.
func (m serverMessage) String() string {
	return m.text
}

//clientMessage includes the text of the message, the time it was sent and the client who sent it.
type clientMessage struct {
	text string
	time time.Time
	Sender Client
}

//String formats the clientMessage as time [Sender]: text.
func (m clientMessage) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v",m.time.Format(layout),m.Sender.Name(),m.text)
}

//newMessage creates a new client message
func newClientMessage(t string, s Client) *clientMessage {
	msg := new(clientMessage)
	msg.text = t
	msg.time = time.Now()
	msg.Sender = s
	return msg
}


//restMessage is a message sent from the REST API.
type restMessage struct {
	Name string
	Text string
	Time time.Time
}

//String returns a rest message string formated as Time [Name]: Text.
func (m *restMessage) String() string {
	const layout = "3:04pm"
	return fmt.Sprintf("%s [%v]: %v", m.Time.Format(layout),m.Name, m.Text)
}

